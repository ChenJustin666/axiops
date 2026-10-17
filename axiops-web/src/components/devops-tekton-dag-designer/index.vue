<template>
  <div class="tekton-dag-designer" @click="closeContextMenu">
    <section class="dag-canvas-panel">
      <div class="dag-toolbar">
        <div class="dag-title">
          <strong>{{ localConfig.pipeline.name || 'Tekton Pipeline' }}</strong>
          <span>{{ activeNodes.length }} 个节点 / {{ localConfig.edges.length }} 条依赖</span>
        </div>
        <div class="dag-actions">
          <ElButton @click="openPipelineConfigDialog()">Pipeline 配置</ElButton>
          <ElButton :disabled="readonly" type="primary" @click="openTaskLibraryDialog()">
            添加 Task
          </ElButton>
          <ElButton :disabled="readonly" @click="openTaskLibraryDialog('finally')">
            添加 Finally
          </ElButton>
          <ElBadge :value="issues.length" :hidden="issues.length === 0" class="dag-validate-badge">
            <ElButton
              :type="hasErrors ? 'danger' : issues.length ? 'warning' : 'default'"
              @click="validationDialogVisible = true"
            >
              校验
            </ElButton>
          </ElBadge>
          <ElButton :disabled="readonly" @click="applyAutoLayout()">自动布局</ElButton>
          <ElButton @click="fitView()">适应视图</ElButton>
        </div>
      </div>

      <div
        v-if="activeNode"
        class="dag-config-guide"
        :class="{ invalid: issueByTarget[activeNode.id] }"
      >
        <div class="dag-config-guide-main">
          <strong>{{ activeNodeGuideTitle }}</strong>
          <span>{{ activeNodeGuideDescription }}</span>
        </div>
        <div class="dag-config-guide-status">
          <span>{{ nodeRequiredParamSummary(activeNode) }}</span>
          <span>{{ nodeRequiredWorkspaceSummary(activeNode) }}</span>
        </div>
        <div class="dag-config-guide-actions">
          <ElButton size="small" type="primary" @click="openConfigDialog(activeNodeGuideTab)">
            去配置
          </ElButton>
          <ElButton size="small" @click="openConfigDialog('params')">参数</ElButton>
          <ElButton size="small" @click="openConfigDialog('workspaces')">Workspace</ElButton>
        </div>
      </div>

      <VueFlow
        ref="flowRef"
        :nodes="flowNodes"
        :edges="flowEdges"
        :nodes-draggable="!readonly"
        :nodes-connectable="!readonly"
        :elements-selectable="!readonly"
        :fit-view-on-init="false"
        class="dag-flow"
        @node-click="handleNodeClick"
        @node-context-menu="handleNodeContextMenu"
        @pane-click="handlePaneClick"
        @connect="handleConnect"
        @node-drag-stop="handleNodeDragStop"
      >
        <Controls :show-interactive="!readonly" />
        <MiniMap pannable zoomable />
        <template #node-tektonTask="{ data }">
          <div
            class="dag-node"
            :class="[
              `type-${data.node.type}`,
              { active: activeNodeId === data.node.id, invalid: issueByTarget[data.node.id] }
            ]"
            :style="taskNodeStyle(data.node)"
            @contextmenu.capture.prevent.stop="openNodeContextMenu($event, data.node.id)"
            @mousedown.right.stop
          >
            <button
              v-if="!readonly"
              type="button"
              class="dag-node-add dag-node-add-left"
              title="在前面插入 Task"
              @pointerdown.stop
              @click.stop="openTaskLibraryDialog('task', data.node.id, 'before')"
            >
              +
            </button>
            <div class="dag-node-head">
              <span class="dag-node-icon" :style="taskColorStyle(data.node.iconColor)">
                <ArtSvgIcon :icon="data.node.icon || 'ri:flow-chart'" />
              </span>
              <span class="dag-node-title">
                <strong :title="data.node.displayName || data.node.name">
                  {{ data.node.displayName || data.node.name }}
                </strong>
                <em>{{ nodeCardHint(data.node) }}</em>
              </span>
              <button
                type="button"
                class="dag-node-config"
                title="配置节点"
                @pointerdown.stop
                @click.stop="openNodeConfigFromMenu(data.node.id)"
              >
                配置
              </button>
            </div>
            <span v-if="issueByTarget[data.node.id]" class="dag-node-warning">
              {{ issueByTarget[data.node.id] }}
            </span>
            <div class="dag-node-foot">
              <span :class="{ done: !issueByTarget[data.node.id] }">
                {{ nodeConfigStatusText(data.node) }}
              </span>
              <span v-if="requiredNodeParamCount(data.node) > 0">
                参数 {{ configuredRequiredNodeParamCount(data.node) }}/{{
                  requiredNodeParamCount(data.node)
                }}
              </span>
              <span v-if="requiredNodeWorkspaceCount(data.node) > 0">
                Workspace {{ configuredRequiredNodeWorkspaceCount(data.node) }}/{{
                  requiredNodeWorkspaceCount(data.node)
                }}
              </span>
            </div>
            <button
              v-if="!readonly"
              type="button"
              class="dag-node-add dag-node-add-right"
              title="添加下一步"
              @pointerdown.stop
              @click.stop="openTaskLibraryDialog('task', data.node.id, 'after')"
            >
              +
            </button>
            <Handle type="target" :position="Position.Left" />
            <Handle type="source" :position="Position.Right" />
          </div>
        </template>
      </VueFlow>

      <div v-if="activeNodes.length === 0" class="dag-empty-state">
        <strong>从添加 Task 开始</strong>
        <span>主动选择 Task 后在弹窗里配置参数、Workspace、When 和 Matrix。</span>
        <ElButton type="primary" :disabled="readonly" @click="openTaskLibraryDialog()">
          添加 Task
        </ElButton>
      </div>

      <div
        v-if="contextMenu.visible && !readonly"
        class="dag-context-menu"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        @click.stop
      >
        <button type="button" @click="openTaskLibraryDialog('task', contextMenu.nodeId, 'after')">
          添加下一步
        </button>
        <button type="button" @click="openTaskLibraryDialog('task', contextMenu.nodeId, 'before')">
          在前面插入
        </button>
        <button
          type="button"
          @click="openTaskLibraryDialog('finally', contextMenu.nodeId, 'after')"
        >
          添加 Finally
        </button>
        <button type="button" @click="openNodeConfigFromMenu(contextMenu.nodeId)">配置节点</button>
        <button type="button" class="danger" @click="removeNode(contextMenu.nodeId)">
          删除节点
        </button>
      </div>
    </section>

    <ElDialog
      v-model="taskLibraryDialogVisible"
      :title="taskInsertMode === 'finally' ? '添加 Finally Task' : '添加 Task'"
      width="960px"
      align-center
      destroy-on-close
      :close-on-click-modal="false"
      class="tekton-dag-dialog tekton-task-library-dialog"
    >
      <div class="step-select-panel dialog-library">
        <div class="step-select-filter">
          <ElSelect
            v-model="taskCategoryFilter"
            placeholder="Task 分类"
            :clearable="false"
            filterable
            @change="handleTaskCategoryChange"
          >
            <ElOption
              v-for="item in taskCategoryOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
          <ElInput v-model="keyword" clearable placeholder="搜索 Task 名称或编码" />
          <ElButton :loading="props.libraryLoading" @click="refreshTaskLibrary">刷新</ElButton>
        </div>
        <div
          v-loading="props.libraryLoading"
          class="step-choice-grid"
          :class="{ 'is-empty': !props.libraryLoading && filteredLibrary.length === 0 }"
        >
          <ElEmpty
            v-if="!props.libraryLoading && filteredLibrary.length === 0"
            class="step-choice-empty"
            description="当前分类暂无可用 Task"
          />
          <button
            v-for="item in filteredLibrary"
            :key="item.id"
            type="button"
            class="step-choice-card"
            :disabled="readonly"
            @click="addTaskNode(item)"
          >
            <span class="library-icon" :style="{ color: taskLibraryIconColor(item) }">
              <ArtSvgIcon :icon="item.icon || tektonTaskDefaultIcon(item.source)" />
            </span>
            <span class="library-info">
              <strong :title="item.name">{{ item.name }}</strong>
              <em :title="item.code">{{ item.code }}</em>
              <small>{{ taskLibraryContractSummary(item) }}</small>
            </span>
          </button>
        </div>
      </div>
    </ElDialog>

    <ElDialog
      v-model="configDialogVisible"
      :title="activeNode ? '节点配置' : 'Pipeline 配置'"
      width="1080px"
      top="5vh"
      destroy-on-close
      :close-on-click-modal="false"
      class="tekton-dag-dialog"
      @close="closeConfigDialog"
    >
      <div class="dag-inspector dialog-inspector">
        <div class="panel-head">
          <strong>{{ activeNode ? '节点配置' : 'Pipeline 配置' }}</strong>
          <span>{{ activeNode?.name || localConfig.pipeline.name || '-' }}</span>
        </div>

        <ElForm v-if="!activeNode" label-position="top" class="inspector-form">
          <ElFormItem label="Pipeline 名称">
            <ElInput v-model="localConfig.pipeline.name" :disabled="readonly" @change="commit" />
          </ElFormItem>
          <ElFormItem label="展示名称">
            <ElInput
              v-model="localConfig.pipeline.displayName"
              :disabled="readonly"
              placeholder="可选，对应 Tekton spec.displayName"
              @change="commit"
            />
          </ElFormItem>
          <ElFormItem label="描述">
            <ElInput
              v-model="localConfig.pipeline.description"
              :disabled="readonly"
              type="textarea"
              :rows="2"
              placeholder="可选，对应 Tekton spec.description"
              @change="commit"
            />
          </ElFormItem>
          <ElFormItem label="Namespace">
            <ElInput
              v-model="localConfig.pipeline.namespace"
              :disabled="readonly"
              @change="commit"
            />
          </ElFormItem>
          <ElFormItem label="ServiceAccount">
            <ElSelect
              v-model="localConfig.pipeline.serviceAccountName"
              :disabled="readonly"
              clearable
              filterable
              allow-create
              default-first-option
              placeholder="默认使用构建渠道配置"
              style="width: 100%"
              @change="commit"
            >
              <ElOption
                v-for="item in serviceAccountOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="Pipeline Timeout">
            <ElInput v-model="localConfig.pipeline.timeout" :disabled="readonly" @change="commit" />
          </ElFormItem>
          <div class="inspector-section">
            <div class="section-title">
              <div class="section-heading">
                <strong>Pipeline 元数据</strong>
                <span>维护 Pipeline metadata.labels 和 metadata.annotations</span>
              </div>
              <div class="section-actions">
                <ElButton
                  size="small"
                  text
                  type="primary"
                  :disabled="readonly"
                  @click="addPipelineLabel"
                >
                  新增 Label
                </ElButton>
                <ElButton
                  size="small"
                  text
                  type="primary"
                  :disabled="readonly"
                  @click="addPipelineAnnotation"
                >
                  新增 Annotation
                </ElButton>
              </div>
            </div>
            <div class="pipeline-meta-list">
              <div
                v-for="(item, index) in localConfig.pipeline.labels"
                :key="`pipeline-label-${index}`"
                class="pipeline-meta-row"
              >
                <ElTag size="small" type="success">Label</ElTag>
                <ElInput
                  v-model="item.name"
                  :disabled="readonly"
                  placeholder="Key"
                  @change="commit"
                />
                <ElInput
                  v-model="item.value"
                  :disabled="readonly"
                  placeholder="Value"
                  @change="commit"
                />
                <ElButton
                  text
                  type="danger"
                  :disabled="readonly"
                  @click="removePipelineLabel(index)"
                >
                  删除
                </ElButton>
              </div>
              <div
                v-for="(item, index) in localConfig.pipeline.annotations"
                :key="`pipeline-annotation-${index}`"
                class="pipeline-meta-row"
              >
                <ElTag size="small" type="warning">Annotation</ElTag>
                <ElInput
                  v-model="item.name"
                  :disabled="readonly"
                  placeholder="Key"
                  @change="commit"
                />
                <ElInput
                  v-model="item.value"
                  :disabled="readonly"
                  placeholder="Value"
                  @change="commit"
                />
                <ElButton
                  text
                  type="danger"
                  :disabled="readonly"
                  @click="removePipelineAnnotation(index)"
                >
                  删除
                </ElButton>
              </div>
              <ElEmpty
                v-if="
                  !localConfig.pipeline.labels?.length && !localConfig.pipeline.annotations?.length
                "
                description="未配置额外元数据"
              />
            </div>
          </div>
          <div class="inspector-section">
            <div class="section-title">
              <div class="section-heading">
                <strong>Pipeline 参数汇总</strong>
                <span>{{ localConfig.params.length }} 个参数</span>
              </div>
            </div>
            <ElEmpty v-if="localConfig.params.length === 0" description="暂无 Pipeline 参数" />
            <div v-else class="pipeline-summary-list">
              <div v-for="param in localConfig.params" :key="param.name" class="pipeline-param-row">
                <div class="pipeline-summary-main">
                  <strong :title="param.displayName || param.name">
                    {{ param.displayName || param.name || '-' }}
                  </strong>
                  <span :title="pipelineParamUsageText(param.name)">
                    {{ param.name }} / {{ pipelineParamUsageText(param.name) || '未被节点引用' }}
                  </span>
                </div>
                <div class="pipeline-summary-tags">
                  <ElTag size="small" type="success">Tekton {{ param.type || 'string' }}</ElTag>
                  <ElTag v-if="param.paramType" size="small" type="info">
                    表单 {{ nodeParamTypeText(param.paramType) }}
                  </ElTag>
                  <ElTag v-if="param.required" size="small" type="danger">必填</ElTag>
                  <ElTag v-if="param.runtimeConfig" size="small" type="success">运行配置</ElTag>
                  <ElTag v-if="param.readonly" size="small" type="info">只读</ElTag>
                  <ElTag v-if="param.enum?.length" size="small">enum {{ param.enum.length }}</ElTag>
                  <ElTag v-if="param.properties" size="small" type="info">properties</ElTag>
                </div>
                <span class="pipeline-summary-desc" :title="param.description || ''">
                  {{ param.description || '无说明' }}
                </span>
              </div>
            </div>
          </div>
          <div class="inspector-section">
            <div class="section-title">
              <div class="section-heading">
                <strong>Pipeline Result</strong>
                <span>{{ localConfig.results?.length || 0 }} 个结果</span>
              </div>
              <div class="section-actions">
                <ElButton
                  size="small"
                  text
                  type="primary"
                  :disabled="readonly"
                  @click="addPipelineResult"
                >
                  新增
                </ElButton>
              </div>
            </div>
            <ElEmpty v-if="!localConfig.results?.length" description="暂无 Pipeline Result" />
            <div v-else class="pipeline-result-list">
              <div
                v-for="(result, index) in localConfig.results"
                :key="index"
                class="pipeline-result-row"
              >
                <ElInput
                  v-model="result.name"
                  :disabled="readonly"
                  placeholder="Result 名称"
                  @change="commit"
                />
                <ElSelect
                  v-model="result.type"
                  :disabled="readonly"
                  placeholder="类型"
                  @change="commit"
                >
                  <ElOption label="string" value="string" />
                  <ElOption label="array" value="array" />
                  <ElOption label="object" value="object" />
                </ElSelect>
                <ElSelect
                  v-model="result.value"
                  :disabled="readonly"
                  filterable
                  allow-create
                  default-first-option
                  placeholder="$(tasks.task.results.result)"
                  @change="commit"
                >
                  <ElOption
                    v-for="item in pipelineResultValueOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
                <ElInput
                  v-model="result.description"
                  :disabled="readonly"
                  placeholder="描述"
                  @change="commit"
                />
                <div class="result-actions">
                  <ElPopover
                    v-if="result.type === 'object'"
                    trigger="click"
                    width="360"
                    placement="left"
                  >
                    <template #reference>
                      <ElButton text type="primary" :disabled="readonly">属性</ElButton>
                    </template>
                    <ElInput
                      :model-value="pipelineResultPropertiesText(result)"
                      type="textarea"
                      :rows="6"
                      :disabled="readonly"
                      placeholder='例如 {"digest":{"type":"string"}}'
                      @change="
                        (value: string | number) =>
                          updatePipelineResultProperties(result, String(value ?? ''))
                      "
                    />
                  </ElPopover>
                  <ElButton
                    text
                    type="danger"
                    :disabled="readonly"
                    @click="removePipelineResult(index)"
                  >
                    删除
                  </ElButton>
                </div>
              </div>
            </div>
          </div>
          <div class="inspector-section">
            <div class="section-title">
              <div class="section-heading">
                <strong>Workspace 汇总</strong>
                <span>{{ localConfig.workspaces.length }} 个 Workspace</span>
              </div>
            </div>
            <ElEmpty v-if="localConfig.workspaces.length === 0" description="暂无 Workspace" />
            <div
              v-for="(workspace, index) in localConfig.workspaces"
              :key="workspace.name"
              class="workspace-item"
            >
              <div class="workspace-row">
                <div class="pipeline-summary-main">
                  <strong :title="workspace.name">{{ workspace.name || '-' }}</strong>
                  <span :title="pipelineWorkspaceUsageText(workspace.name)">
                    {{ pipelineWorkspaceUsageText(workspace.name) || '未被节点映射' }}
                  </span>
                </div>
                <div class="workspace-policy-tags">
                  <ElTag size="small" :type="workspace.optional ? 'info' : 'danger'">
                    {{ workspace.optional ? '可选' : '必填' }}
                  </ElTag>
                  <ElTag size="small">
                    {{ workspaceCapabilityText(workspace.bindingType) }}
                  </ElTag>
                  <ElTag v-if="workspace.runtimeConfig" size="small" type="success">
                    允许运行覆盖
                  </ElTag>
                </div>
                <span class="pipeline-summary-desc" :title="workspace.description || ''">
                  {{ workspace.description || workspaceRuntimeValueHint(workspace.bindingType) }}
                </span>
                <div class="workspace-card-actions">
                  <ElButton size="small" @click="openPipelineWorkspaceEditor(index)">
                    配置限制
                  </ElButton>
                </div>
              </div>
              <div class="workspace-limit-summary">
                <ElTag
                  v-for="item in workspaceAllowedBindingTypes(workspace)"
                  :key="item"
                  size="small"
                >
                  允许 {{ workspaceCapabilityText(item) }}
                </ElTag>
                <ElTag v-if="workspace.allowedSourceTypes?.length" size="small" type="success">
                  Secret 来源 {{ workspace.allowedSourceTypes.length }} 类
                </ElTag>
                <ElTag v-if="workspace.allowedPlatformTypes?.length" size="small" type="success">
                  平台资源 {{ workspace.allowedPlatformTypes.length }} 类
                </ElTag>
                <ElTag v-if="workspace.storage" size="small" type="info">
                  动态 PVC
                  {{
                    [
                      workspace.storage,
                      workspace.storageClassName,
                      workspace.accessMode,
                      workspace.volumeMode
                    ]
                      .filter(Boolean)
                      .join(' / ') || '10Gi'
                  }}
                </ElTag>
                <ElTag v-if="workspace.subPath" size="small" type="info">
                  subPath {{ workspace.subPath }}
                </ElTag>
              </div>
            </div>
          </div>

          <p class="config-note">
            Pipeline 配置只定义结构和平台能力限制；最终绑定值在运行配置中填写。
          </p>
        </ElForm>

        <ElForm v-else label-position="top" class="inspector-form node-inspector-form">
          <div class="node-config-hero">
            <div class="node-config-hero-main">
              <span class="node-config-eyebrow">当前 Task</span>
              <strong>{{ activeNode.displayName || activeNode.name }}</strong>
              <span>{{ nodeReferenceName(activeNode) }}</span>
            </div>
            <div class="node-config-hero-meta">
              <div>
                <span>Task Namespace</span>
                <strong>{{ taskRefNamespaceText(activeNode) }}</strong>
                <em>{{ taskNamespaceSourceText(activeNode) }}</em>
              </div>
              <div>
                <span>失败策略</span>
                <strong>{{ nodeFailurePolicyText(activeNode) }}</strong>
                <em>默认失败后终止 Pipeline</em>
              </div>
              <div>
                <span>运行约束</span>
                <strong>{{ activeNode.retries || 0 }} 次重试</strong>
                <em>{{ activeNode.timeout || '使用 Pipeline 超时' }}</em>
              </div>
            </div>
          </div>
          <ElTabs v-model="nodeConfigActiveTab" class="node-config-tabs">
            <ElTabPane label="基础配置" name="basic">
              <div class="inspector-section">
                <div class="section-title">
                  <strong>基础信息</strong>
                </div>
                <div class="form-grid two-columns">
                  <ElFormItem label="节点名称">
                    <ElInput v-model="activeNode.name" :disabled="readonly" @change="commit" />
                  </ElFormItem>
                  <ElFormItem label="展示名称">
                    <ElInput
                      v-model="activeNode.displayName"
                      :disabled="readonly"
                      placeholder="默认使用 Task 名称"
                      @change="commit"
                    />
                  </ElFormItem>
                  <ElFormItem label="描述">
                    <ElInput
                      v-model="activeNode.description"
                      :disabled="readonly"
                      placeholder="可选，对应 PipelineTask description"
                      @change="commit"
                    />
                  </ElFormItem>
                </div>
                <div class="readonly-grid">
                  <div class="readonly-item">
                    <span>引用类型</span>
                    <strong>{{ taskRefKindText(activeNode) }}</strong>
                  </div>
                  <div class="readonly-item">
                    <span>Task 引用</span>
                    <strong>{{ nodeReferenceName(activeNode) }}</strong>
                  </div>
                  <div class="readonly-item">
                    <span>API Version</span>
                    <strong>{{ activeNode.taskRef?.apiVersion || '-' }}</strong>
                  </div>
                  <div class="readonly-item">
                    <span>Task Namespace</span>
                    <strong>{{ taskRefNamespaceText(activeNode) }}</strong>
                  </div>
                  <div class="readonly-item">
                    <span>来源</span>
                    <strong>{{ activeNode.taskCode || activeNode.taskId || '-' }}</strong>
                  </div>
                </div>
                <p class="config-note">
                  Task 引用信息来自资源库选择结果，页面不直接手填，避免保存后和真实 Tekton
                  资源不一致。
                </p>
              </div>
            </ElTabPane>

            <ElTabPane label="参数映射" name="params">
              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong>参数映射</strong>
                    <span>
                      {{ activeNode.params.length }} 个参数 /
                      {{ requiredNodeParamCount(activeNode) }} 个必填
                    </span>
                  </div>
                  <div class="section-actions">
                    <ElCheckbox v-model="showRequiredNodeParamsOnly">只看必填</ElCheckbox>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="completeRequiredNodeParams"
                    >
                      补齐必填
                    </ElButton>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="keepRequiredNodeParams"
                    >
                      保留必填
                    </ElButton>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="addNodeParam"
                    >
                      新增
                    </ElButton>
                  </div>
                </div>
                <div class="node-param-list">
                  <ElEmpty
                    v-if="visibleNodeParamRows.length === 0"
                    description="暂无参数映射"
                    :image-size="72"
                  />
                  <div
                    v-for="{ param, index, requirement } in visibleNodeParamRows"
                    :key="index"
                    class="node-param-item"
                    :class="{ required: isRequiredNodeParam(param, requirement) }"
                  >
                    <div class="node-param-main">
                      <div class="node-param-title">
                        <i v-if="param.required">*</i>
                        <strong :title="param.displayName || param.name || 'Task 参数'">
                          {{ param.displayName || param.name || 'Task 参数' }}
                        </strong>
                        <span :title="param.sourceCode || param.name">
                          {{ param.sourceCode || param.name || '未选择参数' }}
                        </span>
                      </div>
                      <div class="node-param-binding-summary">
                        <span :title="nodeParamBindingText(param)">
                          {{ nodeParamBindingText(param) }}
                        </span>
                        <span
                          v-if="nodeParamDescriptionText(param, requirement)"
                          :title="nodeParamDescriptionText(param, requirement)"
                        >
                          {{ nodeParamDescriptionText(param, requirement) }}
                        </span>
                      </div>
                    </div>
                    <div class="node-param-head-tags">
                      <ElTag size="small" type="success">
                        Tekton {{ nodeParamOfficialTypeText(param, requirement) }}
                      </ElTag>
                      <ElTag size="small" type="info">
                        表单 {{ nodeParamEffectiveTypeText(param, requirement) }}
                      </ElTag>
                      <ElTag size="small">{{ nodeParamSourceText(param.source) }}</ElTag>
                      <ElTag
                        v-if="isRequiredNodeParam(param, requirement)"
                        size="small"
                        type="danger"
                      >
                        必填
                      </ElTag>
                      <ElTag v-if="param.runtimeConfig" size="small" type="warning">运行时</ElTag>
                      <ElTag v-if="param.readonly" size="small" type="info">只读</ElTag>
                      <ElTag v-if="nodeParamOptions(param, requirement).length" size="small">
                        选项 {{ nodeParamOptions(param, requirement).length }}
                      </ElTag>
                    </div>
                    <div class="node-param-actions">
                      <ElButton size="small" @click="openNodeParamEditor(index)">配置</ElButton>
                      <ElButton
                        size="small"
                        text
                        type="danger"
                        :disabled="
                          readonly || isRequiredNodeParam(param, requirement) || param.readonly
                        "
                        @click="removeNodeParam(index)"
                      >
                        删除
                      </ElButton>
                    </div>
                  </div>
                </div>
                <p class="config-note">
                  带星号的是 Task 必要映射或平台强约束参数，删除或缺省会阻止保存。
                </p>
              </div>
            </ElTabPane>

            <ElTabPane label="工作空间" name="workspaces">
              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong>Workspace 映射</strong>
                    <span>
                      {{ activeNode.workspaces.length }} 个映射 /
                      {{ requiredNodeWorkspaceCount(activeNode) }} 个必填
                    </span>
                  </div>
                  <div class="section-actions">
                    <ElCheckbox v-model="showRequiredWorkspacesOnly">只看必填</ElCheckbox>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="completeRequiredNodeWorkspaces"
                    >
                      补齐 Workspace
                    </ElButton>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="keepRequiredNodeWorkspaces"
                    >
                      保留必填
                    </ElButton>
                    <ElButton
                      size="small"
                      text
                      type="primary"
                      :disabled="readonly"
                      @click="addNodeWorkspace"
                    >
                      新增
                    </ElButton>
                  </div>
                </div>
                <div
                  v-for="{ workspace, index, requirement } in visibleWorkspaceRows"
                  :key="index"
                  class="workspace-map-item"
                  :class="{ required: isRequiredNodeWorkspace(workspace, requirement) }"
                >
                  <div class="workspace-map-row">
                    <ElSelect
                      v-model="workspace.taskWorkspace"
                      placeholder="Task workspace"
                      :disabled="readonly"
                      @change="handleNodeWorkspaceNameChange(workspace)"
                    >
                      <ElOption
                        v-for="item in activeNode.workspaceRequirements || []"
                        :key="item.name"
                        :label="`${item.name}${item.optional ? '（可选）' : '（必填）'}`"
                        :value="item.name"
                      />
                    </ElSelect>
                    <ElSelect
                      v-model="workspace.pipelineWorkspace"
                      placeholder="Pipeline workspace"
                      :disabled="readonly"
                      @change="handlePipelineWorkspaceSelect(workspace)"
                    >
                      <ElOption
                        v-for="item in localConfig.workspaces"
                        :key="item.name"
                        :label="item.name"
                        :value="item.name"
                      />
                    </ElSelect>
                    <label class="workspace-subpath-field">
                      <span>子路径</span>
                      <ElInput
                        v-model="workspace.subPath"
                        placeholder="例如 src"
                        :disabled="readonly"
                        clearable
                        @change="commit"
                      />
                    </label>
                    <ElButton
                      text
                      type="danger"
                      :disabled="readonly || isRequiredNodeWorkspace(workspace, requirement)"
                      @click="removeNodeWorkspace(index)"
                    >
                      删除
                    </ElButton>
                  </div>
                  <div v-if="workspace.pipelineWorkspace" class="workspace-policy-editor">
                    <ElFormItem label="是否必填">
                      <ElTag
                        :type="isRequiredNodeWorkspace(workspace, requirement) ? 'danger' : 'info'"
                      >
                        {{ isRequiredNodeWorkspace(workspace, requirement) ? '必填' : '可选' }}
                      </ElTag>
                    </ElFormItem>
                    <ElFormItem label="平台类型限制">
                      <ElTag size="small">
                        {{
                          workspaceCapabilityText(
                            pipelineWorkspaceBindingType(workspace.pipelineWorkspace)
                          )
                        }}
                      </ElTag>
                    </ElFormItem>
                    <ElFormItem label="运行配置">
                      <span class="workspace-runtime-hint">
                        {{
                          workspaceRuntimeValueHint(
                            pipelineWorkspaceBindingType(workspace.pipelineWorkspace)
                          )
                        }}
                      </span>
                    </ElFormItem>
                  </div>
                  <div class="workspace-binding-summary">
                    <ElTag
                      v-if="isRequiredNodeWorkspace(workspace, requirement)"
                      size="small"
                      type="danger"
                    >
                      必填 Workspace
                    </ElTag>
                    <ElTag v-if="workspace.requiredBindingType" size="small" type="warning">
                      要求 {{ workspaceCapabilityText(workspace.requiredBindingType) }}
                    </ElTag>
                    <ElTag v-if="workspace.bindingParamCode" size="small" type="info">
                      来源参数 {{ workspace.bindingParamCode }}
                    </ElTag>
                    <ElTag size="small">
                      类型
                      {{
                        workspaceCapabilityText(
                          pipelineWorkspaceBindingType(workspace.pipelineWorkspace)
                        )
                      }}
                    </ElTag>
                    <ElTag
                      v-if="
                        effectivePipelineWorkspaceBindingType(workspace.pipelineWorkspace) !==
                        pipelineWorkspaceBindingType(workspace.pipelineWorkspace)
                      "
                      size="small"
                      type="success"
                    >
                      运行生效
                      {{
                        workspaceCapabilityText(
                          effectivePipelineWorkspaceBindingType(workspace.pipelineWorkspace)
                        )
                      }}
                    </ElTag>
                    <span v-if="workspace.description">{{ workspace.description }}</span>
                  </div>
                </div>
                <p class="config-note">
                  Task workspace 只做映射、必填状态和平台能力限制；最终绑定值在运行配置中填写。
                </p>
              </div>
            </ElTabPane>

            <ElTabPane label="高级" name="advanced">
              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong>运行控制</strong>
                    <span>控制当前 Task 失败、重试和超时行为</span>
                  </div>
                </div>
                <div class="form-grid three-columns">
                  <ElFormItem>
                    <template #label>
                      <span class="field-label-with-help">
                        重试次数
                        <ElTooltip content="TaskRun 失败后自动重试的次数，0 表示不重试。">
                          <span class="help-dot">?</span>
                        </ElTooltip>
                      </span>
                    </template>
                    <ElInputNumber
                      v-model="activeNode.retries"
                      :min="0"
                      :disabled="readonly"
                      @change="commit"
                    />
                  </ElFormItem>
                  <ElFormItem>
                    <template #label>
                      <span class="field-label-with-help">
                        超时时间
                        <ElTooltip
                          content="当前 Task 的最长运行时间，例如 30m、1h。为空时使用 Pipeline 默认限制。"
                        >
                          <span class="help-dot">?</span>
                        </ElTooltip>
                      </span>
                    </template>
                    <ElInput v-model="activeNode.timeout" :disabled="readonly" @change="commit" />
                  </ElFormItem>
                  <ElFormItem>
                    <template #label>
                      <span class="field-label-with-help">
                        失败策略
                        <ElTooltip
                          content="默认失败后终止 Pipeline；只有明确需要收集后续信息时才选择继续。"
                        >
                          <span class="help-dot">?</span>
                        </ElTooltip>
                      </span>
                    </template>
                    <ElSelect v-model="activeNode.onError" :disabled="readonly" @change="commit">
                      <ElOption label="失败终止 Pipeline" value="stopAndFail" />
                      <ElOption label="失败后继续" value="continue" />
                    </ElSelect>
                  </ElFormItem>
                </div>
              </div>

              <div v-if="activeNode.taskRef?.resolver" class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong class="title-with-help">
                      Resolver 参数
                      <ElTooltip
                        content="Resolver 用来从远端仓库或集群动态解析 Task。普通任务通常不需要配置。"
                      >
                        <span class="help-dot">?</span>
                      </ElTooltip>
                    </strong>
                    <span>用于从远端或集群解析 Task 引用，普通 Task 通常不需要修改</span>
                  </div>
                  <ElButton
                    size="small"
                    text
                    type="primary"
                    :disabled="readonly"
                    @click="addResolverParam"
                  >
                    新增
                  </ElButton>
                </div>
                <div
                  v-for="(param, index) in activeNode.taskRef?.params"
                  :key="index"
                  class="inline-row"
                >
                  <ElInput
                    v-model="param.name"
                    placeholder="参数名"
                    :disabled="readonly"
                    @change="commit"
                  />
                  <ElInput
                    v-model="param.value"
                    placeholder="参数值"
                    :disabled="readonly"
                    @change="commit"
                  />
                  <ElButton
                    text
                    type="danger"
                    :disabled="readonly"
                    @click="removeResolverParam(index)"
                  >
                    删除
                  </ElButton>
                </div>
              </div>

              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong class="title-with-help">
                      依赖连线
                      <ElTooltip
                        content="runAfter 表示等待上游完成；Result 表示读取上游输出；When 表示满足条件才执行。"
                      >
                        <span class="help-dot">?</span>
                      </ElTooltip>
                    </strong>
                    <span>控制上游依赖、Result 传递和条件边</span>
                  </div>
                  <ElButton
                    size="small"
                    text
                    type="primary"
                    :disabled="readonly"
                    @click="addIncomingEdge"
                  >
                    新增
                  </ElButton>
                </div>
                <div v-for="edge in incomingEdges" :key="edge.id" class="edge-row">
                  <ElSelect
                    v-model="edge.source"
                    placeholder="上游 Task"
                    :disabled="readonly"
                    @change="commit"
                  >
                    <ElOption
                      v-for="item in resultSourceOptions"
                      :key="item.id"
                      :label="item.name"
                      :value="item.id"
                    />
                  </ElSelect>
                  <ElSelect
                    v-model="edge.type"
                    :disabled="readonly"
                    @change="handleEdgeTypeChange(edge)"
                  >
                    <ElOption label="顺序依赖 runAfter" value="runAfter" />
                    <ElOption label="结果传递 Result" value="result" />
                    <ElOption label="条件执行 When" value="when" />
                  </ElSelect>
                  <ElSelect
                    v-if="edge.type === 'result'"
                    v-model="edge.resultName"
                    filterable
                    allow-create
                    default-first-option
                    placeholder="Result 名称"
                    :disabled="readonly"
                    @change="commit"
                  >
                    <ElOption
                      v-for="item in resultNameOptions(edge.source)"
                      :key="item"
                      :label="item"
                      :value="item"
                    />
                  </ElSelect>
                  <ElSelect
                    v-if="edge.type === 'result'"
                    v-model="edge.targetParam"
                    filterable
                    allow-create
                    default-first-option
                    placeholder="目标参数"
                    :disabled="readonly"
                    @change="commit"
                  >
                    <ElOption
                      v-for="item in activeNode.params"
                      :key="item.name"
                      :label="item.name"
                      :value="item.name"
                    />
                  </ElSelect>
                  <ElInput
                    v-if="edge.type === 'when'"
                    v-model="edge.whenInput"
                    placeholder="条件表达式，例如 $(params.deploy)"
                    :disabled="readonly"
                    @change="commit"
                  />
                  <ElSelect
                    v-if="edge.type === 'when'"
                    v-model="edge.whenOperator"
                    :disabled="readonly"
                    @change="commit"
                  >
                    <ElOption label="in" value="in" />
                    <ElOption label="notin" value="notin" />
                  </ElSelect>
                  <ElInput
                    v-if="edge.type === 'when'"
                    :model-value="(edge.whenValues || []).join(',')"
                    placeholder="匹配值，逗号分隔"
                    :disabled="readonly"
                    @input="(value) => updateEdgeWhenValues(edge, String(value))"
                  />
                  <ElButton text type="danger" :disabled="readonly" @click="removeEdge(edge.id)">
                    删除
                  </ElButton>
                </div>
              </div>

              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong class="title-with-help">
                      When 条件
                      <ElTooltip
                        content="Tekton 的条件执行规则。只有表达式匹配时，当前 Task 才会运行。"
                      >
                        <span class="help-dot">?</span>
                      </ElTooltip>
                    </strong>
                    <span>例如参数等于指定值时才执行当前 Task</span>
                  </div>
                  <ElButton size="small" text type="primary" :disabled="readonly" @click="addWhen">
                    新增
                  </ElButton>
                </div>
                <div v-for="(when, index) in activeNode.when" :key="index" class="when-row">
                  <ElInput
                    v-model="when.cel"
                    placeholder="CEL 表达式，可选"
                    :disabled="readonly"
                    clearable
                    @change="commit"
                  />
                  <ElInput
                    v-model="when.input"
                    placeholder="$(params.deploy)"
                    :disabled="readonly || Boolean(when.cel)"
                    @change="commit"
                  />
                  <ElSelect
                    v-model="when.operator"
                    :disabled="readonly || Boolean(when.cel)"
                    @change="commit"
                  >
                    <ElOption label="in" value="in" />
                    <ElOption label="notin" value="notin" />
                  </ElSelect>
                  <ElInput
                    :model-value="(when.values || []).join(',')"
                    placeholder="true,false"
                    :disabled="readonly || Boolean(when.cel)"
                    @change="(value: string) => updateWhenValues(index, value)"
                  />
                  <ElButton text type="danger" :disabled="readonly" @click="removeWhen(index)">
                    删除
                  </ElButton>
                </div>
              </div>

              <div class="inspector-section">
                <div class="section-title">
                  <div class="section-heading">
                    <strong class="title-with-help">
                      Matrix 并行矩阵
                      <ElTooltip
                        content="Tekton Matrix 会按参数值组合生成多个并行 TaskRun。仅在需要同一 Task 针对多组参数并行运行时启用。"
                      >
                        <span class="help-dot">?</span>
                      </ElTooltip>
                    </strong>
                    <span>高级并行能力，默认关闭</span>
                  </div>
                  <ElButton
                    v-if="!activeNode.matrix"
                    size="small"
                    text
                    type="primary"
                    :disabled="readonly"
                    @click="enableMatrix"
                  >
                    启用
                  </ElButton>
                  <ElButton
                    v-else
                    size="small"
                    text
                    type="danger"
                    :disabled="readonly"
                    @click="disableMatrix"
                  >
                    关闭
                  </ElButton>
                </div>
                <template v-if="activeNode.matrix">
                  <div
                    v-for="(item, index) in activeNode.matrix.params"
                    :key="index"
                    class="matrix-row"
                  >
                    <ElInput
                      v-model="item.name"
                      placeholder="Matrix 参数名，例如 arch"
                      :disabled="readonly"
                      @change="commit"
                    />
                    <ElInput
                      :model-value="item.values.join(',')"
                      placeholder="取值列表，例如 amd64,arm64"
                      :disabled="readonly"
                      @change="(value: string) => updateMatrixValues(index, value)"
                    />
                    <ElButton
                      text
                      type="danger"
                      :disabled="readonly"
                      @click="removeMatrixParam(index)"
                    >
                      删除
                    </ElButton>
                  </div>
                  <ElButton size="small" :disabled="readonly" @click="addMatrixParam">
                    新增 Matrix 参数
                  </ElButton>
                  <div class="section-subtitle title-with-help">
                    Include 例外组合
                    <ElTooltip
                      content="Include 用来额外指定某些 Matrix 组合，适合少量特殊构建场景。"
                    >
                      <span class="help-dot">?</span>
                    </ElTooltip>
                  </div>
                  <div
                    v-for="(item, index) in activeNode.matrix.include"
                    :key="index"
                    class="matrix-include-row"
                  >
                    <ElInput
                      v-model="item.name"
                      placeholder="组合名称，可选，例如 linux-amd64"
                      :disabled="readonly"
                      @change="commit"
                    />
                    <ElInput
                      :model-value="formatNameValueList(item.params)"
                      placeholder="参数键值，例如 os=linux,arch=amd64"
                      :disabled="readonly"
                      @change="(value: string) => updateMatrixIncludeParams(index, value)"
                    />
                    <ElButton
                      text
                      type="danger"
                      :disabled="readonly"
                      @click="removeMatrixInclude(index)"
                    >
                      删除
                    </ElButton>
                  </div>
                  <ElButton size="small" :disabled="readonly" @click="addMatrixInclude">
                    新增 Include
                  </ElButton>
                </template>
              </div>
            </ElTabPane>
          </ElTabs>
        </ElForm>
      </div>
      <template #footer>
        <ElButton @click="closeConfigDialog">{{ readonly ? '关闭' : '取消' }}</ElButton>
        <ElButton v-if="!readonly" type="primary" @click="saveConfigDialog">保存</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="pipelineWorkspaceEditorVisible"
      title="Workspace 限制配置"
      width="860px"
      append-to-body
      class="tekton-dag-dialog workspace-editor-dialog"
      @close="handlePipelineWorkspaceEditorClosed"
    >
      <ElForm v-if="editingPipelineWorkspace" label-width="130px" class="workspace-editor-form">
        <div class="workspace-editor-title">
          <strong>{{ editingPipelineWorkspace.name }}</strong>
          <span>{{
            editingPipelineWorkspace.description || '配置项目流水线可选择的 Workspace 绑定方式'
          }}</span>
        </div>
        <div class="workspace-editor-grid">
          <ElFormItem label="默认绑定类型">
            <ElSelect
              v-model="editingPipelineWorkspace.bindingType"
              placeholder="默认绑定类型"
              :disabled="readonly"
              @change="handlePipelineWorkspaceBindingTypeChange(editingPipelineWorkspace)"
            >
              <ElOption
                v-for="item in workspaceBindingTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="允许绑定类型">
            <ElSelect
              v-model="editingPipelineWorkspace.allowedBindingTypes"
              multiple
              collapse-tags
              collapse-tags-tooltip
              placeholder="项目流水线可选的绑定类型"
              :disabled="readonly"
              @change="handleWorkspaceAllowedBindingTypesChange(editingPipelineWorkspace)"
            >
              <ElOption
                v-for="item in workspaceBindingTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="运行时覆盖">
            <ElSwitch
              v-model="editingPipelineWorkspace.runtimeConfig"
              :disabled="readonly"
              active-text="允许"
              inactive-text="不允许"
              @change="commit"
            />
          </ElFormItem>
          <ElFormItem
            v-if="workspaceAllowsGeneratedSecret(editingPipelineWorkspace)"
            label="Secret 来源"
          >
            <ElSelect
              v-model="editingPipelineWorkspace.allowedSourceTypes"
              multiple
              collapse-tags
              collapse-tags-tooltip
              :disabled="readonly"
              placeholder="允许后台生成 Secret 的来源"
              @change="handleWorkspaceAllowedSourceTypesChange(editingPipelineWorkspace)"
            >
              <ElOption
                v-for="item in workspaceSourceTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem
            v-if="workspaceAllowsGeneratedSecret(editingPipelineWorkspace)"
            label="平台资源类型"
            class="workspace-editor-wide"
          >
            <ElSelect
              v-model="editingPipelineWorkspace.allowedPlatformTypes"
              multiple
              filterable
              collapse-tags
              collapse-tags-tooltip
              :disabled="readonly"
              placeholder="允许选择的凭证或配置资源类型"
              @change="commit"
            >
              <ElOption
                v-for="item in workspacePlatformTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <template v-if="editingPipelineWorkspace.bindingType === 'volumeClaimTemplate'">
            <ElFormItem v-if="!isWorkspaceContractMode" label="StorageClass">
              <ElSelect
                v-model="editingPipelineWorkspace.storageClassName"
                clearable
                filterable
                :disabled="readonly"
                placeholder="StorageClass"
                @change="commit"
              >
                <ElOption
                  v-for="item in props.storageClassOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem v-if="!isWorkspaceContractMode" label="容量">
              <ElInput
                v-model="editingPipelineWorkspace.storage"
                :disabled="readonly"
                placeholder="例如 10Gi"
                @change="commit"
              />
            </ElFormItem>
            <ElFormItem v-if="!isWorkspaceContractMode" label="访问模式">
              <ElSelect
                v-model="editingPipelineWorkspace.accessMode"
                :disabled="readonly"
                placeholder="访问模式"
                @change="commit"
              >
                <ElOption
                  v-for="item in workspaceAccessModeOptions"
                  :key="item"
                  :label="item"
                  :value="item"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem v-if="!isWorkspaceContractMode" label="卷模式">
              <ElSelect
                v-model="editingPipelineWorkspace.volumeMode"
                clearable
                :disabled="readonly"
                placeholder="卷模式"
                @change="commit"
              >
                <ElOption
                  v-for="item in workspaceVolumeModeOptions"
                  :key="item"
                  :label="item"
                  :value="item"
                />
              </ElSelect>
            </ElFormItem>
          </template>
          <ElFormItem
            v-if="!isWorkspaceContractMode && editingPipelineWorkspace.bindingType !== 'emptyDir'"
            label="subPath"
            class="workspace-editor-wide"
          >
            <ElInput
              v-model="editingPipelineWorkspace.subPath"
              :disabled="readonly"
              placeholder="可选，挂载到 Workspace 的子路径"
              @change="commit"
            />
          </ElFormItem>
        </div>
      </ElForm>
      <template #footer>
        <ElButton @click="cancelPipelineWorkspaceEditor">{{ readonly ? '关闭' : '取消' }}</ElButton>
        <ElButton v-if="!readonly" type="primary" @click="savePipelineWorkspaceEditor">
          确定
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="nodeParamEditorVisible"
      title="编辑参数"
      width="980px"
      append-to-body
      class="tekton-dag-dialog node-param-editor-dialog"
      @close="handleNodeParamEditorClosed"
    >
      <ElForm
        v-if="editingNodeParam && activeNode"
        label-width="96px"
        class="node-param-editor-form"
      >
        <div class="node-param-editor-grid">
          <ElFormItem
            v-if="isPipelineParamBinding(editingNodeParam)"
            label="Pipeline 中文"
            required
          >
            <ElInput
              :model-value="
                pipelineParamContractValue(editingNodeParam, 'displayName') ||
                editingNodeParam.displayName ||
                editingNodeParam.name
              "
              placeholder="例如 仓库地址"
              :disabled="readonly"
              @input="
                (value: string) =>
                  updatePipelineParamContract(editingNodeParam, { displayName: value })
              "
            />
          </ElFormItem>
          <ElFormItem label="Pipeline 字段" required>
            <ElSelect
              v-if="isPipelineParamBinding(editingNodeParam)"
              v-model="editingNodeParam.pipelineParam"
              placeholder="选择或输入字段，例如 url"
              filterable
              allow-create
              default-first-option
              :disabled="readonly"
              @change="handlePipelineParamSelect(editingNodeParam)"
            >
              <ElOption
                v-for="item in pipelineFieldOptions()"
                :key="item.name"
                :label="pipelineParamOptionLabel(item)"
                :value="item.name"
              />
            </ElSelect>
            <ElInput
              v-else
              :model-value="editingNodeParam.name"
              disabled
              placeholder="先选择当前 Task 入参"
            />
          </ElFormItem>
          <ElFormItem label="映射 Task code" required>
            <ElSelect
              v-model="editingNodeParam.name"
              filterable
              allow-create
              default-first-option
              placeholder="选择当前 Task 入参"
              :disabled="readonly || editingNodeParam.readonly"
              @change="handleNodeParamNameChange(editingNodeParam)"
            >
              <ElOption
                v-for="item in activeNode.paramRequirements || []"
                :key="item.name"
                :label="taskParamOptionLabel(item)"
                :value="item.name"
              />
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="node-param-mode-row">
          <span>模式</span>
          <ElSegmented
            v-model="editingNodeParam.source"
            :options="nodeParamSourceOptions"
            :disabled="readonly || editingNodeParam.readonly"
            @change="handleParamSourceChange(editingNodeParam)"
          />
        </div>

        <div class="node-param-editor-grid">
          <ElFormItem
            v-if="editingNodeParam.source === 'fixed'"
            label="默认值"
            class="node-param-editor-wide"
          >
            <ElSelect
              v-if="nodeParamOptions(editingNodeParam, editingNodeParamRequirement).length"
              v-model="editingNodeParam.value"
              clearable
              filterable
              allow-create
              default-first-option
              :placeholder="editingNodeParam.defaultValue || '请选择参数值'"
              :disabled="readonly || editingNodeParam.readonly"
              @change="commit"
            >
              <ElOption
                v-for="option in nodeParamOptions(editingNodeParam, editingNodeParamRequirement)"
                :key="option.value"
                :label="option.label"
                :value="option.value"
              />
            </ElSelect>
            <ElSwitch
              v-else-if="editingNodeParam.paramType === 'booleanParam'"
              :model-value="editingNodeParam.value === 'true'"
              :disabled="readonly || editingNodeParam.readonly"
              @change="
                (value: string | number | boolean) =>
                  updateNodeParamFixedValue(editingNodeParam, value === true ? 'true' : 'false')
              "
            />
            <ElInputNumber
              v-else-if="editingNodeParam.paramType === 'number'"
              :model-value="nodeParamNumberValue(editingNodeParam)"
              controls-position="right"
              style="width: 100%"
              :disabled="readonly || editingNodeParam.readonly"
              @change="
                (value: number | undefined) => updateNodeParamFixedValue(editingNodeParam, value)
              "
            />
            <ElInput
              v-else-if="editingNodeParam.paramType === 'list'"
              v-model="editingNodeParam.value"
              type="textarea"
              :rows="3"
              resize="vertical"
              :placeholder="editingNodeParam.defaultValue || '请输入 YAML 数组或逗号分隔列表'"
              :disabled="readonly || editingNodeParam.readonly"
              @change="commit"
            />
            <ElInput
              v-else-if="editingNodeParam.paramType === 'objectList'"
              v-model="editingNodeParam.value"
              type="textarea"
              :rows="4"
              resize="vertical"
              :placeholder="editingNodeParam.defaultValue || '请输入 JSON 数组或对象'"
              :disabled="readonly || editingNodeParam.readonly"
              @change="commit"
            />
            <ElInput
              v-else
              v-model="editingNodeParam.value"
              :placeholder="editingNodeParam.defaultValue || '值/表达式'"
              :type="editingNodeParam.paramType === 'text' ? 'textarea' : 'text'"
              :rows="editingNodeParam.paramType === 'text' ? 4 : undefined"
              :disabled="readonly || editingNodeParam.readonly"
              @change="commit"
            />
          </ElFormItem>

          <template v-if="editingNodeParam.source === 'result'">
            <ElFormItem label="上游 Task" required>
              <ElSelect
                v-model="editingNodeParam.resultNodeId"
                placeholder="选择上游 Task"
                :disabled="readonly || editingNodeParam.readonly"
                @change="syncResultParamEdge(editingNodeParam)"
              >
                <ElOption
                  v-for="item in resultSourceOptions"
                  :key="item.id"
                  :label="item.name"
                  :value="item.id"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="Result 名称" required>
              <ElSelect
                v-model="editingNodeParam.resultName"
                filterable
                allow-create
                default-first-option
                placeholder="Result 名称"
                :disabled="readonly || editingNodeParam.readonly"
                @change="syncResultParamEdge(editingNodeParam)"
              >
                <ElOption
                  v-for="item in resultNameOptions(editingNodeParam.resultNodeId)"
                  :key="item"
                  :label="item"
                  :value="item"
                />
              </ElSelect>
            </ElFormItem>
          </template>
        </div>

        <div v-if="isPipelineParamBinding(editingNodeParam)" class="pipeline-param-contract">
          <div class="pipeline-param-contract-head">
            <div>
              <strong>Pipeline 参数契约</strong>
              <span>{{ pipelineParamContractSummary(editingNodeParam) }}</span>
            </div>
            <div class="pipeline-param-contract-tags">
              <ElTag size="small" type="info">
                表单 {{ nodeParamTypeText(pipelineParamFormType(editingNodeParam)) }}
              </ElTag>
              <ElTag size="small" type="success">
                Tekton {{ pipelineParamExpectedTektonType(editingNodeParam) }}
              </ElTag>
            </div>
          </div>
          <ElAlert
            v-if="pipelineParamTypeMismatchText(editingNodeParam)"
            :title="pipelineParamTypeMismatchText(editingNodeParam)"
            type="warning"
            :closable="false"
          />
          <div class="pipeline-param-contract-grid">
            <ElFormItem label="平台表单类型">
              <ElSelect
                :model-value="pipelineParamContractValue(editingNodeParam, 'paramType') || 'string'"
                filterable
                default-first-option
                :disabled="readonly"
                @change="
                  (value: string) =>
                    updatePipelineParamContract(editingNodeParam, { paramType: value })
                "
              >
                <ElOptionGroup
                  v-for="group in pipelinePlatformParamTypeGroups"
                  :key="group.label"
                  :label="group.label"
                >
                  <ElOption
                    v-for="item in group.options"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElOptionGroup>
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="默认值">
              <ElSwitch
                v-if="pipelineParamFormType(editingNodeParam) === 'booleanParam'"
                :model-value="
                  String(pipelineParamContractValue(editingNodeParam, 'defaultValue')) === 'true'
                "
                :disabled="readonly"
                @change="
                  (value: string | number | boolean) =>
                    updatePipelineParamDefaultValue(
                      editingNodeParam,
                      value === true ? 'true' : 'false'
                    )
                "
              />
              <ElInputNumber
                v-else-if="pipelineParamFormType(editingNodeParam) === 'number'"
                :model-value="pipelineParamNumberDefaultValue(editingNodeParam)"
                controls-position="right"
                style="width: 100%"
                :disabled="readonly"
                placeholder="请输入数字"
                @change="
                  (value: number | undefined) =>
                    updatePipelineParamDefaultValue(editingNodeParam, value)
                "
              />
              <ElInput
                v-else-if="pipelineParamFormType(editingNodeParam) === 'text'"
                :model-value="pipelineParamContractValue(editingNodeParam, 'defaultValue')"
                type="textarea"
                :rows="3"
                resize="vertical"
                :disabled="readonly"
                placeholder="请输入多行文本"
                @change="
                  (value: string | number) =>
                    updatePipelineParamDefaultValue(editingNodeParam, String(value ?? ''))
                "
              />
              <ElInput
                v-else-if="pipelineParamFormType(editingNodeParam) === 'list'"
                :model-value="pipelineParamContractValue(editingNodeParam, 'defaultValue')"
                type="textarea"
                :rows="3"
                resize="vertical"
                :disabled="readonly"
                placeholder="请输入 YAML 数组或逗号分隔列表"
                @change="
                  (value: string | number) =>
                    updatePipelineParamDefaultValue(editingNodeParam, String(value ?? ''))
                "
              />
              <ElInput
                v-else-if="pipelineParamFormType(editingNodeParam) === 'objectList'"
                :model-value="pipelineParamContractValue(editingNodeParam, 'defaultValue')"
                type="textarea"
                :rows="3"
                resize="vertical"
                :disabled="readonly"
                placeholder="请输入 YAML 对象"
                @change="
                  (value: string | number) =>
                    updatePipelineParamDefaultValue(editingNodeParam, String(value ?? ''))
                "
              />
              <ElInput
                v-else
                :model-value="pipelineParamContractValue(editingNodeParam, 'defaultValue')"
                :disabled="readonly"
                :placeholder="
                  pipelineParamFormType(editingNodeParam) === 'file'
                    ? '请输入文件路径或制品地址'
                    : '项目流水线默认值'
                "
                @change="
                  (value: string | number) =>
                    updatePipelineParamDefaultValue(editingNodeParam, String(value ?? ''))
                "
              />
            </ElFormItem>
            <ElFormItem label="Tekton 类型">
              <ElSelect
                :model-value="pipelineParamExpectedTektonType(editingNodeParam)"
                disabled
                @change="
                  (value: string) => updatePipelineParamContract(editingNodeParam, { type: value })
                "
              >
                <ElOption
                  v-for="item in pipelineParamTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
            <ElFormItem label="输出方式">
              <ElInput model-value="Tekton params" disabled />
            </ElFormItem>
          </div>
          <div class="pipeline-param-switch-grid">
            <ElFormItem label="强制配置" class="param-switch-field">
              <ElSwitch
                :model-value="pipelineParamBooleanValue(editingNodeParam, 'required')"
                :disabled="readonly"
                @change="
                  (value: string | number | boolean) =>
                    updatePipelineParamContract(editingNodeParam, { required: value === true })
                "
              />
            </ElFormItem>
            <ElFormItem label="不允许修改" class="param-switch-field">
              <ElSwitch
                :model-value="pipelineParamBooleanValue(editingNodeParam, 'readonly')"
                :disabled="readonly"
                @change="
                  (value: string | number | boolean) =>
                    updatePipelineParamContract(editingNodeParam, { readonly: value === true })
                "
              />
            </ElFormItem>
            <ElFormItem label="运行时覆盖" class="param-switch-field">
              <ElSwitch
                :model-value="pipelineParamBooleanValue(editingNodeParam, 'runtimeConfig')"
                :disabled="readonly"
                @change="
                  (value: string | number | boolean) =>
                    updatePipelineParamContract(editingNodeParam, { runtimeConfig: value === true })
                "
              />
            </ElFormItem>
            <ElFormItem label="占据整行" class="param-switch-field">
              <ElSwitch
                :model-value="pipelineParamConfigBooleanValue(editingNodeParam, 'fullRow')"
                :disabled="readonly"
                @change="
                  (value: string | number | boolean) =>
                    updatePipelineParamConfigField(editingNodeParam, 'fullRow', value === true)
                "
              />
            </ElFormItem>
          </div>
          <div
            v-if="pipelineParamRequiresPlatformConfig(editingNodeParam)"
            class="pipeline-param-platform"
          >
            <div class="pipeline-param-platform-head">
              <strong>平台配置</strong>
              <span>{{ pipelinePlatformConfigHint(editingNodeParam) }}</span>
            </div>
            <div
              v-if="pipelineParamFormType(editingNodeParam) === 'channelGroupCode'"
              class="pipeline-param-platform-grid"
            >
              <ElFormItem label="渠道分组" required>
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'channelGroupCode')"
                  filterable
                  placeholder="选择渠道分组"
                  :loading="platformOptionLoading"
                  :disabled="readonly"
                  @visible-change="handlePipelinePlatformOptionVisible"
                  @change="
                    (value: string) => handlePipelineChannelGroupChange(editingNodeParam, value)
                  "
                >
                  <ElOption
                    v-for="item in channelGroupOptions"
                    :key="item.code"
                    :label="`${item.name}（${item.code}）`"
                    :value="item.code"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="渠道类型" required>
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'channelTypeFilter')"
                  filterable
                  placeholder="选择渠道类型"
                  :disabled="
                    readonly || !pipelineParamConfigValue(editingNodeParam, 'channelGroupCode')
                  "
                  @visible-change="handlePipelinePlatformOptionVisible"
                  @change="
                    (value: string) => handlePipelineChannelTypeChange(editingNodeParam, value)
                  "
                >
                  <ElOption
                    v-for="item in pipelineChannelTypeOptions(editingNodeParam)"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="映射字段" required>
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'mappingField')"
                  filterable
                  placeholder="选择渠道字段"
                  :disabled="
                    readonly || !pipelineParamConfigValue(editingNodeParam, 'channelTypeFilter')
                  "
                  @visible-change="
                    (visible: boolean) =>
                      handlePipelineMappingFieldVisible(editingNodeParam, visible)
                  "
                  @change="
                    (value: string) =>
                      handlePipelineChannelMappingFieldChange(editingNodeParam, value)
                  "
                >
                  <ElOption
                    v-for="item in pipelineChannelMappingOptions(editingNodeParam)"
                    :key="item.field"
                    :label="`${item.name || item.field}（${item.field}）`"
                    :value="item.field"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem
                v-if="pipelineNeedChannelBindingParam(editingNodeParam)"
                :label="pipelineChannelBindingLabel(editingNodeParam)"
                :required="
                  currentPipelineMappingFieldOption(editingNodeParam)?.allowManualInput !== true
                "
              >
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'channelParamCode')"
                  filterable
                  :placeholder="`选择${pipelineChannelBindingLabel(editingNodeParam)}`"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'channelParamCode', value)
                  "
                >
                  <ElOption
                    v-for="item in pipelineChannelVariableOptions(editingNodeParam)"
                    :key="item.name"
                    :label="pipelineParamOptionLabel(item)"
                    :value="item.name"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem
                v-for="dep in pipelineChannelParamDependencies(editingNodeParam)"
                :key="dep.field"
                :label="dep.name || dep.field"
                :class="{
                  'wide-label-field': tektonCodeRepoGitRefProjectDependency(
                    editingNodeParam,
                    dep.field
                  )
                }"
              >
                <ElSelect
                  :model-value="pipelineDependencyParamCode(editingNodeParam, dep.field)"
                  filterable
                  placeholder="选择依赖参数"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      setPipelineDependencyParamCode(editingNodeParam, dep.field, value)
                  "
                >
                  <ElOption
                    v-for="item in pipelineChannelDependencyOptions(editingNodeParam, dep.field)"
                    :key="item.name"
                    :label="pipelineParamOptionLabel(item)"
                    :value="item.name"
                  />
                </ElSelect>
              </ElFormItem>
              <div v-if="pipelineMappingFieldMetaText(editingNodeParam)" class="platform-meta">
                {{ pipelineMappingFieldMetaText(editingNodeParam) }}
              </div>
            </div>
            <div
              v-else-if="pipelineParamFormType(editingNodeParam) === 'voucherModel'"
              class="pipeline-param-platform-grid"
            >
              <ElFormItem label="凭证类型" required>
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'voucherModel')"
                  placeholder="选择凭证类型"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'voucherModel', value)
                  "
                >
                  <ElOption
                    v-for="item in credentialTypeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="输出方式">
                <ElSelect
                  :model-value="pipelineCredentialMode(editingNodeParam)"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'credentialMode', value)
                  "
                >
                  <ElOption
                    v-for="item in credentialModeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem
                v-if="pipelineCredentialMode(editingNodeParam) === 'field'"
                label="凭证字段"
                required
              >
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'mappingField')"
                  placeholder="选择凭证字段"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'mappingField', value)
                  "
                >
                  <ElOption
                    v-for="item in credentialFieldOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </div>
            <div
              v-else-if="pipelineParamFormType(editingNodeParam) === 'channelCredential'"
              class="pipeline-param-platform-grid"
            >
              <ElFormItem label="渠道来源" required>
                <ElSelect
                  :model-value="
                    pipelineParamConfigValue(editingNodeParam, 'credentialSourceParamCode')
                  "
                  filterable
                  placeholder="选择渠道参数"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(
                        editingNodeParam,
                        'credentialSourceParamCode',
                        value
                      )
                  "
                >
                  <ElOption
                    v-for="item in pipelineCredentialSourceOptions(editingNodeParam)"
                    :key="item.name"
                    :label="pipelineParamOptionLabel(item)"
                    :value="item.name"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem label="输出方式">
                <ElSelect
                  :model-value="pipelineCredentialMode(editingNodeParam)"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'credentialMode', value)
                  "
                >
                  <ElOption
                    v-for="item in credentialModeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem
                v-if="pipelineCredentialMode(editingNodeParam) === 'field'"
                label="凭证字段"
                required
              >
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'mappingField')"
                  placeholder="选择凭证字段"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'mappingField', value)
                  "
                >
                  <ElOption
                    v-for="item in credentialFieldOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </div>
            <div
              v-else-if="pipelineParamFormType(editingNodeParam) === 'configCenter'"
              class="pipeline-param-platform-grid"
            >
              <ElFormItem label="配置类型 ID">
                <ElInput
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'configTypeId')"
                  :disabled="readonly"
                  placeholder="可选，配置中心类型 ID"
                  @change="
                    (value: string | number) =>
                      updatePipelineParamConfigField(
                        editingNodeParam,
                        'configTypeId',
                        String(value ?? '')
                      )
                  "
                />
              </ElFormItem>
              <ElFormItem label="配置类型编码">
                <ElInput
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'configTypeCode')"
                  :disabled="readonly"
                  placeholder="可选，配置中心类型编码"
                  @change="
                    (value: string | number) =>
                      updatePipelineParamConfigField(
                        editingNodeParam,
                        'configTypeCode',
                        String(value ?? '')
                      )
                  "
                />
              </ElFormItem>
              <ElFormItem label="输出方式">
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'valueMode') || 'string'"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'valueMode', value)
                  "
                >
                  <ElOption
                    v-for="item in configCenterValueModeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </div>
            <div
              v-else-if="pipelineParamFormType(editingNodeParam) === 'hostGroupTargets'"
              class="pipeline-param-platform-grid"
            >
              <ElFormItem label="输出模式" required>
                <ElSelect
                  :model-value="pipelineParamConfigValue(editingNodeParam, 'renderMode') || 'json'"
                  :disabled="readonly"
                  @change="
                    (value: string) =>
                      updatePipelineParamConfigField(editingNodeParam, 'renderMode', value)
                  "
                >
                  <ElOption
                    v-for="item in hostGroupRenderModeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </div>
          </div>
          <div v-if="pipelineParamNeedsOptions(editingNodeParam)" class="pipeline-param-options">
            <div class="pipeline-param-options-head">
              <span>选项列表</span>
              <ElButton
                text
                type="primary"
                :disabled="readonly"
                @click="addPipelineParamOption(editingNodeParam)"
              >
                新增选项
              </ElButton>
            </div>
            <div
              v-for="(option, optionIndex) in pipelineParamOptionRows(editingNodeParam)"
              :key="optionIndex"
              class="pipeline-param-option-row"
            >
              <ElInput
                :model-value="option.label"
                :disabled="readonly"
                placeholder="显示名称"
                @change="
                  (value: string | number) =>
                    updatePipelineParamOption(editingNodeParam, optionIndex, 'label', value)
                "
              />
              <ElInput
                :model-value="option.value"
                :disabled="readonly"
                placeholder="实际值"
                @change="
                  (value: string | number) =>
                    updatePipelineParamOption(editingNodeParam, optionIndex, 'value', value)
                "
              />
              <ElButton
                text
                type="danger"
                :disabled="readonly"
                @click="removePipelineParamOption(editingNodeParam, optionIndex)"
              >
                删除
              </ElButton>
            </div>
          </div>
          <ElFormItem
            v-if="pipelineParamContractValue(editingNodeParam, 'type') === 'object'"
            label="对象属性"
            class="node-param-description"
          >
            <ElInput
              :model-value="pipelineParamPropertiesText(editingNodeParam)"
              type="textarea"
              :rows="4"
              :disabled="readonly"
              placeholder='例如 {"url":{"type":"string"},"revision":{"type":"string"}}'
              @change="
                (value: string | number) =>
                  updatePipelineParamProperties(editingNodeParam, String(value ?? ''))
              "
            />
          </ElFormItem>
          <ElFormItem label="描述" class="node-param-description">
            <ElInput
              :model-value="pipelineParamContractValue(editingNodeParam, 'description')"
              type="textarea"
              :rows="3"
              :disabled="readonly"
              placeholder="请输入参数描述"
              @change="
                (value: string | number) =>
                  updatePipelineParamContract(editingNodeParam, {
                    description: String(value ?? '')
                  })
              "
            />
          </ElFormItem>
        </div>
      </ElForm>
      <template #footer>
        <ElButton @click="cancelNodeParamEditor">{{ readonly ? '关闭' : '取消' }}</ElButton>
        <ElButton v-if="!readonly" type="primary" @click="saveNodeParamEditor">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="validationDialogVisible"
      title="DAG 校验"
      width="720px"
      class="tekton-dag-dialog"
    >
      <div class="validation-dialog-body">
        <ElEmpty v-if="issues.length === 0" description="校验通过" />
        <div v-else class="issue-list">
          <div
            v-for="(issue, index) in issues"
            :key="index"
            class="issue-item"
            :class="issue.level"
          >
            <strong>{{ issue.level === 'error' ? '错误' : '警告' }}</strong>
            <span>{{ issue.message }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <ElButton @click="validationDialogVisible = false">关闭</ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue'
  import { Handle, Position, VueFlow, type Connection, type NodeDragEvent } from '@vue-flow/core'
  import { Controls } from '@vue-flow/controls'
  import { MiniMap } from '@vue-flow/minimap'
  import { parseDocument } from 'yaml'
  import { ElMessage, ElMessageBox, type Action } from 'element-plus'
  import { listDevopsChannelTypeApi, type DevopsChannelType } from '@/api/devops/channel'
  import {
    listDevopsChannelGroupOptionsApi,
    listDevopsChannelMappingFieldOptionsApi,
    type ChannelGroupOption,
    type ChannelMappingFieldDependency,
    type ChannelMappingFieldOption,
    type DevopsStepParam,
    type DevopsTektonTaskParam,
    type DevopsTektonTaskResult,
    type DevopsTektonWorkspace
  } from '@/api/devops/pipeline'
  import type {
    TektonDagConfig,
    TektonDagEdge,
    TektonDagLibraryItem,
    TektonDagNode,
    TektonDagPipelineParam,
    TektonDagPipelineResult,
    TektonDagParamRequirement,
    TektonDagTaskParam,
    TektonDagWorkspace,
    TektonDagWorkspaceMapping,
    TektonDagWorkspaceRequirement,
    TektonTaskCategoryOption,
    TektonNameValue,
    TektonResourceOption,
    TektonWorkspaceBindingType,
    TektonWorkspaceSourceType
  } from './types'
  import { layoutTektonDag } from './layout'
  import { validateTektonDagConfig } from './validate'
  import '@vue-flow/core/dist/style.css'
  import '@vue-flow/core/dist/theme-default.css'
  import '@vue-flow/controls/dist/style.css'
  import '@vue-flow/minimap/dist/style.css'

  const props = withDefaults(
    defineProps<{
      modelValue: TektonDagConfig
      libraryItems?: TektonDagLibraryItem[]
      categoryOptions?: TektonTaskCategoryOption[]
      categoryId?: string
      libraryLoading?: boolean
      serviceAccountOptions?: TektonResourceOption[]
      storageClassOptions?: TektonResourceOption[]
      secretOptions?: TektonResourceOption[]
      pvcOptions?: TektonResourceOption[]
      configMapOptions?: TektonResourceOption[]
      readonly?: boolean
      workspaceMode?: 'contract' | 'binding'
    }>(),
    {
      libraryItems: () => [],
      categoryOptions: () => [],
      categoryId: '',
      libraryLoading: false,
      serviceAccountOptions: () => [],
      storageClassOptions: () => [],
      secretOptions: () => [],
      pvcOptions: () => [],
      configMapOptions: () => [],
      readonly: false,
      workspaceMode: 'binding'
    }
  )

  const emit = defineEmits<{
    (event: 'update:modelValue', value: TektonDagConfig): void
    (event: 'update:categoryId', value: string): void
    (event: 'refresh-library', payload?: { categoryId?: string }): void
    (event: 'save'): void
  }>()

  const clone = <T,>(value: T): T => JSON.parse(JSON.stringify(value || {}))
  const localConfig = reactive<TektonDagConfig>(clone(props.modelValue))
  const activeNodeId = ref('')
  const taskLibraryDialogVisible = ref(false)
  const taskInsertMode = ref<'task' | 'finally'>('task')
  const configDialogVisible = ref(false)
  const validationDialogVisible = ref(false)
  const nodeConfigActiveTab = ref('basic')
  const keyword = ref('')
  const taskCategoryFilter = ref('')
  const taskInsertAnchorId = ref('')
  const taskInsertRelation = ref<'append' | 'before' | 'after'>('append')
  const showRequiredNodeParamsOnly = ref(false)
  const showRequiredWorkspacesOnly = ref(false)
  const nodeParamEditorVisible = ref(false)
  const editingNodeParamIndex = ref(-1)
  const nodeParamEditorSnapshot = ref<TektonDagConfig | null>(null)
  const pipelineWorkspaceEditorVisible = ref(false)
  const editingPipelineWorkspaceIndex = ref(-1)
  const pipelineWorkspaceEditorSnapshot = ref<TektonDagConfig | null>(null)
  const configDialogSnapshot = ref<TektonDagConfig | null>(null)
  const flowRef = ref<InstanceType<typeof VueFlow>>()
  const channelGroupOptions = ref<ChannelGroupOption[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const mappingFieldMap = ref<Record<string, ChannelMappingFieldOption[]>>({})
  const platformOptionLoading = ref(false)
  const isWorkspaceContractMode = computed(() => props.workspaceMode === 'contract')
  const contextMenu = ref({
    visible: false,
    x: 0,
    y: 0,
    nodeId: ''
  })

  function ensureOptionalConfig() {
    if (!Array.isArray(localConfig.results)) {
      localConfig.results = []
    }
    if (!localConfig.runPolicy) {
      localConfig.runPolicy = { workspaceStrategy: 'emptyDir', concurrencyPolicy: 'allow' }
    }
    localConfig.triggerConfig = {
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
      ...(localConfig.triggerConfig || {})
    }
    if (!localConfig.scheduleConfig) {
      localConfig.scheduleConfig = {
        enabled: false,
        timezone: 'Asia/Shanghai',
        concurrencyPolicy: 'skip',
        parameterSnapshot: true
      }
    }
    localConfig.prunerPolicy = {
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
      ...(localConfig.prunerPolicy || {})
    }
    ;(localConfig.nodes || []).forEach((node) => {
      if (!node.taskRef) {
        node.taskRef = { kind: 'Task', name: '' }
      }
    })
  }

  ensureOptionalConfig()

  watch(
    () => props.modelValue,
    (value) => {
      Object.assign(localConfig, clone(value))
      ensureOptionalConfig()
    },
    { deep: true }
  )

  watch(
    () => props.categoryId,
    (value) => {
      if (value && taskCategoryFilter.value !== value) taskCategoryFilter.value = value
    },
    { immediate: true }
  )

  const platformLibraryItems = computed(() => props.libraryItems)
  const filteredLibrary = computed(() => {
    const key = keyword.value.trim().toLowerCase()
    return platformLibraryItems.value.filter((item) =>
      [matchTaskCategory(item), matchTaskKeyword(item, key)].every(Boolean)
    )
  })
  const isRedundantTaskCategory = (item: TektonTaskCategoryOption) => {
    const label = String(item.label || '').replace(/（.*$/, '')
    return ['全部', '普通步骤', '人工审核'].includes(label)
  }
  const taskCategoryOptions = computed(() =>
    props.categoryOptions.filter((item) => !isRedundantTaskCategory(item))
  )
  const workspaceBindingTypeOptions: Array<{ label: string; value: TektonWorkspaceBindingType }> = [
    { label: 'emptyDir 临时目录', value: 'emptyDir' },
    { label: '已有 PVC', value: 'persistentVolumeClaim' },
    { label: '动态 PVC', value: 'volumeClaimTemplate' },
    { label: 'Secret', value: 'secret' },
    { label: '平台生成 Secret', value: 'generatedSecret' },
    { label: 'ConfigMap', value: 'configMap' },
    { label: 'Projected', value: 'projected' }
  ]
  const workspaceSourceTypeOptions: Array<{ label: string; value: TektonWorkspaceSourceType }> = [
    { label: '平台凭证', value: 'platformCredential' },
    { label: '配置中心', value: 'configCenter' }
  ]
  const workspacePlatformTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH Key', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: '证书', value: 'certificate' },
    { label: 'JSON', value: 'json' }
  ]
  const credentialTypeOptions = [
    { label: '用户名密码', value: 'username_password' },
    { label: 'Token', value: 'token' },
    { label: 'SSH Key', value: 'ssh_key' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secret_text' },
    { label: '证书', value: 'certificate' }
  ]
  const credentialModeOptions = [
    { label: '字段输出', value: 'field' },
    { label: 'JSON 输出', value: 'jsonData' },
    { label: 'JSON Base64', value: 'jsonBase64' }
  ]
  const credentialFieldOptions = [
    { label: '用户名', value: 'username' },
    { label: '密码', value: 'password' },
    { label: 'Token', value: 'token' },
    { label: '私钥', value: 'privateKey' },
    { label: '私钥口令', value: 'passphrase' },
    { label: 'Kubeconfig', value: 'kubeconfig' },
    { label: 'Secret Text', value: 'secretText' },
    { label: '证书', value: 'certificate' }
  ]
  const configCenterValueModeOptions = [
    { label: '字符串', value: 'string' },
    { label: '多行文本', value: 'text' },
    { label: 'Secret 文件', value: 'credentialsSecretFile' },
    { label: 'Base64', value: 'base64' }
  ]
  const hostGroupRenderModeOptions = [
    { label: 'JSON', value: 'json' },
    { label: 'JSON Base64', value: 'jsonBase64' },
    { label: 'Ansible', value: 'ansible' },
    { label: 'Ansible Base64', value: 'ansibleBase64' }
  ]
  const pipelineParamTypeOptions: Array<{
    label: string
    value: TektonDagPipelineParam['type']
  }> = [
    { label: 'string 字符串', value: 'string' },
    { label: 'array 数组', value: 'array' },
    { label: 'object 对象', value: 'object' }
  ]
  const nodeParamSourceOptions = [
    { label: '固定值', value: 'fixed' },
    { label: 'Pipeline 参数', value: 'pipelineParam' },
    { label: 'Task Result', value: 'result' }
  ]
  const pipelinePlatformParamTypeGroups = [
    {
      label: '基础限制',
      options: [
        { label: '字符', value: 'string' },
        { label: '多行文本', value: 'text' },
        { label: '布尔', value: 'booleanParam' },
        { label: '数字', value: 'number' },
        { label: '密码', value: 'password' },
        { label: '文件上传', value: 'file' },
        { label: '单选', value: 'choice' },
        { label: '对象列表', value: 'objectList' },
        { label: '列表', value: 'list' }
      ]
    },
    {
      label: '平台自定义',
      options: [
        { label: '渠道变量', value: 'channelGroupCode' },
        { label: '平台凭证', value: 'voucherModel' },
        { label: '渠道凭证', value: 'channelCredential' },
        { label: '配置中心', value: 'configCenter' },
        { label: '主机组目标', value: 'hostGroupTargets' }
      ]
    }
  ]
  const pipelinePlatformParamTypeValues = new Set(
    pipelinePlatformParamTypeGroups.flatMap((group) => group.options.map((item) => item.value))
  )
  const workspaceAccessModeOptions = ['ReadWriteOnce', 'ReadOnlyMany', 'ReadWriteMany']
  const workspaceVolumeModeOptions = ['Filesystem', 'Block']
  const activeNodes = computed(() => localConfig.nodes.filter((item) => item.enabled !== false))
  const activeNode = computed(() =>
    localConfig.nodes.find((item) => item.id === activeNodeId.value)
  )
  const editingNodeParam = computed(() => {
    const node = activeNode.value
    if (!node || editingNodeParamIndex.value < 0) return undefined
    return node.params[editingNodeParamIndex.value]
  })
  const editingNodeParamRequirement = computed(() => {
    const param = editingNodeParam.value
    if (!param) return undefined
    return activeNode.value?.paramRequirements?.find((item) => item.name === param.name)
  })
  const editingPipelineWorkspace = computed(() => {
    if (editingPipelineWorkspaceIndex.value < 0) return undefined
    return localConfig.workspaces[editingPipelineWorkspaceIndex.value]
  })
  const activeNodeGuideTab = computed(() =>
    activeNode.value ? nodeNextConfigTab(activeNode.value) : 'basic'
  )
  const activeNodeGuideTitle = computed(() => {
    const node = activeNode.value
    if (!node) return ''
    return `当前步骤：${node.displayName || node.name}`
  })
  const activeNodeGuideDescription = computed(() => {
    const node = activeNode.value
    if (!node) return ''
    return nodeGuideDescription(node)
  })
  const visibleNodeParamRows = computed(() => {
    const node = activeNode.value
    if (!node) return []
    return node.params
      .map((param, index) => ({
        param,
        index,
        requirement: node.paramRequirements?.find((item) => item.name === param.name)
      }))
      .filter(({ param, requirement }) => {
        if (!showRequiredNodeParamsOnly.value) return true
        return isRequiredNodeParam(param, requirement)
      })
  })
  const visibleWorkspaceRows = computed(() => {
    const node = activeNode.value
    if (!node) return []
    return node.workspaces
      .map((workspace, index) => ({
        workspace,
        index,
        requirement: node.workspaceRequirements?.find(
          (item) => item.name === workspace.taskWorkspace
        )
      }))
      .filter(({ workspace, requirement }) => {
        if (!showRequiredWorkspacesOnly.value) return true
        return isRequiredNodeWorkspace(workspace, requirement)
      })
  })
  const issues = computed(() => validateTektonDagConfig(localConfig))
  const hasErrors = computed(() => issues.value.some((item) => item.level === 'error'))
  const issueByTarget = computed(() =>
    issues.value.reduce<Record<string, string>>((acc, item) => {
      if (item.target && !acc[item.target]) acc[item.target] = item.message
      return acc
    }, {})
  )
  const resultSourceOptions = computed(() =>
    localConfig.nodes
      .filter(
        (node) => node.enabled !== false && node.type !== 'note' && node.id !== activeNodeId.value
      )
      .map((node) => ({
        id: node.id,
        name: node.displayName || node.name,
        results: node.results || []
      }))
  )
  const pipelineResultValueOptions = computed(() =>
    localConfig.nodes
      .filter((node) => node.enabled !== false && node.type !== 'note')
      .flatMap((node) => {
        const prefix = node.type === 'finally' ? 'finally' : 'tasks'
        return (node.results || []).map((result) => ({
          label: `${node.displayName || node.name} / ${result}`,
          value: `$(${prefix}.${node.name}.results.${result})`
        }))
      })
  )
  const incomingEdges = computed(() =>
    activeNode.value ? localConfig.edges.filter((edge) => edge.target === activeNode.value?.id) : []
  )
  const flowNodes = computed(() =>
    localConfig.nodes.map((node) => ({
      id: node.id,
      type: 'tektonTask',
      position: node.position,
      data: { node }
    }))
  )
  const flowEdges = computed(() =>
    localConfig.edges.map((edge) => ({
      id: edge.id,
      source: edge.source,
      target: edge.target,
      label: edge.type === 'result' ? `result:${edge.resultName || ''}` : edge.type,
      animated: edge.type === 'result',
      class: `edge-${edge.type}`
    }))
  )

  function commit() {
    emit('update:modelValue', clone(localConfig))
  }

  function closeConfigDialog() {
    if (configDialogSnapshot.value && !props.readonly) {
      Object.assign(localConfig, clone(configDialogSnapshot.value))
      ensureOptionalConfig()
      commit()
    }
    configDialogSnapshot.value = null
    configDialogVisible.value = false
  }

  function addPipelineResult() {
    if (!localConfig.results) localConfig.results = []
    localConfig.results.push({
      name: `result-${localConfig.results.length + 1}`,
      type: 'string',
      value: pipelineResultValueOptions.value[0]?.value || ''
    })
    commit()
  }

  function addPipelineLabel() {
    localConfig.pipeline.labels = [...(localConfig.pipeline.labels || []), { name: '', value: '' }]
    commit()
  }

  function removePipelineLabel(index: number) {
    localConfig.pipeline.labels = (localConfig.pipeline.labels || []).filter((_, i) => i !== index)
    commit()
  }

  function addPipelineAnnotation() {
    localConfig.pipeline.annotations = [
      ...(localConfig.pipeline.annotations || []),
      { name: '', value: '' }
    ]
    commit()
  }

  function removePipelineAnnotation(index: number) {
    localConfig.pipeline.annotations = (localConfig.pipeline.annotations || []).filter(
      (_, i) => i !== index
    )
    commit()
  }

  function removePipelineResult(index: number) {
    localConfig.results?.splice(index, 1)
    commit()
  }

  function pipelineResultPropertiesText(result: TektonDagPipelineResult) {
    return result.properties ? JSON.stringify(result.properties, null, 2) : ''
  }

  function updatePipelineResultProperties(result: TektonDagPipelineResult, value: string) {
    const text = String(value || '').trim()
    if (!text) {
      result.properties = undefined
      commit()
      return
    }
    const doc = parseDocument(text)
    if (doc.errors.length > 0) {
      ElMessage.warning('Result properties 必须是合法 JSON 或 YAML')
      return
    }
    const properties = normalizePipelineParamProperties(doc.toJSON())
    if (!properties) {
      ElMessage.warning('Result properties 必须是对象，例如 {"digest":{"type":"string"}}')
      return
    }
    result.properties = properties
    commit()
  }

  function saveConfigDialog() {
    syncActiveNodeToPipeline()
    configDialogSnapshot.value = null
    commit()
    configDialogVisible.value = false
  }

  function openPipelineWorkspaceEditor(index: number) {
    if (!localConfig.workspaces[index]) return
    editingPipelineWorkspaceIndex.value = index
    pipelineWorkspaceEditorSnapshot.value = clone(localConfig)
    pipelineWorkspaceEditorVisible.value = true
  }

  function restorePipelineWorkspaceEditorSnapshot() {
    if (!pipelineWorkspaceEditorSnapshot.value || props.readonly) return
    Object.assign(localConfig, clone(pipelineWorkspaceEditorSnapshot.value))
    ensureOptionalConfig()
    commit()
  }

  function handlePipelineWorkspaceEditorClosed() {
    restorePipelineWorkspaceEditorSnapshot()
    pipelineWorkspaceEditorSnapshot.value = null
    editingPipelineWorkspaceIndex.value = -1
  }

  function cancelPipelineWorkspaceEditor() {
    pipelineWorkspaceEditorVisible.value = false
  }

  function savePipelineWorkspaceEditor() {
    const workspace = editingPipelineWorkspace.value
    if (!workspace) return
    handlePipelineWorkspaceBindingTypeChange(workspace)
    pipelineWorkspaceEditorSnapshot.value = null
    commit()
    pipelineWorkspaceEditorVisible.value = false
  }

  function openNodeParamEditor(index: number) {
    const param = activeNode.value?.params[index]
    if (!param) return
    normalizeRuntimeParamSource(param)
    editingNodeParamIndex.value = index
    nodeParamEditorSnapshot.value = clone(localConfig)
    nodeParamEditorVisible.value = true
  }

  function restoreNodeParamEditorSnapshot() {
    if (!nodeParamEditorSnapshot.value || props.readonly) return
    Object.assign(localConfig, clone(nodeParamEditorSnapshot.value))
    ensureOptionalConfig()
    commit()
  }

  function handleNodeParamEditorClosed() {
    restoreNodeParamEditorSnapshot()
    nodeParamEditorSnapshot.value = null
    editingNodeParamIndex.value = -1
  }

  function cancelNodeParamEditor() {
    nodeParamEditorVisible.value = false
  }

  function saveNodeParamEditor() {
    const param = editingNodeParam.value
    if (!param) return
    normalizeRuntimeParamSource(param)
    if (!String(param.name || '').trim()) {
      ElMessage.warning('请选择 Task 参数')
      return
    }
    if (isPipelineParamBinding(param)) {
      if (!String(param.pipelineParam || '').trim()) {
        ElMessage.warning('请选择或输入 Pipeline 字段')
        return
      }
      handlePipelineParamSelect(param)
      syncPipelineParamOfficialType(param)
      if (!validatePipelineParamPlatformConfig(param)) return
    }
    if (param.source === 'result' && (!param.resultNodeId || !param.resultName)) {
      ElMessage.warning('请选择上游 Task 和 Result 名称')
      return
    }
    nodeParamEditorSnapshot.value = null
    commit()
    nodeParamEditorVisible.value = false
  }

  function syncActiveNodeToPipeline() {
    const node = activeNode.value
    if (!node) return
    dedupeNodeWorkspaceMappings(node)
    node.params.forEach((param) => {
      const requirement = node.paramRequirements?.find((item) => item.name === param.name)
      normalizeRuntimeParamSource(param)
      if (!isPipelineParamBinding(param)) return
      param.pipelineParam =
        param.pipelineParam ||
        ensurePipelineParam(param.name, paramMetaFromRequirement(requirement, param))
      applyPipelineParamMetaFromNodeParam(param, requirement)
    })
    node.workspaces.forEach((workspace) => {
      if (!workspace.pipelineWorkspace && workspace.taskWorkspace) {
        workspace.pipelineWorkspace = ensurePipelineWorkspace(
          workspace.taskWorkspace,
          workspace.optional,
          {
            description: workspace.description,
            bindingType: workspace.requiredBindingType || 'emptyDir'
          }
        )
      }
      enforceWorkspaceRequiredBinding(workspace)
    })
  }

  function pipelineParamUsageText(name?: string) {
    const paramName = String(name || '').trim()
    if (!paramName) return ''
    return localConfig.nodes
      .filter((node) => node.enabled !== false)
      .filter((node) =>
        node.params.some(
          (param) => isPipelineParamSource(param.source) && param.pipelineParam === paramName
        )
      )
      .map((node) => node.displayName || node.name)
      .join('、')
  }

  function pipelineWorkspaceUsageText(name?: string) {
    const workspaceName = String(name || '').trim()
    if (!workspaceName) return ''
    return localConfig.nodes
      .filter((node) => node.enabled !== false)
      .filter((node) =>
        node.workspaces.some((workspace) => workspace.pipelineWorkspace === workspaceName)
      )
      .map((node) => node.displayName || node.name)
      .join('、')
  }

  function workspaceRuntimeValueHint(type?: TektonWorkspaceBindingType) {
    const map: Record<TektonWorkspaceBindingType, string> = {
      emptyDir: '运行时无需填写绑定值',
      persistentVolumeClaim: '项目配置中选择 PVC，运行时可按开关覆盖',
      volumeClaimTemplate: '项目配置中定义动态 PVC 约束',
      secret: '项目配置中选择 Secret，运行时可按开关覆盖',
      generatedSecret: '项目配置中选择平台资源，由后台生成 Secret',
      configMap: '项目配置中选择 ConfigMap，运行时可按开关覆盖',
      projected: '项目配置中填写 projected sources'
    }
    return map[type || 'emptyDir']
  }

  function workspaceAllowedBindingTypes(workspace: TektonDagWorkspace) {
    const allowed = Array.isArray(workspace.allowedBindingTypes)
      ? workspace.allowedBindingTypes.filter(Boolean)
      : []
    return allowed.length ? allowed : [workspace.bindingType || 'emptyDir']
  }

  function workspaceAllowsGeneratedSecret(workspace: TektonDagWorkspace) {
    return workspaceAllowedBindingTypes(workspace).includes('generatedSecret')
  }

  function uniqueWorkspaceBindingTypes(items: TektonWorkspaceBindingType[]) {
    return Array.from(new Set(items.filter(Boolean)))
  }

  function uniqueWorkspaceSourceTypes(items: TektonWorkspaceSourceType[]) {
    return Array.from(new Set(items.filter(Boolean)))
  }

  function openTaskLibraryDialog(
    mode: 'task' | 'finally' = 'task',
    anchorId = '',
    relation: 'append' | 'before' | 'after' = 'append'
  ) {
    keyword.value = ''
    closeContextMenu()
    taskInsertMode.value = mode
    taskInsertAnchorId.value = anchorId
    taskInsertRelation.value = relation
    const optionValues = new Set(taskCategoryOptions.value.map((item) => item.value))
    const categoryId = optionValues.has(props.categoryId)
      ? props.categoryId
      : taskCategoryOptions.value[0]?.value || ''
    taskCategoryFilter.value = categoryId
    if (categoryId) {
      emit('update:categoryId', categoryId)
      emit('refresh-library', { categoryId })
    }
    taskLibraryDialogVisible.value = true
  }

  function openConfigDialog(preferredTab?: string) {
    closeContextMenu()
    if (activeNode.value) ensureNodeDefaults(activeNode.value)
    if (activeNode.value) dedupeNodeWorkspaceMappings(activeNode.value)
    nodeConfigActiveTab.value = activeNode.value ? preferredTab || 'basic' : 'pipeline'
    configDialogSnapshot.value = clone(localConfig)
    configDialogVisible.value = true
    nextTick(() => {
      document.querySelector<HTMLElement>('.dialog-inspector .inspector-form')?.scrollTo({ top: 0 })
    })
  }

  function openPipelineConfigDialog() {
    activeNodeId.value = ''
    openConfigDialog()
  }

  function ensureNodeDefaults(node: TektonDagNode) {
    if (!node.onError) node.onError = 'stopAndFail'
    if (node.retries === undefined) node.retries = 0
  }

  function addTaskNode(item: TektonDagLibraryItem) {
    const node = buildNodeFromLibrary(item, localConfig.nodes.length)
    if (taskInsertMode.value === 'finally') {
      node.type = 'finally'
      node.name = `finally-${node.name || item.code}`
    }
    insertNodeWithRelation(node)
    activeNodeId.value = node.id
    taskLibraryDialogVisible.value = false
    openConfigDialog('basic')
    commit()
  }

  function preferredNodeConfigTab(node: TektonDagNode) {
    if (configuredRequiredNodeParamCount(node) < requiredNodeParamCount(node)) return 'params'
    if (
      (node.workspaceRequirements || []).some((item) => !item.optional || item.requiredBindingType)
    ) {
      return 'workspaces'
    }
    if (node.params.length > 0) return 'params'
    if (node.workspaces.length > 0 || node.workspaceRequirements?.length) return 'workspaces'
    return 'basic'
  }

  function insertNodeWithRelation(node: TektonDagNode) {
    const anchorId = taskInsertAnchorId.value
    const relation = taskInsertRelation.value
    const anchorNode = localConfig.nodes.find((item) => item.id === anchorId)
    if (!anchorId || !anchorNode || relation === 'append' || node.type === 'finally') {
      localConfig.nodes.push(node)
      if (anchorId && anchorNode && node.type === 'finally') {
        localConfig.edges.push(createEdge(anchorId, node.id, 'finally'))
      }
      return
    }
    if (relation === 'after') {
      const outgoing = localConfig.edges.filter(
        (edge) => edge.source === anchorId && edge.type === 'runAfter'
      )
      outgoing.forEach((edge) => {
        edge.source = node.id
      })
      localConfig.edges.push(createEdge(anchorId, node.id, 'runAfter'))
    } else if (relation === 'before') {
      const incoming = localConfig.edges.filter(
        (edge) => edge.target === anchorId && edge.type === 'runAfter'
      )
      incoming.forEach((edge) => {
        edge.target = node.id
      })
      localConfig.edges.push(createEdge(node.id, anchorId, 'runAfter'))
    }
    node.position = {
      x: anchorNode.position.x + (relation === 'after' ? 320 : -320),
      y: anchorNode.position.y
    }
    localConfig.nodes.push(node)
  }

  function createEdge(source: string, target: string, type: TektonDagEdge['type']): TektonDagEdge {
    return {
      id: `edge-${source}-${target}-${Date.now()}-${Math.random().toString(16).slice(2, 6)}`,
      source,
      target,
      type
    }
  }

  function buildNodeFromLibrary(item: TektonDagLibraryItem, index: number): TektonDagNode {
    const parsed = parseTaskContent(item.stageContent)
    const taskParamContracts = libraryTaskParams(item)
    const taskWorkspaceContracts = libraryTaskWorkspaces(item)
    const taskResultContracts = libraryTaskResults(item)
    const parsedParams = mergeParsedTaskParams(parsed.params, taskParamContracts)
    const parsedWorkspaces = mergeParsedTaskWorkspaces(parsed.workspaces, taskWorkspaceContracts)
    const parsedResults = mergeParsedTaskResults(parsed.results, taskResultContracts)
    const taskKind =
      item.source === 'clusterTask' || parsed.kind === 'ClusterTask' ? 'ClusterTask' : 'Task'
    const taskName = parsed.name || item.code
    const platformParams = item.params || item.raw?.params || []
    const platformParamByCode = new Map(
      platformParams.map((param) => [String(param.code || '').trim(), param])
    )
    const workspaceRequirements = parsedWorkspaces.map((workspace) =>
      resolveWorkspaceRequirement(workspace, platformParams)
    )
    const nodeParams = parsedParams.map((taskParam) => {
      const platformParam = platformParamByCode.get(taskParam.name)
      const meta = mergeTaskParamMeta(taskParam, platformParam)
      const pipelineParam = ensurePipelineParam(taskParam.name, meta)
      return {
        name: taskParam.name,
        source: 'pipelineParam',
        pipelineParam,
        sourceCode: taskParam.name,
        displayName: meta.displayName,
        defaultValue: meta.defaultValue,
        description: meta.description,
        required: meta.required,
        readonly: meta.readonly,
        paramType: meta.paramType,
        options: meta.options,
        selectList: meta.selectList,
        allowCustom: meta.allowCustom,
        config: meta.config
      } as TektonDagTaskParam
    })
    const taskRef: TektonDagNode['taskRef'] = { kind: taskKind, name: taskName }
    if (taskKind === 'Task' && parsed.namespace) {
      taskRef.resolver = 'cluster'
      taskRef.namespace = parsed.namespace
      taskRef.params = [
        { name: 'kind', value: 'task' },
        { name: 'name', value: taskName },
        { name: 'namespace', value: parsed.namespace }
      ]
    } else if (parsed.namespace) {
      taskRef.namespace = parsed.namespace
    }
    return {
      id: `task-${Date.now()}-${index}`,
      type: 'task',
      name: taskName,
      displayName: item.name,
      taskId: item.id,
      taskCode: item.code,
      categoryId: item.categoryId,
      categoryName: item.categoryName,
      categoryCode: item.categoryCode,
      icon: item.icon || tektonTaskDefaultIcon(item.source),
      iconColor: item.iconColor || tektonTaskDefaultColor(item.source),
      taskRef,
      params: nodeParams,
      paramRequirements: parsedParams.map((param) => {
        const exposed = platformParamByCode.has(param.name)
        const meta = mergeTaskParamMeta(param, platformParamByCode.get(param.name))
        return {
          name: param.name,
          type: meta.type,
          displayName: meta.displayName,
          exposed,
          required: meta.required,
          defaultValue: meta.defaultValue,
          runtimeConfig: meta.runtimeConfig,
          sourceCode: meta.sourceCode,
          description: meta.description,
          readonly: meta.readonly,
          paramType: meta.paramType,
          options: meta.options,
          selectList: meta.selectList,
          allowCustom: meta.allowCustom,
          config: meta.config
        }
      }),
      workspaces: workspaceRequirements
        .filter((workspace) => workspace.pipelineWorkspace)
        .map((workspace) => ({
          taskWorkspace: workspace.name,
          pipelineWorkspace: workspace.pipelineWorkspace,
          optional: workspace.optional,
          description: workspace.description,
          requiredBindingType: workspace.requiredBindingType,
          bindingParamCode: workspace.bindingParamCode
        })),
      workspaceRequirements: workspaceRequirements.map((workspace) => ({
        name: workspace.name,
        optional: workspace.optional,
        description: workspace.description,
        requiredBindingType: workspace.requiredBindingType,
        bindingParamCode: workspace.bindingParamCode
      })),
      when: [],
      retries: 0,
      onError: 'stopAndFail',
      results: parsedResults,
      enabled: true,
      position: { x: 100 + index * 80, y: 150 + index * 40 },
      sourceTask: item.raw
    }
  }

  function parseTaskContent(content: string) {
    try {
      const doc = parseDocument(content || '')
      const data = doc.toJSON() as any
      const spec = data?.spec || {}
      return {
        name: String(data?.metadata?.name || ''),
        namespace: String(data?.metadata?.namespace || ''),
        kind: String(data?.kind || ''),
        params: Array.isArray(spec.params)
          ? spec.params
              .map((item: any) => ({
                name: String(item.name || ''),
                type: normalizeTektonParamType(item.type),
                hasDefault: Object.prototype.hasOwnProperty.call(item, 'default'),
                defaultValue:
                  item.default !== undefined
                    ? formatTaskDefaultValue(item.default)
                    : String(item.defaultValue || ''),
                description: String(item.description || ''),
                enum: Array.isArray(item.enum)
                  ? item.enum.map((value: any) => String(value || '')).filter(Boolean)
                  : [],
                properties: normalizePipelineParamProperties(item.properties)
              }))
              .filter((item: { name: string }) => item.name)
          : [],
        workspaces: Array.isArray(spec.workspaces)
          ? spec.workspaces
              .map((item: any) => ({
                name: String(item.name || ''),
                optional: item.optional === true,
                description: String(item.description || '')
              }))
              .filter((item: { name: string }) => item.name)
          : [],
        results: Array.isArray(spec.results)
          ? spec.results.map((item: any) => String(item.name || '')).filter(Boolean)
          : []
      }
    } catch {
      return { name: '', namespace: '', kind: '', params: [], workspaces: [], results: [] }
    }
  }

  type ParsedTaskParam = {
    name: string
    type?: 'string' | 'array' | 'object'
    defaultValue?: string
    hasDefault?: boolean
    description?: string
    enum?: string[]
    properties?: TektonDagPipelineParam['properties']
    required?: boolean
  }

  type ParsedTaskWorkspace = {
    name: string
    optional?: boolean
    description?: string
  }

  type PipelineParamMeta = {
    type?: 'string' | 'array' | 'object'
    paramType?: string
    defaultValue?: string
    description?: string
    required?: boolean
    runtimeConfig?: boolean
    readonly?: boolean
    options?: Array<{ label: string; value: string }>
    selectList?: Array<{ label: string; value: string }>
    enum?: string[]
    properties?: TektonDagPipelineParam['properties']
    allowCustom?: boolean
    config?: DevopsStepParam['config']
    sourceCode?: string
    displayName?: string
  }

  function libraryTaskParams(item: TektonDagLibraryItem): DevopsTektonTaskParam[] {
    return item.taskParams || item.raw?.taskParams || []
  }

  function libraryTaskResults(item: TektonDagLibraryItem): DevopsTektonTaskResult[] {
    return item.taskResults || item.raw?.taskResults || []
  }

  function libraryTaskWorkspaces(item: TektonDagLibraryItem): DevopsTektonWorkspace[] {
    return item.taskWorkspaces || item.raw?.taskWorkspaces || []
  }

  function mergeParsedTaskParams(
    parsedParams: ParsedTaskParam[],
    contracts: DevopsTektonTaskParam[]
  ): ParsedTaskParam[] {
    const merged = new Map<string, ParsedTaskParam>()
    parsedParams.forEach((item) => {
      if (item.name) merged.set(item.name, { ...item })
    })
    contracts.forEach((item) => {
      const name = String(item.name || '').trim()
      if (!name) return
      const base = merged.get(name) || { name }
      const hasContractDefault = Object.prototype.hasOwnProperty.call(item, 'defaultValue')
      merged.set(name, {
        ...base,
        name,
        type: normalizeTektonParamType(item.type || base.type),
        defaultValue: hasContractDefault ? item.defaultValue || '' : base.defaultValue || '',
        hasDefault: hasContractDefault || base.hasDefault,
        description: item.description || base.description || '',
        enum: item.enum?.length ? item.enum : base.enum,
        properties: item.properties || base.properties,
        required: item.required
      })
    })
    return Array.from(merged.values())
  }

  function mergeParsedTaskWorkspaces(
    parsedWorkspaces: ParsedTaskWorkspace[],
    contracts: DevopsTektonWorkspace[]
  ): ParsedTaskWorkspace[] {
    const merged = new Map<string, ParsedTaskWorkspace>()
    parsedWorkspaces.forEach((item) => {
      if (item.name) merged.set(item.name, { ...item })
    })
    contracts.forEach((item) => {
      const name = String(item.name || '').trim()
      if (!name) return
      const base = merged.get(name) || { name }
      merged.set(name, {
        ...base,
        name,
        optional: item.optional ?? base.optional,
        description: item.description || base.description || ''
      })
    })
    return Array.from(merged.values())
  }

  function mergeParsedTaskResults(
    parsedResults: string[],
    contracts: DevopsTektonTaskResult[]
  ): string[] {
    const names = new Set(parsedResults.filter(Boolean))
    contracts.forEach((item) => {
      const name = String(item.name || '').trim()
      if (name) names.add(name)
    })
    return Array.from(names)
  }

  function platformParamTypeFromTektonType(type?: string) {
    if (type === 'array') return 'list'
    if (type === 'object') return 'objectList'
    return 'string'
  }

  function mergeTaskParamMeta(
    taskParam: ParsedTaskParam,
    platformParam?: DevopsStepParam
  ): PipelineParamMeta {
    const options = platformParam?.selectList?.length
      ? platformParam.selectList
      : platformParam?.config?.options || []
    return {
      type: taskParam.type || 'string',
      paramType: normalizePipelinePlatformParamType(
        platformParam?.paramType || platformParamTypeFromTektonType(taskParam.type)
      ),
      defaultValue: platformParam?.defaultValue ?? taskParam.defaultValue ?? '',
      description: platformParam?.description || taskParam.description || '',
      required:
        platformParam?.required !== undefined
          ? platformParam.required
          : taskParam.required !== undefined
            ? taskParam.required
            : !(taskParam.hasDefault === true || String(taskParam.defaultValue || '').trim()),
      runtimeConfig: platformParam?.runtimeConfig === true,
      readonly: platformParam?.readonly === true,
      options,
      selectList: platformParam?.selectList || options,
      enum: taskParam.enum,
      properties: taskParam.properties,
      allowCustom: platformParam?.allowCustom,
      config: platformParam?.config ? clone(platformParam.config) : undefined,
      sourceCode: platformParam?.code || taskParam.name,
      displayName: platformParam?.name || taskParam.description || taskParam.name
    }
  }

  function workspaceBindingKind(param: DevopsStepParam): TektonWorkspaceBindingType | '' {
    if (['workspaceSecret', 'kubernetesSecretName'].includes(param.paramType)) return 'secret'
    if (['workspacePvc', 'kubernetesPvcName'].includes(param.paramType)) {
      return 'persistentVolumeClaim'
    }
    if (String(param.config?.provider || '').startsWith('volumeClaimTemplate.')) {
      return 'volumeClaimTemplate'
    }
    return ''
  }

  function workspaceBindingParams(workspaceName: string, platformParams: DevopsStepParam[]) {
    return platformParams.filter(
      (param) => String(param.config?.mappingField || '').trim() === workspaceName
    )
  }

  function resolveWorkspaceRequirement(
    workspace: ParsedTaskWorkspace,
    platformParams: DevopsStepParam[]
  ) {
    const bindingParams = workspaceBindingParams(workspace.name, platformParams)
    const bindingParam = bindingParams.find((param) => workspaceBindingKind(param))
    const requiredBindingType = bindingParam ? workspaceBindingKind(bindingParam) : undefined
    const shouldMount = !workspace.optional || Boolean(requiredBindingType)
    const pipelineWorkspace = shouldMount
      ? ensurePipelineWorkspace(workspace.name, workspace.optional, {
          description: workspace.description,
          bindingType: requiredBindingType || 'emptyDir',
          bindingParams
        })
      : ''
    return {
      name: workspace.name,
      optional: workspace.optional,
      description: workspace.description,
      requiredBindingType,
      bindingParamCode: bindingParam?.code,
      pipelineWorkspace
    }
  }

  function applyWorkspaceBindingParam(
    workspaceName: string,
    workspaceType: TektonWorkspaceBindingType
  ) {
    const workspace = localConfig.workspaces.find((item) => item.name === workspaceName)
    if (!workspace) return
    if (workspaceType === 'secret') workspace.bindingType = 'secret'
    if (workspaceType === 'persistentVolumeClaim') workspace.bindingType = 'persistentVolumeClaim'
    workspace.allowedBindingTypes = uniqueWorkspaceBindingTypes([
      ...(workspace.allowedBindingTypes || []),
      workspaceType
    ])
  }

  function applyDynamicPvcBindingParam(workspaceName: string, param: DevopsStepParam) {
    const field = String(param.config?.provider || '').replace('volumeClaimTemplate.', '')
    if (!field) return
    const meta = mergeTaskParamMeta(
      {
        name: param.code,
        type: 'string',
        defaultValue: param.defaultValue || '',
        description: param.description || ''
      },
      param
    )
    const pipelineParam = ensurePipelineParam(param.code, meta)
    const workspace = localConfig.workspaces.find((item) => item.name === workspaceName)
    if (!workspace) return
    workspace.bindingType = 'volumeClaimTemplate'
    workspace.allowedBindingTypes = uniqueWorkspaceBindingTypes([
      ...(workspace.allowedBindingTypes || []),
      'volumeClaimTemplate'
    ])
    const ref = `$(params.${pipelineParam})`
    if (field === 'storage') workspace.storage = workspace.storage || ref
    if (field === 'storageClassName') workspace.storageClassName = workspace.storageClassName || ref
    if (field === 'accessMode') workspace.accessMode = workspace.accessMode || ref
    if (field === 'volumeMode') workspace.volumeMode = workspace.volumeMode || ref
  }

  function matchTaskCategory(item: TektonDagLibraryItem) {
    if (!taskCategoryFilter.value) return true
    return [item.categoryId, item.categoryCode].includes(taskCategoryFilter.value)
  }

  function handleTaskCategoryChange(value: string) {
    taskCategoryFilter.value = value
    emit('update:categoryId', value)
    emit('refresh-library', { categoryId: value })
  }

  function refreshTaskLibrary() {
    emit('refresh-library', { categoryId: taskCategoryFilter.value })
  }

  function matchTaskKeyword(item: TektonDagLibraryItem, key: string) {
    if (!key) return true
    return `${item.name} ${item.code} ${item.categoryName || ''}`.toLowerCase().includes(key)
  }

  function tektonTaskDefaultIcon(source?: string) {
    if (source === 'clusterTask') return 'simple-icons:kubernetes'
    if (source === 'cluster') return 'ri:flow-chart'
    if (source === 'resolver') return 'ri:git-branch-line'
    return 'ri:flow-chart'
  }

  function tektonTaskDefaultColor(source?: string) {
    if (source === 'clusterTask') return '#326ce5'
    if (source === 'cluster') return '#0f766e'
    if (source === 'resolver') return '#7c3aed'
    return '#2563eb'
  }

  function taskLibraryIconColor(item: TektonDagLibraryItem) {
    return item.iconColor || tektonTaskDefaultColor(item.source)
  }

  function taskLibraryContractSummary(item: TektonDagLibraryItem) {
    const parsed = parseTaskContent(item.stageContent)
    const paramCount = libraryTaskParams(item).length || item.params?.length || parsed.params.length
    const resultCount = libraryTaskResults(item).length || parsed.results.length
    const workspaceCount = libraryTaskWorkspaces(item).length || parsed.workspaces.length
    return `${paramCount} 参数 / ${resultCount} Result / ${workspaceCount} Workspace`
  }

  function taskColorStyle(color?: string) {
    const nextColor = color || '#2563eb'
    return {
      color: nextColor,
      backgroundColor: `${nextColor}14`,
      borderColor: `${nextColor}33`
    }
  }

  function taskNodeStyle(node: TektonDagNode) {
    const color = node.iconColor || tektonTaskDefaultColor()
    return {
      '--task-color': color,
      '--task-color-soft': `${color}10`,
      '--task-color-border': `${color}38`
    }
  }

  function normalizeTektonParamType(type?: string): 'string' | 'array' | 'object' {
    if (type === 'array' || type === 'object') return type
    return 'string'
  }

  function formatTaskDefaultValue(value: unknown) {
    if (value === undefined || value === null) return ''
    if (Array.isArray(value)) return value.join(',')
    if (typeof value === 'object') return JSON.stringify(value, null, 2)
    return String(value)
  }

  function normalizePipelineParamProperties(value: unknown): TektonDagPipelineParam['properties'] {
    if (!value || typeof value !== 'object' || Array.isArray(value)) return undefined
    const result: TektonDagPipelineParam['properties'] = {}
    Object.entries(value as Record<string, any>).forEach(([key, item]) => {
      const type = normalizeTektonParamType(item?.type)
      result[key] = { type }
    })
    return Object.keys(result).length > 0 ? result : undefined
  }

  function pipelineEnumFromOptions(options: Array<{ value: string }>) {
    const values = options.map((item) => String(item.value || '').trim()).filter(Boolean)
    return values.length > 0 ? Array.from(new Set(values)) : undefined
  }

  function ensurePipelineParam(name: string, meta: PipelineParamMeta = {}) {
    const paramName = String(name || '').trim()
    if (!paramName) return ''
    if (!localConfig.params.some((item) => item.name === paramName)) {
      localConfig.params.push({
        name: paramName,
        displayName: meta.displayName || paramName,
        type: meta.type || 'string',
        paramType: meta.paramType,
        defaultValue: meta.defaultValue || '',
        description: meta.description || '',
        runtimeConfig: meta.runtimeConfig === true,
        required: meta.required === true,
        readonly: meta.readonly === true,
        options: meta.options || meta.selectList || [],
        selectList: meta.selectList || meta.options || [],
        enum: meta.enum || pipelineEnumFromOptions(meta.options || meta.selectList || []),
        properties: meta.properties,
        allowCustom: meta.allowCustom,
        config: meta.config,
        sourceCode: meta.sourceCode || paramName
      })
    } else if (Object.keys(meta).length > 0) {
      const current = localConfig.params.find((item) => item.name === paramName)
      if (current) {
        if (meta.type && !current.type) current.type = meta.type
        if (meta.displayName && !current.displayName) current.displayName = meta.displayName
        if (meta.paramType && !current.paramType) current.paramType = meta.paramType
        if (meta.defaultValue && !current.defaultValue) current.defaultValue = meta.defaultValue
        if (meta.description && !current.description) current.description = meta.description
        if (meta.required === true) current.required = true
        if (meta.runtimeConfig === true) current.runtimeConfig = true
        if (meta.readonly === true) current.readonly = true
        if ((meta.options || meta.selectList)?.length && !current.options?.length) {
          current.options = meta.options || meta.selectList
        }
        if ((meta.selectList || meta.options)?.length && !current.selectList?.length) {
          current.selectList = meta.selectList || meta.options
        }
        if (meta.enum?.length && !current.enum?.length) current.enum = meta.enum
        if (meta.properties && !current.properties) current.properties = meta.properties
        if (meta.allowCustom !== undefined) current.allowCustom = meta.allowCustom
        if (meta.config && !current.config) current.config = meta.config
        if (meta.sourceCode && !current.sourceCode) current.sourceCode = meta.sourceCode
      }
    }
    return paramName
  }

  function markPipelineParamRuntime(name?: string) {
    const paramName = String(name || '').trim()
    if (!paramName) return
    const param = localConfig.params.find((item) => item.name === paramName)
    if (param) param.runtimeConfig = true
  }

  function ensurePipelineWorkspace(
    name: string,
    optional = false,
    meta: {
      description?: string
      bindingType?: TektonWorkspaceBindingType
      bindingParams?: DevopsStepParam[]
    } = {}
  ) {
    const workspaceName = String(name || '').trim()
    if (!workspaceName) return ''
    let workspace = localConfig.workspaces.find((item) => item.name === workspaceName)
    if (!workspace) {
      workspace = {
        name: workspaceName,
        bindingType: meta.bindingType || 'emptyDir',
        allowedBindingTypes: [meta.bindingType || 'emptyDir'],
        optional,
        description: meta.description
      }
      localConfig.workspaces.push(workspace)
    } else {
      if (meta.description && !workspace.description) workspace.description = meta.description
      if (meta.bindingType && workspace.bindingType === 'emptyDir') {
        workspace.bindingType = meta.bindingType
      }
      workspace.allowedBindingTypes = uniqueWorkspaceBindingTypes([
        ...(workspace.allowedBindingTypes || []),
        meta.bindingType || workspace.bindingType || 'emptyDir'
      ])
    }
    ;(meta.bindingParams || []).forEach((param) => {
      const kind = workspaceBindingKind(param)
      if (kind === 'secret' || kind === 'persistentVolumeClaim') {
        applyWorkspaceBindingParam(workspaceName, kind)
      } else if (kind === 'volumeClaimTemplate') {
        applyDynamicPvcBindingParam(workspaceName, param)
      }
    })
    return workspaceName
  }

  function handlePipelineWorkspaceBindingTypeChange(workspace: TektonDagWorkspace) {
    workspace.allowedBindingTypes = uniqueWorkspaceBindingTypes([
      workspace.bindingType || 'emptyDir',
      ...(workspace.allowedBindingTypes || [])
    ])
    if (workspace.bindingType !== 'persistentVolumeClaim') workspace.claimName = ''
    if (workspace.bindingType !== 'secret') workspace.secretName = ''
    if (workspace.bindingType !== 'generatedSecret') {
      workspace.sourceType = undefined
      workspace.platformType = ''
      workspace.resourceId = ''
      workspace.resourceName = ''
      workspace.config = {}
    } else {
      workspace.allowedSourceTypes = workspace.allowedSourceTypes?.length
        ? workspace.allowedSourceTypes
        : ['platformCredential', 'configCenter']
      workspace.allowedPlatformTypes = workspace.allowedPlatformTypes?.length
        ? workspace.allowedPlatformTypes
        : ['username_password', 'token', 'ssh_key', 'secret_text']
    }
    if (workspace.bindingType !== 'configMap') workspace.configMapName = ''
    if (workspace.bindingType !== 'volumeClaimTemplate') {
      workspace.storageClassName = ''
      workspace.storage = ''
      workspace.accessMode = ''
      workspace.volumeMode = ''
    } else if (isWorkspaceContractMode.value) {
      workspace.storageClassName = ''
      workspace.storage = ''
      workspace.accessMode = ''
      workspace.volumeMode = ''
    } else {
      workspace.storage = workspace.storage || '10Gi'
      workspace.accessMode = workspace.accessMode || 'ReadWriteOnce'
    }
    if (isWorkspaceContractMode.value || workspace.bindingType === 'emptyDir') {
      workspace.subPath = ''
    }
    commit()
  }

  function handleWorkspaceAllowedBindingTypesChange(workspace: TektonDagWorkspace) {
    workspace.allowedBindingTypes = uniqueWorkspaceBindingTypes(workspace.allowedBindingTypes || [])
    if (workspace.bindingType && !workspace.allowedBindingTypes.includes(workspace.bindingType)) {
      workspace.allowedBindingTypes.unshift(workspace.bindingType)
    }
    if (!workspaceAllowsGeneratedSecret(workspace)) {
      workspace.allowedSourceTypes = []
      workspace.allowedPlatformTypes = []
    }
    commit()
  }

  function handleWorkspaceAllowedSourceTypesChange(workspace: TektonDagWorkspace) {
    workspace.allowedSourceTypes = uniqueWorkspaceSourceTypes(workspace.allowedSourceTypes || [])
    commit()
  }

  function handleConnect(connection: Connection) {
    if (props.readonly || !connection.source || !connection.target) return
    localConfig.edges.push(createEdge(connection.source, connection.target, 'runAfter'))
    commit()
  }

  function handleNodeClick(event: { node: { id: string } }) {
    closeContextMenu()
    activeNodeId.value = event.node.id
    openConfigDialog()
  }

  function handlePaneClick() {
    closeContextMenu()
    if (!configDialogVisible.value) activeNodeId.value = ''
  }

  function handleNodeContextMenu(payload: unknown) {
    const data = payload as {
      event?: MouseEvent
      node?: { id?: string }
      id?: string
    }
    const event = payload instanceof MouseEvent ? payload : data.event
    const nodeId = data.node?.id || data.id
    if (!event || !nodeId) return
    openNodeContextMenu(event, nodeId)
  }

  function openNodeContextMenu(event: MouseEvent, nodeId: string) {
    if (props.readonly) return
    event.preventDefault()
    event.stopPropagation()
    activeNodeId.value = nodeId
    const panel =
      event.currentTarget instanceof HTMLElement
        ? event.currentTarget.closest('.dag-canvas-panel')
        : document.querySelector<HTMLElement>('.dag-canvas-panel')
    const rect = panel?.getBoundingClientRect()
    contextMenu.value = {
      visible: true,
      x: rect ? event.clientX - rect.left + 8 : event.offsetX + 8,
      y: rect ? event.clientY - rect.top + 8 : event.offsetY + 8,
      nodeId
    }
  }

  function closeContextMenu() {
    contextMenu.value.visible = false
  }

  function openNodeConfigFromMenu(nodeId: string) {
    activeNodeId.value = nodeId
    closeContextMenu()
    openConfigDialog()
  }

  async function removeNode(nodeId: string) {
    const node = localConfig.nodes.find((item) => item.id === nodeId)
    if (!node || props.readonly) return
    closeContextMenu()
    try {
      await ElMessageBox.confirm(`确认删除节点「${node.displayName || node.name}」？`, '删除节点', {
        confirmButtonText: '删除',
        cancelButtonText: '取消',
        type: 'warning'
      })
    } catch (action) {
      if ((action as Action) !== 'confirm') return
    }
    localConfig.nodes.splice(
      0,
      localConfig.nodes.length,
      ...localConfig.nodes.filter((item) => item.id !== nodeId)
    )
    localConfig.edges.splice(
      0,
      localConfig.edges.length,
      ...localConfig.edges.filter((edge) => edge.source !== nodeId && edge.target !== nodeId)
    )
    if (activeNodeId.value === nodeId) activeNodeId.value = ''
    commit()
  }

  function handleNodeDragStop(event: NodeDragEvent) {
    const node = localConfig.nodes.find((item) => item.id === event.node.id)
    if (!node) return
    node.position = { x: event.node.position.x, y: event.node.position.y }
    commit()
  }

  function applyAutoLayout() {
    Object.assign(localConfig, layoutTektonDag(localConfig))
    commit()
  }

  function fitView() {
    flowRef.value?.fitView?.({ padding: 0.24 })
  }

  function paramMetaFromRequirement(
    requirement?: TektonDagParamRequirement,
    param?: TektonDagTaskParam
  ): PipelineParamMeta {
    return {
      type: requirement?.type || 'string',
      paramType:
        requirement?.paramType ||
        param?.paramType ||
        platformParamTypeFromTektonType(requirement?.type),
      defaultValue: requirement?.defaultValue || param?.defaultValue || '',
      description: requirement?.description || param?.description || '',
      required: requirement?.required === true || param?.required === true,
      runtimeConfig: requirement?.runtimeConfig === true || param?.runtimeConfig === true,
      readonly: requirement?.readonly === true || param?.readonly === true,
      options: requirement?.options || requirement?.selectList || param?.options || [],
      selectList: requirement?.selectList || requirement?.options || param?.options || [],
      allowCustom: requirement?.allowCustom,
      config: requirement?.config,
      sourceCode: requirement?.sourceCode || param?.sourceCode || requirement?.name || param?.name,
      displayName:
        requirement?.displayName || param?.displayName || requirement?.name || param?.name
    }
  }

  function createNodeParamFromRequirement(
    requirement?: TektonDagParamRequirement
  ): TektonDagTaskParam {
    if (!requirement) {
      return { name: '', source: 'fixed', value: '' }
    }
    const exposeAsPipelineParam = shouldExposeRequirementAsPipelineParam(requirement)
    const source = exposeAsPipelineParam ? 'pipelineParam' : 'fixed'
    const pipelineParam = exposeAsPipelineParam
      ? ensurePipelineParam(requirement.name, paramMetaFromRequirement(requirement))
      : ''
    if (requirement.runtimeConfig) markPipelineParamRuntime(pipelineParam)
    return {
      name: requirement.name,
      source,
      pipelineParam,
      value: requirement.defaultValue || '',
      sourceCode: requirement.sourceCode || requirement.name,
      displayName: requirement.displayName,
      defaultValue: requirement.defaultValue,
      description: requirement.description,
      required: requirement.required,
      readonly: requirement.readonly,
      paramType: requirement.paramType || platformParamTypeFromTektonType(requirement.type),
      options: requirement.options || requirement.selectList || [],
      selectList: requirement.selectList || requirement.options || [],
      allowCustom: requirement.allowCustom,
      config: requirement.config,
      runtimeConfig: requirement.runtimeConfig
    }
  }

  function shouldExposeRequirementAsPipelineParam(requirement?: TektonDagParamRequirement) {
    return (
      requirement?.exposed === true ||
      requirement?.runtimeConfig === true ||
      requirement?.readonly === true
    )
  }

  function isRequiredNodeParam(
    param?: TektonDagTaskParam,
    requirement?: TektonDagParamRequirement
  ) {
    return (
      param?.required === true ||
      param?.readonly === true ||
      param?.runtimeConfig === true ||
      requirement?.required === true ||
      requirement?.readonly === true ||
      requirement?.runtimeConfig === true
    )
  }

  function requiredNodeParamCount(node?: TektonDagNode) {
    return requiredNodeParamNames(node).size
  }

  function requiredNodeParamNames(node?: TektonDagNode) {
    const names = new Set<string>()
    if (!node) return names
    node.params.forEach((param) => {
      const requirement = node.paramRequirements?.find((item) => item.name === param.name)
      if (isRequiredNodeParam(param, requirement)) names.add(param.name)
    })
    ;(node.paramRequirements || []).forEach((requirement) => {
      if (isRequiredNodeParam(undefined, requirement)) names.add(requirement.name)
    })
    return names
  }

  function configuredRequiredNodeParamCount(node?: TektonDagNode) {
    if (!node) return 0
    return Array.from(requiredNodeParamNames(node)).filter((name) => {
      const param = node.params.find((item) => item.name === name)
      if (!param) return false
      if (isPipelineParamSource(param.source)) {
        return Boolean(param.pipelineParam)
      }
      if (param.source === 'result') return Boolean(param.resultNodeId && param.resultName)
      return Boolean(String(param.value ?? param.defaultValue ?? '').trim())
    }).length
  }

  function nodeParamOptions(param: TektonDagTaskParam, requirement?: TektonDagParamRequirement) {
    return (
      param.selectList ||
      param.options ||
      requirement?.selectList ||
      requirement?.options ||
      requirement?.config?.options ||
      param.config?.options ||
      []
    )
  }

  function nodeParamTypeText(type?: string) {
    const map: Record<string, string> = {
      string: '字符串',
      text: '多行文本',
      password: '密码',
      file: '文件上传',
      number: '数字',
      booleanParam: '布尔',
      choice: '单选',
      objectList: '对象列表',
      list: '列表',
      channelGroupCode: '渠道变量',
      channelCredential: '渠道凭证',
      gitProject: 'Git 项目',
      gitRepositoryUrl: 'Git 仓库',
      gitBranch: 'Git 分支',
      gitTag: 'Git Tag',
      harborProject: 'Harbor 项目',
      harborImage: 'Harbor 镜像',
      configCenter: '配置中心',
      mavenConfig: 'Maven 配置',
      hostGroupTargets: '主机目标组',
      voucherModel: '平台凭证',
      workspacePvc: 'Workspace PVC',
      workspaceSecret: 'Workspace Secret',
      kubernetesPvcName: 'PVC',
      kubernetesSecretName: 'Secret'
    }
    return map[String(type || '')] || type || '字符串'
  }

  function normalizePipelinePlatformParamType(type?: string) {
    const value = String(type || 'string').trim()
    return pipelinePlatformParamTypeValues.has(value) ? value : 'string'
  }

  function nodeParamEffectiveTypeText(
    param: TektonDagTaskParam,
    requirement?: TektonDagParamRequirement
  ) {
    const pipelineParam = nodePipelineParam(param)
    return nodeParamTypeText(pipelineParam?.paramType || param.paramType || requirement?.paramType)
  }

  function nodeParamOfficialTypeText(
    param: TektonDagTaskParam,
    requirement?: TektonDagParamRequirement
  ) {
    const pipelineParam = nodePipelineParam(param)
    return (
      pipelineParam?.type ||
      requirement?.type ||
      tektonTypeFromPipelineParamType(pipelineParam?.paramType || param.paramType)
    )
  }

  function isPipelineParamSource(source?: string) {
    return source === 'pipelineParam' || source === 'runtime'
  }

  function isPipelineParamBinding(param?: TektonDagTaskParam) {
    return Boolean(param && isPipelineParamSource(param.source))
  }

  function normalizeRuntimeParamSource(param?: TektonDagTaskParam) {
    if (!param || param.source !== 'runtime') return
    param.source = 'pipelineParam'
    param.runtimeConfig = true
    if (param.pipelineParam) markPipelineParamRuntime(param.pipelineParam)
  }

  function nodeParamSourceText(source?: string) {
    const map: Record<string, string> = {
      fixed: '固定值',
      pipelineParam: 'Pipeline 参数',
      runtime: 'Pipeline 参数',
      result: 'Task Result',
      dynamic: '动态值'
    }
    return map[String(source || '')] || '固定值'
  }

  function nodeParamBindingText(param: TektonDagTaskParam) {
    if (isPipelineParamSource(param.source)) {
      const pipelineParam = nodePipelineParam(param)
      const label = pipelineParam?.displayName
        ? `${pipelineParam.displayName} / ${pipelineParam.name}`
        : param.pipelineParam || '未绑定'
      const suffix = pipelineParam?.runtimeConfig ? '，可运行时覆盖' : ''
      return `取值：Pipeline 参数 ${label}${suffix}`
    }
    if (param.source === 'result') {
      const sourceNode = localConfig.nodes.find((item) => item.id === param.resultNodeId)
      const sourceName = sourceNode?.displayName || sourceNode?.name || '未选择上游 Task'
      return `取值：${sourceName}.${param.resultName || '未选择 Result'}`
    }
    const value = String(param.value ?? param.defaultValue ?? '').trim()
    return `取值：${value || '未填写固定值'}`
  }

  function nodeParamDescriptionText(
    param: TektonDagTaskParam,
    requirement?: TektonDagParamRequirement
  ) {
    const pipelineParam = nodePipelineParam(param)
    return pipelineParam?.description || param.description || requirement?.description || ''
  }

  function pipelineParamOptionLabel(item: TektonDagPipelineParam) {
    const tags = [
      item.paramType ? nodeParamTypeText(item.paramType) : '',
      item.runtimeConfig ? '可覆盖' : '',
      item.required ? '必填' : ''
    ].filter(Boolean)
    const label = item.displayName ? `${item.displayName} / ${item.name}` : item.name
    return tags.length ? `${label}（${tags.join(' / ')}）` : label
  }

  function pipelineFieldOptions() {
    const result = new Map<string, TektonDagPipelineParam>()
    localConfig.params.forEach((item) => {
      const name = String(item.name || '').trim()
      if (name) result.set(name, item)
    })
    ;(activeNode.value?.paramRequirements || []).forEach((item) => {
      const name = String(item.name || '').trim()
      if (!name || result.has(name)) return
      result.set(name, {
        name,
        displayName: item.displayName,
        type: normalizeTektonParamType(item.type),
        paramType: item.paramType || platformParamTypeFromTektonType(item.type),
        defaultValue: item.defaultValue,
        description: item.description,
        runtimeConfig: item.runtimeConfig,
        required: item.required,
        readonly: item.readonly,
        selectList: item.selectList || item.options || [],
        options: item.options || item.selectList || [],
        allowCustom: item.allowCustom,
        config: item.config,
        sourceCode: item.sourceCode || name
      })
    })
    return Array.from(result.values())
  }

  function taskParamOptionLabel(item: TektonDagParamRequirement) {
    return item.displayName && item.displayName !== item.name
      ? `${item.displayName} / ${item.name}`
      : item.name
  }

  function nodePipelineParam(param: TektonDagTaskParam) {
    return localConfig.params.find((item) => item.name === param.pipelineParam)
  }

  function ensureNodePipelineParam(param: TektonDagTaskParam) {
    const name = String(param.pipelineParam || param.name || '').trim()
    if (!name) return undefined
    param.pipelineParam = name
    ensurePipelineParam(name, paramMetaFromRequirement(undefined, param))
    return nodePipelineParam(param)
  }

  function pipelineParamContractValue(
    param: TektonDagTaskParam,
    key: keyof TektonDagPipelineParam
  ) {
    const pipelineParam = nodePipelineParam(param)
    return pipelineParam?.[key] ?? ''
  }

  function pipelineParamBooleanValue(
    param: TektonDagTaskParam,
    key: 'required' | 'runtimeConfig' | 'readonly' | 'allowCustom',
    defaultValue = false
  ) {
    const value = nodePipelineParam(param)?.[key]
    return value === undefined ? defaultValue : value === true
  }

  function pipelineParamConfigBooleanValue(
    param: TektonDagTaskParam,
    key: keyof NonNullable<DevopsStepParam['config']>
  ) {
    return nodePipelineParam(param)?.config?.[key] === true
  }

  function pipelineParamFormType(param: TektonDagTaskParam) {
    return String(pipelineParamContractValue(param, 'paramType') || param.paramType || 'string')
  }

  function pipelineParamExpectedTektonType(param: TektonDagTaskParam) {
    return tektonTypeFromPipelineParamType(pipelineParamFormType(param))
  }

  function pipelineParamActualTektonType(param: TektonDagTaskParam) {
    return normalizePipelineParamTektonType(
      String(pipelineParamContractValue(param, 'type') || 'string')
    )
  }

  function pipelineParamTypeMismatchText(param: TektonDagTaskParam) {
    const expected = pipelineParamExpectedTektonType(param)
    const actual = pipelineParamActualTektonType(param)
    if (expected === actual) return ''
    return `当前 Tekton 类型为 ${actual}，保存时会同步为 ${expected}`
  }

  function pipelineParamContractSummary(param: TektonDagTaskParam) {
    const pipelineParam = nodePipelineParam(param)
    const parts = [
      pipelineParam?.name
        ? `${pipelineParam.displayName || pipelineParam.name} / ${pipelineParam.name}`
        : '未绑定 Pipeline 参数',
      pipelineParam?.required ? '必填' : '可选',
      pipelineParam?.runtimeConfig ? '允许运行时覆盖' : '项目流水线配置值'
    ]
    return parts.join(' / ')
  }

  function pipelineParamNumberDefaultValue(param: TektonDagTaskParam) {
    const value = String(pipelineParamContractValue(param, 'defaultValue') || '').trim()
    if (!value) return undefined
    const next = Number(value)
    return Number.isFinite(next) ? next : undefined
  }

  function updatePipelineParamDefaultValue(
    param: TektonDagTaskParam,
    value: string | number | boolean | undefined
  ) {
    updatePipelineParamContract(param, {
      defaultValue: value === undefined || value === null ? '' : String(value)
    })
  }

  function normalizePipelineParamTektonType(value?: string): TektonDagPipelineParam['type'] {
    return value === 'array' || value === 'object' ? value : 'string'
  }

  function tektonTypeFromPipelineParamType(value?: string): TektonDagPipelineParam['type'] {
    const paramType = String(value || '').trim()
    if (paramType === 'list' || paramType === 'array') return 'array'
    if (paramType === 'objectList' || paramType === 'object') return 'object'
    return 'string'
  }

  function syncPipelineParamOfficialType(param: TektonDagTaskParam) {
    const pipelineParam = ensureNodePipelineParam(param)
    if (!pipelineParam) return
    const nextType = tektonTypeFromPipelineParamType(pipelineParam.paramType || param.paramType)
    pipelineParam.type = nextType
    if (nextType !== 'object') pipelineParam.properties = undefined
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
  }

  function updatePipelineParamContract(
    param: TektonDagTaskParam,
    patch: Partial<
      Pick<
        TektonDagPipelineParam,
        | 'paramType'
        | 'displayName'
        | 'defaultValue'
        | 'description'
        | 'required'
        | 'runtimeConfig'
        | 'readonly'
        | 'allowCustom'
      >
    > & { type?: string }
  ) {
    normalizeRuntimeParamSource(param)
    const pipelineParam = ensureNodePipelineParam(param)
    if (!pipelineParam) return
    if (patch.type !== undefined) {
      pipelineParam.type = normalizePipelineParamTektonType(patch.type)
      if (!patch.paramType && (!pipelineParam.paramType || pipelineParam.paramType === 'string')) {
        pipelineParam.paramType = platformParamTypeFromTektonType(pipelineParam.type)
      }
    }
    if (patch.paramType !== undefined) {
      pipelineParam.paramType = normalizePipelinePlatformParamType(patch.paramType)
      pipelineParam.type = tektonTypeFromPipelineParamType(pipelineParam.paramType)
      if (!pipelineParam.config) pipelineParam.config = {}
      sanitizePipelineParamConfigForType(pipelineParam)
      if (
        pipelineParam.paramType === 'booleanParam' &&
        !String(pipelineParam.defaultValue || '').trim()
      ) {
        pipelineParam.defaultValue = 'false'
      }
      if (pipelineParam.paramType === 'number' && pipelineParam.defaultValue) {
        const parsed = Number(pipelineParam.defaultValue)
        if (!Number.isFinite(parsed)) pipelineParam.defaultValue = ''
      }
    }
    if (patch.defaultValue !== undefined) pipelineParam.defaultValue = patch.defaultValue
    if (patch.displayName !== undefined) pipelineParam.displayName = patch.displayName
    if (patch.description !== undefined) pipelineParam.description = patch.description
    if (patch.required !== undefined) pipelineParam.required = patch.required
    if (patch.runtimeConfig !== undefined) pipelineParam.runtimeConfig = patch.runtimeConfig
    if (patch.readonly !== undefined) pipelineParam.readonly = patch.readonly
    if (patch.allowCustom !== undefined) pipelineParam.allowCustom = patch.allowCustom
    if (pipelineParam.paramType === 'channelCredential') {
      pipelineParam.readonly = true
      pipelineParam.runtimeConfig = false
    }
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  function syncNodeParamMetaFromPipelineParam(
    param: TektonDagTaskParam,
    pipelineParam: TektonDagPipelineParam
  ) {
    param.paramType = pipelineParam.paramType || param.paramType
    param.defaultValue = pipelineParam.defaultValue || param.defaultValue
    param.description = pipelineParam.description || param.description
    param.required = pipelineParam.required
    param.readonly = pipelineParam.readonly
    param.runtimeConfig = pipelineParam.runtimeConfig
    param.allowCustom = pipelineParam.allowCustom
    param.options = pipelineParam.options || pipelineParam.selectList || param.options
    param.selectList = pipelineParam.selectList || pipelineParam.options || param.selectList
    param.config = pipelineParam.config || param.config
  }

  function pipelineParamPropertiesText(param: TektonDagTaskParam) {
    const properties = nodePipelineParam(param)?.properties
    return properties ? JSON.stringify(properties, null, 2) : ''
  }

  function updatePipelineParamProperties(param: TektonDagTaskParam, value: string) {
    const pipelineParam = ensureNodePipelineParam(param)
    if (!pipelineParam) return
    const text = String(value || '').trim()
    if (!text) {
      pipelineParam.properties = undefined
      commit()
      return
    }
    const doc = parseDocument(text)
    if (doc.errors.length > 0) {
      ElMessage.warning('对象属性必须是合法 JSON 或 YAML')
      return
    }
    const properties = normalizePipelineParamProperties(doc.toJSON())
    if (!properties) {
      ElMessage.warning('对象属性必须是对象，例如 {"url":{"type":"string"}}')
      return
    }
    pipelineParam.properties = properties
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  function pipelineParamRequiresPlatformConfig(param: TektonDagTaskParam) {
    return [
      'channelGroupCode',
      'voucherModel',
      'channelCredential',
      'configCenter',
      'hostGroupTargets'
    ].includes(pipelineParamFormType(param))
  }

  function pipelinePlatformConfigHint(param: TektonDagTaskParam) {
    const map: Record<string, string> = {
      channelGroupCode: '配置渠道分组、渠道类型、映射字段和依赖参数。',
      voucherModel: '限制项目流水线选择的凭证类型和输出字段。',
      channelCredential: '从渠道实例解析凭证，密钥不进入普通明文参数。',
      configCenter: '限制配置中心类型和输出方式。',
      hostGroupTargets: '限制主机组目标的输出结构。'
    }
    return map[pipelineParamFormType(param)] || '配置平台解析规则。'
  }

  function ensurePipelineParamConfig(param: TektonDagTaskParam) {
    const pipelineParam = ensureNodePipelineParam(param)
    if (!pipelineParam) return undefined
    if (!pipelineParam.config) pipelineParam.config = {}
    return pipelineParam.config
  }

  function pipelineParamConfigValue(
    param: TektonDagTaskParam,
    key: keyof NonNullable<DevopsStepParam['config']>
  ) {
    const value = nodePipelineParam(param)?.config?.[key]
    return value === undefined || value === null ? '' : String(value)
  }

  function pipelineCredentialMode(param: TektonDagTaskParam) {
    const mode = pipelineParamConfigValue(param, 'credentialMode')
    return ['field', 'jsonData', 'jsonBase64'].includes(mode) ? mode : 'field'
  }

  function sanitizePipelineParamConfigForType(pipelineParam: TektonDagPipelineParam) {
    const config = pipelineParam.config || {}
    const paramType = String(pipelineParam.paramType || '').trim()
    if (paramType !== 'channelGroupCode') {
      delete config.channelGroupCode
      delete config.channelTypeFilter
      delete config.channelParamCode
      delete config.channelBindingId
      delete config.projectParamCode
      delete config.componentParamCode
      delete config.dependencyParamCodes
    }
    if (!['channelGroupCode', 'voucherModel', 'channelCredential'].includes(paramType)) {
      delete config.mappingField
    }
    if (paramType !== 'voucherModel') {
      delete config.voucherModel
    }
    if (!['voucherModel', 'channelCredential'].includes(paramType)) {
      delete config.credentialMode
      delete config.credentialSourceParamCode
    }
    if (paramType !== 'configCenter') {
      delete config.configTypeId
      delete config.configTypeCode
      delete config.valueMode
    }
    if (paramType !== 'hostGroupTargets') {
      delete config.renderMode
    }
    if (paramType === 'voucherModel' || paramType === 'channelCredential') {
      config.credentialMode = config.credentialMode || 'field'
      if (
        config.credentialMode === 'field' &&
        !credentialFieldOptions.some((item) => item.value === config.mappingField)
      ) {
        config.mappingField = 'username'
      }
    }
    if (paramType === 'configCenter') config.valueMode = config.valueMode || 'string'
    if (paramType === 'hostGroupTargets') config.renderMode = config.renderMode || 'json'
    pipelineParam.config = config
  }

  function updatePipelineParamConfigField(
    param: TektonDagTaskParam,
    key: keyof NonNullable<DevopsStepParam['config']>,
    value: string | number | boolean | undefined
  ) {
    const pipelineParam = ensureNodePipelineParam(param)
    const config = ensurePipelineParamConfig(param)
    if (!pipelineParam || !config) return
    ;(config as Record<string, any>)[key] = value
    if (key === 'credentialMode') {
      if (String(value || '') === 'field') config.mappingField = config.mappingField || 'username'
      else config.mappingField = ''
    }
    sanitizePipelineParamConfigForType(pipelineParam)
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  async function fetchPipelinePlatformOptions() {
    if (platformOptionLoading.value) return
    if (channelGroupOptions.value.length > 0 && channelTypes.value.length > 0) return
    platformOptionLoading.value = true
    try {
      const [groupRes, channelTypeRes] = await Promise.all([
        listDevopsChannelGroupOptionsApi(),
        listDevopsChannelTypeApi({ page: 1, pageSize: 200, status: 1 })
      ])
      channelGroupOptions.value = groupRes.items || []
      channelTypes.value = channelTypeRes.items || []
    } catch {
      ElMessage.error('加载平台参数选项失败')
    } finally {
      platformOptionLoading.value = false
    }
  }

  async function handlePipelinePlatformOptionVisible(visible: boolean) {
    if (!visible) return
    await fetchPipelinePlatformOptions()
  }

  const mappingCacheKey = (groupCode?: string, channelType?: string, channelTypeId?: string) =>
    [
      String(groupCode || '').trim(),
      String(channelType || '').trim(),
      String(channelTypeId || '').trim()
    ].join(':')

  const pipelineChannelTypeIdByCode = (groupCode?: string, channelType?: string) =>
    channelTypes.value.find(
      (item) =>
        item.code === String(channelType || '').trim() &&
        item.groupCode === String(groupCode || '').trim()
    )?.id || ''

  async function ensurePipelineMappingFields(groupCode?: string, channelType?: string) {
    await fetchPipelinePlatformOptions()
    const channelTypeId = pipelineChannelTypeIdByCode(groupCode, channelType)
    const key = mappingCacheKey(groupCode, channelType, channelTypeId)
    if (!groupCode || mappingFieldMap.value[key]) return
    const res = await listDevopsChannelMappingFieldOptionsApi(groupCode, channelType, channelTypeId)
    mappingFieldMap.value[key] = res.items || []
  }

  const parsePipelineMappingFields = (raw?: string): ChannelMappingFieldOption[] => {
    const text = String(raw || '').trim()
    if (!text) return []
    try {
      const data = JSON.parse(text) as { fields?: Array<Record<string, any>> }
      const parseDependencies = (items: unknown): ChannelMappingFieldDependency[] =>
        Array.isArray(items)
          ? items.map((dep) => ({
              field: String(dep.field || '').trim(),
              source: String(dep.source || '').trim(),
              name: String(dep.name || dep.field || '').trim(),
              required: dep.required !== false
            }))
          : []
      return (data.fields || [])
        .map((item) => ({
          groupCode: String(item.groupCode || '').trim(),
          field: String(item.field || '').trim(),
          name: String(item.name || item.field || '').trim(),
          kind: String(item.kind || '').trim(),
          uiControl: String(item.uiControl || '').trim(),
          provider: String(item.provider || '').trim(),
          dependencies: parseDependencies(item.dependencies),
          allOf: parseDependencies(item.allOf),
          anyOf: parseDependencies(item.anyOf),
          allowManualInput: Boolean(item.allowManualInput),
          addressBuilder: String(item.addressBuilder || '').trim(),
          required: Boolean(item.required),
          channelTypes: Array.isArray(item.channelTypes)
            ? item.channelTypes.map((value) => String(value || '').trim()).filter(Boolean)
            : [],
          valueType: String(item.valueType || '').trim(),
          outputMode: String(item.outputMode || '').trim(),
          outputTemplate: String(item.outputTemplate || '').trim(),
          credentialPolicy: String(item.credentialPolicy || '').trim(),
          fallbackAddressFields: Array.isArray(item.fallbackAddressFields)
            ? item.fallbackAddressFields.map((value) => String(value || '').trim()).filter(Boolean)
            : [],
          resolveEndpoint: Boolean(item.resolveEndpoint),
          options: Array.isArray(item.options)
            ? item.options.map((option) => ({
                label: String(option.label || option.value || '').trim(),
                value: String(option.value || '').trim()
              }))
            : [],
          sortOrder: Number(item.sortOrder || 0)
        }))
        .filter((item) => item.field)
    } catch {
      return []
    }
  }

  const pipelineMappingOptions = (groupCode?: string, channelType?: string) =>
    groupCode
      ? mappingFieldMap.value[
          mappingCacheKey(
            groupCode,
            channelType,
            pipelineChannelTypeIdByCode(groupCode, channelType)
          )
        ] || []
      : []

  const dedupePipelineMappingOptions = (items: ChannelMappingFieldOption[]) => {
    const seen = new Set<string>()
    return items.filter((item) => {
      const field = String(item.field || '').trim()
      if (!field || seen.has(field)) return false
      seen.add(field)
      return true
    })
  }

  const codeRepoMappingFields = new Set([
    'id',
    'name',
    'code',
    'endpoint',
    'dynamic.project',
    'dynamic.branch',
    'dynamic.tag',
    'address.projectUrl'
  ])

  const imageRegistryMappingFields = new Set([
    'id',
    'name',
    'code',
    'endpoint',
    'dynamic.project',
    'dynamic.image',
    'dynamic.tag',
    'dynamic.imageTag',
    'address.projectUrl',
    'address.imageUrl',
    'address.imageTagUrl'
  ])
  const artifactRepoMappingFields = new Set([
    'id',
    'name',
    'code',
    'endpoint',
    'dynamic.repository',
    'dynamic.artifactName',
    'dynamic.artifactVersion',
    'address.repositoryUrl',
    'address.artifactUrl',
    'address.artifactVersionUrl'
  ])
  const scanMappingFields = new Set([
    'id',
    'name',
    'code',
    'endpoint',
    'dynamic.projectName',
    'dynamic.projectKey'
  ])
  const deployMappingFields = new Set([
    'id',
    'name',
    'code',
    'endpoint',
    'dynamic.namespace',
    'dynamic.workloadType',
    'dynamic.resourceName',
    'dynamic.containerName',
    'dynamic.imageName',
    'address.deploymentConfig',
    'address.deployConfig',
    'address.hostConfig',
    'address.groupConfig'
  ])

  function filterPipelineCodeRepoMappingOptions(
    groupCode: string | undefined,
    items: ChannelMappingFieldOption[]
  ) {
    const group = String(groupCode || '').trim()
    if (group === 'image_registry') {
      return items.filter((item) => imageRegistryMappingFields.has(String(item.field || '').trim()))
    }
    if (group === 'artifact_repo') {
      return items.filter((item) => artifactRepoMappingFields.has(String(item.field || '').trim()))
    }
    if (group === 'code_scan') {
      return items.filter((item) => scanMappingFields.has(String(item.field || '').trim()))
    }
    if (group === 'deploy_target') {
      return items.filter((item) => deployMappingFields.has(String(item.field || '').trim()))
    }
    if (group !== 'code_repo') return items
    return items.filter((item) => codeRepoMappingFields.has(String(item.field || '').trim()))
  }

  function pipelineChannelTypeOptions(param: TektonDagTaskParam) {
    const groupCode = pipelineParamConfigValue(param, 'channelGroupCode')
    return channelTypes.value
      .filter((item) => item.status === 1 && item.groupCode === groupCode)
      .map((item) => ({ label: `${item.name}（${item.code}）`, value: item.code }))
  }

  function pipelineChannelMappingOptions(param: TektonDagTaskParam) {
    const groupCode = pipelineParamConfigValue(param, 'channelGroupCode')
    const channelType = pipelineParamConfigValue(param, 'channelTypeFilter')
    const options = pipelineMappingOptions(groupCode, channelType)
    const type = channelTypes.value.find(
      (item) => item.code === channelType && item.groupCode === groupCode
    )
    return filterPipelineCodeRepoMappingOptions(
      groupCode,
      dedupePipelineMappingOptions([...options, ...parsePipelineMappingFields(type?.mappingFields)])
    )
  }

  function currentPipelineMappingFieldOption(param: TektonDagTaskParam) {
    const field = pipelineParamConfigValue(param, 'mappingField')
    return pipelineChannelMappingOptions(param).find((item) => item.field === field)
  }

  const fallbackChannelDependencies = (
    param: TektonDagTaskParam
  ): ChannelMappingFieldDependency[] => {
    const field = pipelineParamConfigValue(param, 'mappingField')
    const channelType = pipelineParamConfigValue(param, 'channelTypeFilter')
    const groupCode = pipelineParamConfigValue(param, 'channelGroupCode')
    const map: Record<string, ChannelMappingFieldDependency[]> = {
      'dynamic.branch': [
        { field: 'dynamic.project', source: 'param', name: '仓库项目或仓库地址', required: true }
      ],
      'dynamic.artifact': [
        { field: 'dynamic.repository', source: 'param', name: '仓库或制品地址', required: true }
      ],
      'dynamic.artifactName': [
        { field: 'dynamic.repository', source: 'param', name: '仓库或制品地址', required: true }
      ],
      'dynamic.artifactVersion': [
        { field: 'dynamic.repository', source: 'param', name: '仓库或制品地址', required: true },
        { field: 'dynamic.artifactName', source: 'param', name: '制品', required: true }
      ],
      'dynamic.artifactTag': [
        { field: 'dynamic.repository', source: 'param', name: '仓库或制品地址', required: true },
        { field: 'dynamic.artifactName', source: 'param', name: '制品', required: true }
      ],
      'dynamic.image': [
        { field: 'dynamic.project', source: 'param', name: '镜像项目或镜像地址', required: true }
      ],
      'dynamic.imageTag': [
        { field: 'dynamic.project', source: 'param', name: '镜像项目或镜像地址', required: true },
        { field: 'dynamic.image', source: 'param', name: '镜像', required: true }
      ],
      'dynamic.cluster': [
        { field: 'dynamic.project', source: 'param', name: '项目', required: true }
      ],
      'dynamic.workspace': [
        { field: 'dynamic.project', source: 'param', name: '项目', required: true },
        { field: 'dynamic.cluster', source: 'param', name: '集群', required: true }
      ],
      'dynamic.application': [
        { field: 'dynamic.workspace', source: 'param', name: '工作空间', required: true }
      ],
      'dynamic.version': [
        { field: 'dynamic.application', source: 'param', name: '应用', required: true }
      ]
    }
    if (field === 'dynamic.tag') {
      return groupCode === 'image_registry'
        ? [
            {
              field: 'dynamic.project',
              source: 'param',
              name: '镜像项目或镜像地址',
              required: true
            },
            { field: 'dynamic.image', source: 'param', name: '镜像', required: true }
          ]
        : [
            {
              field: 'dynamic.project',
              source: 'param',
              name: '仓库项目或仓库地址',
              required: true
            }
          ]
    }
    if (field === 'dynamic.container' || field === 'dynamic.containerName') {
      return channelType === 'kubernetes'
        ? [{ field: 'dynamic.namespace', source: 'param', name: '命名空间', required: true }]
        : [
            { field: 'dynamic.application', source: 'param', name: '应用', required: true },
            { field: 'dynamic.version', source: 'param', name: '版本', required: true }
          ]
    }
    if (
      [
        'dynamic.deployment',
        'dynamic.statefulSet',
        'dynamic.daemonSet',
        'dynamic.cronJob',
        'dynamic.resourceName'
      ].includes(field)
    ) {
      return [{ field: 'dynamic.namespace', source: 'param', name: '命名空间', required: true }]
    }
    if (field === 'dynamic.imageName') {
      return [
        { field: 'dynamic.namespace', source: 'param', name: '命名空间', required: true },
        { field: 'dynamic.resourceName', source: 'param', name: '部署资源', required: true },
        { field: 'dynamic.containerName', source: 'param', name: '容器', required: true }
      ]
    }
    return map[field] || []
  }

  function pipelineChannelParamDependencies(
    param: TektonDagTaskParam
  ): ChannelMappingFieldDependency[] {
    const option = currentPipelineMappingFieldOption(param)
    const fromOption = (option?.dependencies || [])
      .filter((dep) => dep.source === 'param' && dep.field)
      .map((dep) => ({
        ...dep,
        name: tektonCodeRepoGitRefProjectDependency(param, dep.field)
          ? '仓库项目或仓库地址'
          : dep.name,
        required: option?.allowManualInput === true ? false : dep.required
      }))
    return fromOption.length ? fromOption : fallbackChannelDependencies(param)
  }

  function tektonCodeRepoGitRefProjectDependency(param: TektonDagTaskParam, field: string) {
    return (
      pipelineParamConfigValue(param, 'channelGroupCode') === 'code_repo' &&
      ['dynamic.branch', 'dynamic.tag'].includes(pipelineParamConfigValue(param, 'mappingField')) &&
      field === 'dynamic.project'
    )
  }

  function pipelineChannelVariableNeedsEndpointParam(param: TektonDagTaskParam) {
    const field = pipelineParamConfigValue(param, 'mappingField')
    const option = currentPipelineMappingFieldOption(param)
    if (
      option?.kind === 'address' ||
      ['addressPicker', 'cascadePicker'].includes(option?.uiControl || '')
    )
      return false
    if (
      option?.uiControl === 'deployConfig' ||
      ['dynamic.deployConfig', 'address.deployConfig', 'address.deploymentConfig'].includes(field)
    )
      return false
    if (field.endsWith('Url')) return false
    if (field.startsWith('dynamic.')) return true
    return (option?.dependencies || []).some(
      (dep) => dep.source === 'channel' && dep.field === 'endpoint' && dep.required !== false
    )
  }

  const pipelineNeedChannelBindingParam = (param: TektonDagTaskParam) =>
    pipelineParamFormType(param) === 'channelGroupCode' &&
    pipelineChannelVariableNeedsEndpointParam(param) &&
    !tektonUseCodeRepoAddressDependency(param)

  function pipelineChannelBindingLabel(param: TektonDagTaskParam) {
    return pipelineParamConfigValue(param, 'channelGroupCode') === 'code_repo'
      ? '渠道实例'
      : '渠道实例'
  }

  function samePipelineChannelVariableTarget(
    source: TektonDagPipelineParam,
    target: TektonDagPipelineParam,
    fields: string | string[]
  ) {
    if (target.paramType !== 'channelGroupCode' || source.name === target.name) return false
    const sourceConfig = source.config || {}
    const targetConfig = target.config || {}
    const allowedFields = Array.isArray(fields) ? fields : [fields]
    return (
      String(targetConfig.channelGroupCode || '').trim() ===
        String(sourceConfig.channelGroupCode || '').trim() &&
      String(targetConfig.channelTypeFilter || '').trim() ===
        String(sourceConfig.channelTypeFilter || '').trim() &&
      allowedFields.includes(String(targetConfig.mappingField || '').trim())
    )
  }

  function pipelineChannelVariableOptions(param: TektonDagTaskParam) {
    const pipelineParam = nodePipelineParam(param)
    if (!pipelineParam) return []
    return localConfig.params.filter((item) =>
      samePipelineChannelVariableTarget(pipelineParam, item, 'endpoint')
    )
  }

  function pipelineChannelDependencyOptions(param: TektonDagTaskParam, field: string) {
    const pipelineParam = nodePipelineParam(param)
    if (!pipelineParam) return []
    const fields = tektonCodeRepoGitRefProjectDependency(param, field)
      ? ['dynamic.project', 'address.projectUrl']
      : [field]
    return localConfig.params.filter((item) =>
      samePipelineChannelVariableTarget(pipelineParam, item, fields)
    )
  }

  function pipelineDependencyParamCode(param: TektonDagTaskParam, field: string) {
    return (
      nodePipelineParam(param)?.config?.dependencyParamCodes?.find((dep) => dep.field === field)
        ?.paramCode || ''
    )
  }

  function setPipelineDependencyParamCode(
    param: TektonDagTaskParam,
    field: string,
    paramCode: string
  ) {
    const pipelineParam = ensureNodePipelineParam(param)
    const config = ensurePipelineParamConfig(param)
    if (!pipelineParam || !config) return
    const next = (config.dependencyParamCodes || []).filter((dep) => dep.field !== field)
    if (paramCode) next.push({ field, paramCode })
    config.dependencyParamCodes = next
    if (field === 'dynamic.project' || field === 'dynamic.repository') {
      config.projectParamCode = paramCode
    }
    if (
      tektonCodeRepoGitRefProjectDependency(param, field) &&
      tektonHasCodeRepoAddressDependency(param)
    ) {
      config.channelParamCode = ''
    }
    if (
      field === 'dynamic.image' ||
      field === 'dynamic.artifact' ||
      field === 'dynamic.artifactName'
    ) {
      config.componentParamCode = paramCode
    }
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  function firstPipelineMappingField(groupCode?: string, channelType?: string) {
    const type = channelTypes.value.find(
      (item) =>
        item.code === String(channelType || '').trim() &&
        item.groupCode === String(groupCode || '').trim()
    )
    return filterPipelineCodeRepoMappingOptions(
      groupCode,
      dedupePipelineMappingOptions([
        ...pipelineMappingOptions(groupCode, channelType),
        ...parsePipelineMappingFields(type?.mappingFields)
      ])
    )[0]?.field
  }

  async function handlePipelineChannelGroupChange(param: TektonDagTaskParam, value: string) {
    const pipelineParam = ensureNodePipelineParam(param)
    const config = ensurePipelineParamConfig(param)
    if (!pipelineParam || !config) return
    config.channelGroupCode = value
    config.channelTypeFilter = ''
    config.mappingField = ''
    config.channelParamCode = ''
    config.projectParamCode = ''
    config.componentParamCode = ''
    config.dependencyParamCodes = []
    await ensurePipelineMappingFields(value)
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  async function handlePipelineChannelTypeChange(param: TektonDagTaskParam, value: string) {
    const pipelineParam = ensureNodePipelineParam(param)
    const config = ensurePipelineParamConfig(param)
    if (!pipelineParam || !config) return
    config.channelTypeFilter = value
    config.channelParamCode = ''
    config.projectParamCode = ''
    config.componentParamCode = ''
    config.dependencyParamCodes = []
    await ensurePipelineMappingFields(config.channelGroupCode, value)
    config.mappingField = firstPipelineMappingField(config.channelGroupCode, value) || ''
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  function handlePipelineChannelMappingFieldChange(param: TektonDagTaskParam, value: string) {
    const pipelineParam = ensureNodePipelineParam(param)
    const config = ensurePipelineParamConfig(param)
    if (!pipelineParam || !config) return
    config.mappingField = value
    config.channelParamCode = ''
    config.projectParamCode = ''
    config.componentParamCode = ''
    config.dependencyParamCodes = []
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  async function handlePipelineMappingFieldVisible(param: TektonDagTaskParam, visible: boolean) {
    if (!visible) return
    await ensurePipelineMappingFields(
      pipelineParamConfigValue(param, 'channelGroupCode'),
      pipelineParamConfigValue(param, 'channelTypeFilter')
    )
  }

  function pipelineMappingFieldMetaText(param: TektonDagTaskParam) {
    const option = currentPipelineMappingFieldOption(param)
    if (!option) return ''
    const parts = [
      option.kind ? `字段类型：${option.kind}` : '',
      option.uiControl ? `控件：${option.uiControl}` : '',
      pipelineChannelParamDependencies(param).length
        ? `依赖 ${pipelineChannelParamDependencies(param).length} 个参数`
        : ''
    ].filter(Boolean)
    return parts.join('，')
  }

  function pipelineCredentialSourceOptions(param: TektonDagTaskParam) {
    const current = nodePipelineParam(param)
    return localConfig.params.filter((item) => {
      if (current?.name && item.name === current.name) return false
      if (item.paramType !== 'channelGroupCode') return false
      const field = String(item.config?.mappingField || '').trim()
      return field === 'endpoint' || field.startsWith('address.')
    })
  }

  function validatePipelineParamPlatformConfig(param: TektonDagTaskParam) {
    const pipelineParam = nodePipelineParam(param)
    if (!pipelineParam || !pipelineParamRequiresPlatformConfig(param)) return true
    sanitizePipelineParamConfigForType(pipelineParam)
    const config = pipelineParam.config || {}
    const name = `Pipeline 参数「${pipelineParam.name}」`
    switch (pipelineParam.paramType) {
      case 'channelGroupCode':
        if (!String(config.channelGroupCode || '').trim()) {
          ElMessage.warning(`${name}必须选择渠道分组`)
          return false
        }
        if (!String(config.channelTypeFilter || '').trim()) {
          ElMessage.warning(`${name}必须选择渠道类型`)
          return false
        }
        if (!String(config.mappingField || '').trim()) {
          ElMessage.warning(`${name}必须选择映射字段`)
          return false
        }
        if (
          pipelineNeedChannelBindingParam(param) &&
          currentPipelineMappingFieldOption(param)?.allowManualInput !== true
        ) {
          const channelParamCode = String(config.channelParamCode || '').trim()
          if (!channelParamCode && !tektonHasCodeRepoAddressDependency(param)) {
            ElMessage.warning(`${name}必须选择${pipelineChannelBindingLabel(param)}`)
            return false
          }
          if (
            channelParamCode &&
            !pipelineChannelVariableOptions(param).some((item) => item.name === channelParamCode)
          ) {
            ElMessage.warning(`${name}的${pipelineChannelBindingLabel(param)}不匹配`)
            return false
          }
        }
        for (const dep of pipelineChannelParamDependencies(param)) {
          const paramCode = pipelineDependencyParamCode(param, dep.field)
          if (
            paramCode &&
            !pipelineChannelDependencyOptions(param, dep.field).some(
              (item) => item.name === paramCode
            )
          ) {
            ElMessage.warning(`${name}的${dep.name || dep.field}不匹配`)
            return false
          }
        }
        break
      case 'voucherModel':
        if (!String(config.voucherModel || '').trim()) {
          ElMessage.warning(`${name}必须选择凭证类型`)
          return false
        }
        if (
          pipelineCredentialMode(param) === 'field' &&
          !String(config.mappingField || '').trim()
        ) {
          ElMessage.warning(`${name}必须选择凭证字段`)
          return false
        }
        break
      case 'channelCredential':
        if (!String(config.credentialSourceParamCode || '').trim()) {
          ElMessage.warning(`${name}必须选择渠道来源参数`)
          return false
        }
        if (
          !pipelineCredentialSourceOptions(param).some(
            (item) => item.name === config.credentialSourceParamCode
          )
        ) {
          ElMessage.warning(`${name}的渠道来源参数不匹配`)
          return false
        }
        if (
          pipelineCredentialMode(param) === 'field' &&
          !String(config.mappingField || '').trim()
        ) {
          ElMessage.warning(`${name}必须选择凭证字段`)
          return false
        }
        break
      case 'configCenter':
        if (!String(config.configTypeId || config.configTypeCode || '').trim()) {
          ElMessage.warning(`${name}必须配置配置中心类型`)
          return false
        }
        break
      case 'hostGroupTargets':
        if (
          !['normal', 'inventory', 'json', 'jsonBase64', 'ansible', 'ansibleBase64'].includes(
            String(config.renderMode || 'json')
          )
        ) {
          ElMessage.warning(`${name}主机组输出模式不支持`)
          return false
        }
        break
    }
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    return true
  }

  function tektonHasCodeRepoAddressDependency(param: TektonDagTaskParam) {
    const config = nodePipelineParam(param)?.config || {}
    return (config.dependencyParamCodes || []).some((dep) => {
      const paramCode = String(dep.paramCode || '').trim()
      if (!paramCode) return false
      return localConfig.params.some(
        (item) =>
          item.name === paramCode &&
          item.paramType === 'channelGroupCode' &&
          String(item.config?.channelGroupCode || '').trim() ===
            String(config.channelGroupCode || '').trim() &&
          String(item.config?.channelTypeFilter || '').trim() ===
            String(config.channelTypeFilter || '').trim() &&
          String(item.config?.mappingField || '').trim() === 'address.projectUrl'
      )
    })
  }

  function tektonUseCodeRepoAddressDependency(param: TektonDagTaskParam) {
    const mappingField = pipelineParamConfigValue(param, 'mappingField')
    return (
      pipelineParamConfigValue(param, 'channelGroupCode') === 'code_repo' &&
      ['dynamic.branch', 'dynamic.tag'].includes(mappingField) &&
      tektonHasCodeRepoAddressDependency(param)
    )
  }

  function pipelineParamNeedsOptions(param: TektonDagTaskParam) {
    const paramType = String(
      pipelineParamContractValue(param, 'paramType') || param.paramType || ''
    )
    return paramType === 'choice' || paramType === 'list'
  }

  function pipelineParamOptionRows(param: TektonDagTaskParam) {
    const pipelineParam = nodePipelineParam(param)
    const options = pipelineParam?.selectList || pipelineParam?.options
    if (options?.length) return options
    return (pipelineParam?.enum || []).map((value) => ({ label: value, value }))
  }

  function savePipelineParamOptions(
    param: TektonDagTaskParam,
    options: Array<{ label: string; value: string }>
  ) {
    const pipelineParam = ensureNodePipelineParam(param)
    if (!pipelineParam) return
    const next = options
      .map((item) => ({
        label: String(item.label || item.value || '').trim(),
        value: String(item.value || item.label || '').trim()
      }))
      .filter((item) => item.value)
    pipelineParam.options = next
    pipelineParam.selectList = next
    pipelineParam.enum = pipelineEnumFromOptions(next)
    pipelineParam.config = {
      ...(pipelineParam.config || {}),
      options: next
    }
    syncNodeParamMetaFromPipelineParam(param, pipelineParam)
    commit()
  }

  function addPipelineParamOption(param: TektonDagTaskParam) {
    savePipelineParamOptions(param, [
      ...pipelineParamOptionRows(param),
      { label: `选项 ${pipelineParamOptionRows(param).length + 1}`, value: '' }
    ])
  }

  function updatePipelineParamOption(
    param: TektonDagTaskParam,
    index: number,
    key: 'label' | 'value',
    value: string | number
  ) {
    const options = pipelineParamOptionRows(param).map((item) => ({ ...item }))
    if (!options[index]) return
    options[index][key] = String(value ?? '')
    savePipelineParamOptions(param, options)
  }

  function removePipelineParamOption(param: TektonDagTaskParam, index: number) {
    savePipelineParamOptions(
      param,
      pipelineParamOptionRows(param).filter((_, optionIndex) => optionIndex !== index)
    )
  }

  function nodeParamNumberValue(param: TektonDagTaskParam) {
    const value = String(param.value ?? param.defaultValue ?? '').trim()
    if (!value) return undefined
    const next = Number(value)
    return Number.isFinite(next) ? next : undefined
  }

  function updateNodeParamFixedValue(
    param: TektonDagTaskParam,
    value: string | number | boolean | undefined
  ) {
    param.value = value === undefined ? '' : String(value)
    commit()
  }

  function addNodeParam() {
    const missingRequirements = (activeNode.value?.paramRequirements || []).filter(
      (item) => !activeNode.value?.params.some((param) => param.name === item.name)
    )
    const requirement =
      missingRequirements.find((item) => isRequiredNodeParam(undefined, item)) ||
      missingRequirements[0]
    activeNode.value?.params.push(createNodeParamFromRequirement(requirement))
    commit()
  }

  function completeRequiredNodeParams() {
    const node = activeNode.value
    if (!node) return
    const missing = (node.paramRequirements || []).filter(
      (item) =>
        isRequiredNodeParam(undefined, item) &&
        !node.params.some((param) => param.name === item.name)
    )
    node.params.forEach((param) => {
      const requirement = node.paramRequirements?.find((item) => item.name === param.name)
      if (!requirement) return
      normalizeRuntimeParamSource(param)
      Object.assign(param, {
        sourceCode: requirement.sourceCode || param.sourceCode,
        displayName: requirement.displayName || param.displayName,
        defaultValue: requirement.defaultValue ?? param.defaultValue,
        description: requirement.description || param.description,
        required: requirement.required,
        readonly: requirement.readonly,
        paramType: requirement.paramType || param.paramType,
        options: requirement.options || requirement.selectList || param.options || [],
        runtimeConfig: requirement.runtimeConfig
      })
      if (isPipelineParamBinding(param)) {
        param.pipelineParam =
          param.pipelineParam ||
          ensurePipelineParam(param.name, paramMetaFromRequirement(requirement, param))
        applyPipelineParamMetaFromNodeParam(param, requirement)
      }
    })
    missing.forEach((requirement) => {
      node.params.push(createNodeParamFromRequirement(requirement))
    })
    commit()
    ElMessage.success(missing.length ? `已补齐 ${missing.length} 个必填参数` : '必填参数已完整')
  }

  function keepRequiredNodeParams() {
    const node = activeNode.value
    if (!node) return
    const removedParamNames: string[] = []
    const nextParams = node.params.filter((param) => {
      const requirement = node.paramRequirements?.find((item) => item.name === param.name)
      const keep = isRequiredNodeParam(param, requirement)
      if (!keep && param.name) removedParamNames.push(param.name)
      return keep
    })
    node.params.splice(0, node.params.length, ...nextParams)
    if (removedParamNames.length > 0) {
      const removed = new Set(removedParamNames)
      localConfig.edges.splice(
        0,
        localConfig.edges.length,
        ...localConfig.edges.filter(
          (edge) =>
            !(
              edge.type === 'result' &&
              edge.target === node.id &&
              removed.has(edge.targetParam || '')
            )
        )
      )
    }
    completeRequiredNodeParams()
  }

  function removeNodeParam(index: number) {
    const param = activeNode.value?.params[index]
    const requirement = activeNode.value?.paramRequirements?.find(
      (item) => item.name === param?.name
    )
    if (isRequiredNodeParam(param, requirement)) {
      ElMessage.warning('平台必填或只读参数不能删除')
      return
    }
    activeNode.value?.params.splice(index, 1)
    if (param?.name) removeResultEdgesByParam(param.name)
    commit()
  }

  function handleParamSourceChange(param: TektonDagTaskParam) {
    normalizeRuntimeParamSource(param)
    const requirement = activeNode.value?.paramRequirements?.find(
      (item) => item.name === param.name
    )
    if (isPipelineParamBinding(param)) {
      param.pipelineParam =
        param.pipelineParam ||
        ensurePipelineParam(param.name, paramMetaFromRequirement(requirement, param))
      applyPipelineParamMetaFromNodeParam(param, requirement)
      param.value = ''
    }
    if (param.source === 'result') {
      syncResultParamEdge(param)
      return
    }
    removeResultEdgesByParam(param.name)
    commit()
  }

  function handleNodeParamNameChange(param: TektonDagTaskParam) {
    normalizeRuntimeParamSource(param)
    const requirement = activeNode.value?.paramRequirements?.find(
      (item) => item.name === param.name
    )
    if (requirement) {
      param.displayName = requirement.displayName
      param.required = requirement.required
      param.defaultValue = requirement.defaultValue
      param.runtimeConfig = requirement.runtimeConfig
      param.sourceCode = requirement.sourceCode || requirement.name
      param.description = requirement.description
      param.readonly = requirement.readonly
      param.paramType = requirement.paramType || platformParamTypeFromTektonType(requirement.type)
      param.options = requirement.options || requirement.selectList || []
      param.selectList = requirement.selectList || requirement.options || []
      param.allowCustom = requirement.allowCustom
      param.config = requirement.config
      if (requirement.runtimeConfig) {
        param.source = 'pipelineParam'
        param.runtimeConfig = true
      }
    }
    if (isPipelineParamBinding(param)) {
      param.pipelineParam =
        param.pipelineParam ||
        ensurePipelineParam(param.name, paramMetaFromRequirement(requirement, param))
      applyPipelineParamMetaFromNodeParam(param, requirement)
      if (param.runtimeConfig) markPipelineParamRuntime(param.pipelineParam)
    }
    if (param.source === 'result') {
      syncResultParamEdge(param)
      return
    }
    pruneStaleResultEdgesForNode()
    commit()
  }

  function handlePipelineParamSelect(param: TektonDagTaskParam) {
    normalizeRuntimeParamSource(param)
    const requirement = activeNode.value?.paramRequirements?.find(
      (item) => item.name === param.name
    )
    applyPipelineParamMetaFromNodeParam(param, requirement)
    if (param.runtimeConfig) markPipelineParamRuntime(param.pipelineParam)
    commit()
  }

  function applyPipelineParamMetaFromNodeParam(
    param: TektonDagTaskParam,
    requirement?: TektonDagParamRequirement
  ) {
    normalizeRuntimeParamSource(param)
    if (!param.pipelineParam || !isPipelineParamBinding(param)) return
    ensurePipelineParam(param.pipelineParam, paramMetaFromRequirement(requirement, param))
  }

  function syncResultParamEdge(param: TektonDagTaskParam) {
    const target = activeNode.value
    if (!target || param.source !== 'result') {
      commit()
      return
    }
    pruneStaleResultEdgesForNode(target)
    if (!param.resultNodeId || !param.resultName || !param.name) {
      commit()
      return
    }
    const edge = localConfig.edges.find(
      (item) =>
        item.type === 'result' && item.target === target.id && item.targetParam === param.name
    )
    if (edge) {
      edge.source = param.resultNodeId
      edge.resultName = param.resultName
    } else {
      localConfig.edges.push({
        id: `edge-${param.resultNodeId}-${target.id}-${Date.now()}`,
        source: param.resultNodeId,
        target: target.id,
        type: 'result',
        resultName: param.resultName,
        targetParam: param.name
      })
    }
    commit()
  }

  function pruneStaleResultEdgesForNode(node = activeNode.value) {
    if (!node) return
    const resultParamNames = new Set(
      node.params
        .filter((param) => param.source === 'result' && param.name)
        .map((param) => param.name)
    )
    const nextEdges = localConfig.edges.filter(
      (edge) =>
        !(
          edge.type === 'result' &&
          edge.target === node.id &&
          (!edge.targetParam || !resultParamNames.has(edge.targetParam))
        )
    )
    localConfig.edges.splice(0, localConfig.edges.length, ...nextEdges)
  }

  function removeResultEdgesByParam(paramName: string) {
    const target = activeNode.value
    if (!target) return
    const nextEdges = localConfig.edges.filter(
      (edge) =>
        !(edge.type === 'result' && edge.target === target.id && edge.targetParam === paramName)
    )
    localConfig.edges.splice(0, localConfig.edges.length, ...nextEdges)
  }

  function addIncomingEdge() {
    const target = activeNode.value
    const source = resultSourceOptions.value[0]?.id
    if (!target || !source) return
    localConfig.edges.push({
      id: `edge-${source}-${target.id}-${Date.now()}`,
      source,
      target: target.id,
      type: 'runAfter'
    })
    commit()
  }

  function removeEdge(edgeId: string) {
    const index = localConfig.edges.findIndex((edge) => edge.id === edgeId)
    if (index === -1) return
    localConfig.edges.splice(index, 1)
    commit()
  }

  function handleEdgeTypeChange(edge: TektonDagEdge) {
    if (edge.type === 'result') {
      edge.resultName = edge.resultName || resultNameOptions(edge.source)[0] || ''
      edge.targetParam = edge.targetParam || activeNode.value?.params[0]?.name || ''
      edge.whenInput = ''
      edge.whenValues = []
    } else if (edge.type === 'when') {
      edge.resultName = ''
      edge.targetParam = ''
      edge.whenInput = edge.whenInput || '$(params.branch)'
      edge.whenOperator = edge.whenOperator || 'in'
      edge.whenValues = edge.whenValues?.length ? edge.whenValues : ['main']
    } else {
      edge.resultName = ''
      edge.targetParam = ''
      edge.whenInput = ''
      edge.whenValues = []
    }
    commit()
  }

  function updateEdgeWhenValues(edge: TektonDagEdge, value: string) {
    edge.whenValues = value
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
    commit()
  }

  function addNodeWorkspace() {
    if (activeNode.value) dedupeNodeWorkspaceMappings(activeNode.value)
    const requirement = activeNode.value?.workspaceRequirements?.find(
      (item) =>
        !activeNode.value?.workspaces.some((workspace) => workspace.taskWorkspace === item.name)
    )
    activeNode.value?.workspaces.push(createWorkspaceMappingFromRequirement(requirement))
    commit()
  }

  function createWorkspaceMappingFromRequirement(
    requirement?: TektonDagWorkspaceRequirement
  ): TektonDagWorkspaceMapping {
    if (!requirement?.name) {
      return { taskWorkspace: '', pipelineWorkspace: '' }
    }
    const pipelineWorkspace = ensurePipelineWorkspace(requirement.name, requirement.optional, {
      description: requirement.description,
      bindingType: requirement.requiredBindingType || 'emptyDir'
    })
    return {
      taskWorkspace: requirement.name,
      pipelineWorkspace,
      optional: requirement.optional,
      description: requirement.description,
      requiredBindingType: requirement.requiredBindingType,
      bindingParamCode: requirement.bindingParamCode
    }
  }

  function dedupeNodeWorkspaceMappings(node: TektonDagNode) {
    const seen = new Set<string>()
    const next = node.workspaces.filter((workspace) => {
      const key = String(workspace.taskWorkspace || '').trim()
      if (!key) return true
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
    if (next.length !== node.workspaces.length) {
      node.workspaces.splice(0, node.workspaces.length, ...next)
    }
  }

  function isRequiredNodeWorkspace(
    workspace?: TektonDagWorkspaceMapping,
    requirement?: TektonDagWorkspaceRequirement
  ) {
    if (requirement)
      return requirement.optional !== true || Boolean(requirement.requiredBindingType)
    if (!workspace?.taskWorkspace) return false
    return workspace.optional !== true || Boolean(workspace.requiredBindingType)
  }

  function requiredNodeWorkspaceCount(node?: TektonDagNode) {
    return requiredNodeWorkspaceNames(node).size
  }

  function requiredNodeWorkspaceNames(node?: TektonDagNode) {
    const names = new Set<string>()
    if (!node) return names
    node.workspaces.forEach((workspace) => {
      const requirement = node.workspaceRequirements?.find(
        (item) => item.name === workspace.taskWorkspace
      )
      if (isRequiredNodeWorkspace(workspace, requirement)) names.add(workspace.taskWorkspace)
    })
    ;(node.workspaceRequirements || []).forEach((requirement) => {
      if (isRequiredNodeWorkspace(undefined, requirement)) names.add(requirement.name)
    })
    return names
  }

  function configuredRequiredNodeWorkspaceCount(node?: TektonDagNode) {
    if (!node) return 0
    return Array.from(requiredNodeWorkspaceNames(node)).filter((name) => {
      const workspace = node.workspaces.find((item) => item.taskWorkspace === name)
      return Boolean(workspace?.pipelineWorkspace)
    }).length
  }

  function nodeRequiredParamSummary(node?: TektonDagNode) {
    if (!node) return '参数 0/0'
    return `参数 ${configuredRequiredNodeParamCount(node)}/${requiredNodeParamCount(node)}`
  }

  function nodeRequiredWorkspaceSummary(node?: TektonDagNode) {
    if (!node) return 'Workspace 0/0'
    return `Workspace ${configuredRequiredNodeWorkspaceCount(node)}/${requiredNodeWorkspaceCount(
      node
    )}`
  }

  function nodeNextConfigTab(node: TektonDagNode) {
    if (issueByTarget.value[node.id]) return preferredNodeConfigTab(node)
    if (configuredRequiredNodeWorkspaceCount(node) < requiredNodeWorkspaceCount(node)) {
      return 'workspaces'
    }
    if (configuredRequiredNodeParamCount(node) < requiredNodeParamCount(node)) return 'params'
    if (node.workspaceRequirements?.length || node.workspaces.length > 0) return 'workspaces'
    if (node.params.length > 0) return 'params'
    return 'basic'
  }

  function nodeGuideDescription(node: TektonDagNode) {
    const issue = issueByTarget.value[node.id]
    if (issue) return `先处理：${issue}`
    if (configuredRequiredNodeWorkspaceCount(node) < requiredNodeWorkspaceCount(node)) {
      return '下一步确认 Workspace 映射和绑定方式。'
    }
    if (configuredRequiredNodeParamCount(node) < requiredNodeParamCount(node)) {
      return '下一步确认参数来源和值。'
    }
    if (node.workspaceRequirements?.length || node.workspaces.length > 0) {
      return '先检查 Workspace 绑定，再继续添加下一步或保存。'
    }
    if (node.params.length > 0) return '先检查参数来源，再继续添加下一步或保存。'
    return '基础信息已就绪，可以继续添加下一步或保存。'
  }

  function nodeCardHint(node: TektonDagNode) {
    if (issueByTarget.value[node.id]) return '有待处理项，点击配置'
    if (node.workspaceRequirements?.length || node.workspaces.length > 0)
      return '下一步配置 Workspace'
    if (node.params.length > 0) return '下一步配置参数'
    return '点击配置节点'
  }

  function nodeConfigStatusText(node: TektonDagNode) {
    if (issueByTarget.value[node.id]) return '待处理'
    return '可保存'
  }

  function completeRequiredNodeWorkspaces() {
    const node = activeNode.value
    if (!node) return
    dedupeNodeWorkspaceMappings(node)
    const missing = (node.workspaceRequirements || []).filter(
      (item) =>
        (item.optional !== true || Boolean(item.requiredBindingType)) &&
        !node.workspaces.some((workspace) => workspace.taskWorkspace === item.name)
    )
    node.workspaces.forEach((workspace) => {
      const requirement = node.workspaceRequirements?.find(
        (item) => item.name === workspace.taskWorkspace
      )
      if (!requirement) return
      workspace.optional = requirement.optional
      workspace.description = requirement.description
      workspace.requiredBindingType = requirement.requiredBindingType
      workspace.bindingParamCode = requirement.bindingParamCode
      workspace.pipelineWorkspace =
        workspace.pipelineWorkspace ||
        ensurePipelineWorkspace(requirement.name, requirement.optional, {
          description: requirement.description,
          bindingType: requirement.requiredBindingType || 'emptyDir'
        })
      enforceWorkspaceRequiredBinding(workspace)
    })
    missing.forEach((requirement) => {
      node.workspaces.push(createWorkspaceMappingFromRequirement(requirement))
    })
    commit()
    ElMessage.success(
      missing.length ? `已补齐 ${missing.length} 个 Workspace` : '必填 Workspace 已完整'
    )
  }

  function keepRequiredNodeWorkspaces() {
    const node = activeNode.value
    if (!node) return
    const nextWorkspaces = node.workspaces.filter((workspace) => {
      const requirement = node.workspaceRequirements?.find(
        (item) => item.name === workspace.taskWorkspace
      )
      return isRequiredNodeWorkspace(workspace, requirement)
    })
    node.workspaces.splice(0, node.workspaces.length, ...nextWorkspaces)
    completeRequiredNodeWorkspaces()
  }

  function handleNodeWorkspaceNameChange(workspace: TektonDagWorkspaceMapping) {
    const requirement = activeNode.value?.workspaceRequirements?.find(
      (item) => item.name === workspace.taskWorkspace
    )
    if (requirement) {
      workspace.optional = requirement.optional
      workspace.description = requirement.description
      workspace.requiredBindingType = requirement.requiredBindingType
      workspace.bindingParamCode = requirement.bindingParamCode
      workspace.pipelineWorkspace =
        workspace.pipelineWorkspace ||
        ensurePipelineWorkspace(requirement.name, requirement.optional, {
          description: requirement.description,
          bindingType: requirement.requiredBindingType || 'emptyDir'
        })
    }
    enforceWorkspaceRequiredBinding(workspace)
    if (activeNode.value) dedupeNodeWorkspaceMappings(activeNode.value)
    commit()
  }

  function handlePipelineWorkspaceSelect(workspace: TektonDagWorkspaceMapping) {
    enforceWorkspaceRequiredBinding(workspace)
    commit()
  }

  function pipelineWorkspaceByName(name?: string) {
    return localConfig.workspaces.find((item) => item.name === name)
  }

  function pipelineWorkspaceBindingType(name?: string) {
    return pipelineWorkspaceByName(name)?.bindingType || 'emptyDir'
  }

  function effectivePipelineWorkspaceBindingType(name?: string) {
    const bindingType = pipelineWorkspaceBindingType(name)
    const workspaceStrategy = localConfig.runPolicy?.workspaceStrategy
    return bindingType === 'emptyDir' && workspaceStrategy ? workspaceStrategy : bindingType
  }

  function enforceWorkspaceRequiredBinding(workspace: TektonDagWorkspaceMapping) {
    if (!workspace.requiredBindingType || !workspace.pipelineWorkspace) return
    const pipelineWorkspace = pipelineWorkspaceByName(workspace.pipelineWorkspace)
    if (pipelineWorkspace) pipelineWorkspace.bindingType = workspace.requiredBindingType
  }

  function workspaceCapabilityText(type?: TektonWorkspaceBindingType) {
    if (type === 'emptyDir') return 'EmptyDir'
    if (type === 'persistentVolumeClaim') return 'PVC'
    if (type === 'volumeClaimTemplate') return '动态 PVC'
    if (type === 'secret') return 'Secret'
    if (type === 'generatedSecret') return '平台生成 Secret'
    if (type === 'configMap') return 'ConfigMap'
    if (type === 'projected') return 'Projected'
    return '存储'
  }

  function removeNodeWorkspace(index: number) {
    const workspace = activeNode.value?.workspaces[index]
    const requirement = activeNode.value?.workspaceRequirements?.find(
      (item) => item.name === workspace?.taskWorkspace
    )
    if (isRequiredNodeWorkspace(workspace, requirement)) {
      ElMessage.warning('平台必填 Workspace 不能删除')
      return
    }
    activeNode.value?.workspaces.splice(index, 1)
    commit()
  }

  function addWhen() {
    activeNode.value?.when.push({ input: '', operator: 'in', values: [] })
    commit()
  }

  function removeWhen(index: number) {
    activeNode.value?.when.splice(index, 1)
    commit()
  }

  function updateWhenValues(index: number, value: string) {
    const item = activeNode.value?.when[index]
    if (!item) return
    item.values = value
      .split(',')
      .map((part) => part.trim())
      .filter(Boolean)
    commit()
  }

  function enableMatrix() {
    if (!activeNode.value) return
    activeNode.value.matrix = { params: [{ name: '', values: [] }], include: [] }
    commit()
  }

  function disableMatrix() {
    if (!activeNode.value) return
    activeNode.value.matrix = undefined
    commit()
  }

  function addMatrixParam() {
    activeNode.value?.matrix?.params.push({ name: '', values: [] })
    commit()
  }

  function removeMatrixParam(index: number) {
    activeNode.value?.matrix?.params.splice(index, 1)
    commit()
  }

  function updateMatrixValues(index: number, value: string) {
    const item = activeNode.value?.matrix?.params[index]
    if (!item) return
    item.values = value
      .split(',')
      .map((part) => part.trim())
      .filter(Boolean)
    commit()
  }

  function addMatrixInclude() {
    if (!activeNode.value?.matrix) return
    if (!activeNode.value.matrix.include) activeNode.value.matrix.include = []
    activeNode.value.matrix.include.push({ params: [] })
    commit()
  }

  function removeMatrixInclude(index: number) {
    activeNode.value?.matrix?.include?.splice(index, 1)
    commit()
  }

  function updateMatrixIncludeParams(index: number, value: string) {
    const item = activeNode.value?.matrix?.include?.[index]
    if (!item) return
    item.params = parseNameValueList(value)
    commit()
  }

  function addResolverParam() {
    const node = activeNode.value
    if (!node) return
    if (!node.taskRef) node.taskRef = { kind: 'Task', name: '' }
    if (!node.taskRef.params) node.taskRef.params = []
    node.taskRef.params.push({ name: '', value: '' })
    commit()
  }

  function removeResolverParam(index: number) {
    activeNode.value?.taskRef.params?.splice(index, 1)
    commit()
  }

  function taskRefKindText(node: TektonDagNode) {
    if (node.pipelineRef?.resolver) return `Pipeline Resolver / ${node.pipelineRef.resolver}`
    if (node.pipelineRef?.name) return 'PipelineRef'
    if (node.pipelineSpec) return 'PipelineSpec'
    if (node.taskSpec) return 'TaskSpec'
    if (node.taskRef?.resolver) return `Resolver / ${node.taskRef.resolver}`
    return node.taskRef?.kind || '-'
  }

  function taskRefNamespaceText(node: TektonDagNode) {
    if (node.pipelineRef || node.pipelineSpec) return '随引用 Pipeline 解析'
    if (node.taskRef?.kind === 'ClusterTask') return '集群级'
    return node.taskRef?.namespace || localConfig.pipeline.namespace || '-'
  }

  function taskNamespaceSourceText(node: TektonDagNode) {
    if (node.pipelineRef || node.pipelineSpec) return 'PipelineTask 引用 Pipeline'
    if (node.taskRef?.kind === 'ClusterTask') return 'ClusterTask 不绑定 Namespace'
    if (node.taskRef?.namespace) return 'Task 资源声明的 Namespace'
    if (localConfig.pipeline.namespace) return '来自构建渠道解析的 Pipeline Namespace'
    return '保存项目流水线时由渠道解析'
  }

  function nodeReferenceName(node: TektonDagNode) {
    return (
      node.taskCode ||
      node.taskId ||
      node.pipelineRef?.name ||
      node.taskRef?.name ||
      (node.pipelineSpec ? 'inline pipelineSpec' : '') ||
      (node.taskSpec ? 'inline taskSpec' : '') ||
      '-'
    )
  }

  function nodeFailurePolicyText(node: TektonDagNode) {
    return node.onError === 'continue' ? '失败后继续' : '失败终止 Pipeline'
  }

  function resultNameOptions(nodeId?: string) {
    return resultSourceOptions.value.find((item) => item.id === nodeId)?.results || []
  }

  function formatNameValueList(items?: TektonNameValue[]) {
    return (items || []).map((item) => `${item.name}=${item.value}`).join(',')
  }

  function parseNameValueList(value: string): TektonNameValue[] {
    return value
      .split(',')
      .map((part) => {
        const [name, ...rest] = part.split('=')
        return { name: name.trim(), value: rest.join('=').trim() }
      })
      .filter((item) => item.name)
  }
</script>

<style scoped lang="scss">
  .tekton-dag-designer {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
  }

  .dag-canvas-panel {
    flex: 1;
    min-height: 0;
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    background: var(--el-bg-color);
  }

  .dag-library,
  .dag-inspector {
    display: flex;
    flex-direction: column;
    gap: 12px;
    overflow: hidden;
    padding: 12px;
  }

  .panel-head,
  .dag-toolbar,
  .section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .panel-head span,
  .dag-title span {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .task-filter-bar {
    display: grid;
    grid-template-columns: 220px minmax(240px, 1fr);
    gap: 10px;
    align-items: center;
  }

  .task-list {
    display: flex;
    min-height: 0;
    flex: 1;
    flex-direction: column;
    gap: 8px;
    overflow: auto;
  }

  .task-list.is-empty {
    align-items: center;
    justify-content: center;
    min-height: 260px;
  }

  .task-card {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr);
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 78px;
    border: 1px solid var(--task-color-border, var(--el-border-color-light));
    border-radius: 8px;
    background:
      linear-gradient(90deg, var(--task-color-soft, transparent), transparent 62%),
      var(--el-bg-color);
    padding: 12px 14px;
    text-align: left;
    cursor: pointer;
    transition:
      border-color 0.16s ease,
      box-shadow 0.16s ease,
      transform 0.16s ease;
  }

  .task-card:not(:disabled):hover {
    border-color: var(--task-color, var(--el-color-primary));
    box-shadow: 0 8px 24px rgb(37 99 235 / 10%);
    transform: translateY(-1px);
  }

  .task-card:not(:disabled):focus-visible {
    outline: 2px solid var(--task-color, var(--el-color-primary));
    outline-offset: 2px;
  }

  .task-card:disabled {
    cursor: not-allowed;
    opacity: 0.64;
  }

  .task-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    color: var(--task-color, var(--el-color-primary));
    font-size: 23px;
    background: var(--task-color-soft, var(--el-fill-color-light));
    border: 1px solid var(--task-color-border, var(--el-border-color-lighter));
    border-radius: 8px;
  }

  .task-main {
    min-width: 0;
  }

  .task-main strong,
  .task-main em {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .task-main em {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    font-style: normal;
  }

  .dag-canvas-panel {
    display: flex;
    position: relative;
    min-width: 0;
    flex-direction: column;
    overflow: hidden;
  }

  .dag-toolbar {
    position: relative;
    z-index: 4;
    flex-wrap: wrap;
    align-items: flex-start;
    padding: 10px 12px;
    border-bottom: 1px solid var(--el-border-color-light);
  }

  .dag-title {
    display: grid;
    flex: 1 1 240px;
    min-width: 0;
    gap: 4px;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .dag-actions {
    display: flex;
    flex: 1 1 420px;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: 8px;
    min-width: 0;
  }

  .dag-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .dag-config-guide {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    background: var(--el-fill-color-extra-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .dag-config-guide.invalid {
    background: #fff7f7;
  }

  .dag-config-guide-main {
    display: grid;
    flex: 1 1 260px;
    min-width: 0;
    gap: 4px;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 13px;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .dag-config-guide-status,
  .dag-config-guide-actions {
    display: inline-flex;
    flex: 0 1 auto;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  .dag-config-guide-status span {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-height: 24px;
    padding: 0 8px;
    color: var(--el-text-color-regular);
    font-size: 12px;
    line-height: 1;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .dag-config-guide-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .dag-validate-badge {
    :deep(.el-badge__content) {
      top: 2px;
      right: 8px;
    }
  }

  .dag-flow {
    margin: 12px;
    min-height: 0;
    flex: 1;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background-image:
      linear-gradient(var(--el-border-color-lighter) 1px, transparent 1px),
      linear-gradient(90deg, var(--el-border-color-lighter) 1px, transparent 1px);
    background-size: 24px 24px;
  }

  .dag-flow :deep(.vue-flow__renderer),
  .dag-flow :deep(.vue-flow__transformationpane),
  .dag-flow :deep(.vue-flow__nodes) {
    overflow: visible;
  }

  .dag-flow :deep(.vue-flow__node) {
    overflow: visible;
    z-index: 1;
  }

  .dag-flow :deep(.vue-flow__node:hover),
  .dag-flow :deep(.vue-flow__node.selected) {
    z-index: 20 !important;
  }

  .dag-node {
    position: relative;
    width: 280px;
    border: 1px solid var(--task-color-border, var(--el-border-color-light));
    border-left: 4px solid var(--task-color, var(--el-color-primary));
    border-radius: 8px;
    background: var(--el-bg-color);
    box-shadow: 0 6px 16px rgb(15 23 42 / 7%);
    padding: 12px 14px;
    cursor: pointer;
    transition:
      border-color 0.16s ease,
      box-shadow 0.16s ease,
      transform 0.16s ease;
  }

  .dag-node::after {
    position: absolute;
    inset: -5px;
    pointer-events: none;
    content: '';
    border-radius: 12px;
    box-shadow: 0 0 0 3px var(--task-color-soft, rgb(37 99 235 / 10%));
    opacity: 0;
    transition: opacity 0.16s ease;
  }

  .dag-node:hover {
    border-color: var(--task-color, var(--el-color-primary));
    box-shadow:
      0 0 0 2px var(--task-color-soft, rgb(37 99 235 / 10%)),
      0 8px 18px rgb(15 23 42 / 8%);
    transform: translateY(-1px);
  }

  .dag-node:hover::after,
  .dag-node.active::after {
    opacity: 1;
  }

  .dag-node.active {
    border-color: var(--el-color-primary);
    box-shadow:
      0 0 0 2px rgb(37 99 235 / 18%),
      0 8px 18px rgb(15 23 42 / 8%);
  }

  .dag-node.invalid {
    border-color: var(--el-color-danger);
    background: #fff7f7;
  }

  .dag-node.type-finally {
    border-color: var(--el-color-warning);
  }

  .dag-node-head {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }

  .dag-node-icon {
    display: inline-flex;
    flex: none;
    align-items: center;
    justify-content: center;
    width: 38px;
    height: 38px;
    font-size: 22px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .dag-node-title {
    min-width: 0;
    flex: 1;

    strong {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      font-size: 15px;
    }

    em {
      display: block;
      margin-top: 3px;
      overflow: hidden;
      color: var(--el-text-color-secondary);
      font-size: 12px;
      font-style: normal;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .dag-node-config {
    flex: none;
    height: 28px;
    padding: 0 10px;
    color: var(--task-color, var(--el-color-primary));
    font-size: 12px;
    line-height: 26px;
    cursor: pointer;
    background: var(--task-color-soft, var(--el-color-primary-light-9));
    border: 1px solid var(--task-color-border, var(--el-color-primary-light-7));
    border-radius: 6px;
  }

  .dag-node-config:hover {
    color: #fff;
    background: var(--task-color, var(--el-color-primary));
    border-color: var(--task-color, var(--el-color-primary));
  }

  .dag-node-warning {
    display: block;
    margin-top: 8px;
    overflow: hidden;
    color: var(--el-color-danger);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .dag-node-foot {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 8px;

    span {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-height: 22px;
      padding: 0 7px;
      color: var(--el-text-color-secondary);
      font-size: 12px;
      line-height: 1;
      background: var(--el-fill-color-light);
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 6px;
    }

    span:first-child {
      color: var(--el-color-warning);
      background: var(--el-color-warning-light-9);
      border-color: var(--el-color-warning-light-7);
    }

    span.done:first-child {
      color: var(--el-color-success);
      background: var(--el-color-success-light-9);
      border-color: var(--el-color-success-light-7);
    }
  }

  .dag-node-add {
    position: absolute;
    top: 50%;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    color: var(--el-color-primary);
    font-size: 14px;
    font-weight: 700;
    pointer-events: auto;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-color-primary-light-5);
    border-radius: 50%;
    box-shadow: 0 8px 18px rgb(37 99 235 / 14%);
    opacity: 0.88;
    transition:
      color 0.16s ease,
      background 0.16s ease,
      border-color 0.16s ease,
      opacity 0.16s ease,
      transform 0.16s ease;
    transform: translateY(-50%);
  }

  .dag-node-add:hover {
    color: #fff;
    background: var(--el-color-primary);
    border-color: var(--el-color-primary);
  }

  .dag-node:hover .dag-node-add,
  .dag-node.active .dag-node-add {
    opacity: 1;
    transform: translateY(-50%) scale(1.04);
  }

  .dag-node-add-left {
    left: -13px;
  }

  .dag-node-add-right {
    right: -13px;
  }

  .dag-context-menu {
    position: absolute;
    z-index: 60;
    display: grid;
    min-width: 150px;
    padding: 6px;
    background: var(--el-bg-color-overlay);
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    box-shadow: var(--el-box-shadow);

    button {
      min-height: 30px;
      padding: 0 10px;
      color: var(--el-text-color-primary);
      text-align: left;
      cursor: pointer;
      background: transparent;
      border: 0;
      border-radius: 4px;

      &:hover {
        color: var(--el-color-primary);
        background: var(--el-fill-color-light);
      }

      &.danger {
        color: var(--el-color-danger);
      }
    }
  }

  .dag-empty-state {
    position: absolute;
    top: 50%;
    left: 50%;
    display: grid;
    width: 320px;
    transform: translate(-50%, -50%);
    gap: 10px;
    justify-items: center;
    border: 1px solid var(--el-border-color-light);
    border-radius: 6px;
    background: var(--el-bg-color-overlay);
    box-shadow: var(--el-box-shadow-light);
    padding: 22px;
    text-align: center;
    z-index: 5;
  }

  .dag-empty-state span {
    color: var(--el-text-color-secondary);
    font-size: 13px;
    line-height: 1.5;
  }

  .inspector-form {
    min-height: 0;
    overflow: auto;
    padding-right: 4px;
  }

  .inspector-form :deep(.el-form-item) {
    margin-bottom: 12px;
  }

  .row-form-item {
    margin-bottom: 0 !important;
  }

  .inspector-form :deep(.el-select),
  .inspector-form :deep(.el-input-number) {
    width: 100%;
  }

  .inspector-section {
    display: grid;
    gap: 10px;
    margin-top: 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-blank);
    padding: 12px;
  }

  .section-title {
    display: flex;
    align-items: center;
    justify-content: space-between;
    min-height: 28px;
  }

  .section-heading {
    display: grid;
    min-width: 0;
    gap: 3px;

    strong,
    span {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .section-actions {
    display: inline-flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
    justify-content: flex-end;
  }

  .node-config-tabs {
    min-height: 0;

    :deep(.el-tabs__header) {
      margin: 0 0 12px;
    }

    :deep(.el-tabs__content) {
      overflow: visible;
    }
  }

  .node-config-hero {
    display: grid;
    grid-template-columns: minmax(220px, 0.9fr) minmax(420px, 1.6fr);
    gap: 12px;
    align-items: stretch;
    margin-bottom: 14px;
    padding: 14px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    background: linear-gradient(135deg, var(--el-fill-color-lighter), var(--el-bg-color));
  }

  .node-config-hero-main {
    display: grid;
    align-content: center;
    gap: 4px;
    min-width: 0;

    strong,
    span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      color: var(--el-text-color-primary);
      font-size: 18px;
      font-weight: 700;
    }

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .node-config-eyebrow {
    color: var(--el-color-primary) !important;
    font-weight: 600;
  }

  .node-config-hero-meta {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 10px;

    div {
      min-width: 0;
      padding: 10px 12px;
      border: 1px solid var(--el-border-color-lighter);
      border-radius: 6px;
      background: var(--el-bg-color);
    }

    span,
    strong,
    em {
      display: block;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span,
    em {
      color: var(--el-text-color-secondary);
      font-size: 12px;
      font-style: normal;
    }

    strong {
      margin: 4px 0;
      color: var(--el-text-color-primary);
      font-size: 13px;
    }
  }

  .field-label-with-help,
  .title-with-help {
    display: inline-flex;
    gap: 5px;
    align-items: center;
  }

  .help-dot {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    border: 1px solid var(--el-border-color);
    border-radius: 50%;
    color: var(--el-text-color-secondary);
    font-size: 11px;
    line-height: 1;
    cursor: help;
  }

  .form-grid {
    display: grid;
    gap: 12px;

    &.two-columns {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }

    &.three-columns {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  .readonly-grid {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .readonly-item {
    min-width: 0;
    padding: 10px 12px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;

    span,
    strong {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      margin-bottom: 6px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }

    strong {
      font-size: 13px;
      color: var(--el-text-color-primary);
    }
  }

  .inline-row,
  .param-row,
  .pipeline-param-row,
  .pipeline-result-row,
  .workspace-row,
  .workspace-extra-row,
  .when-row,
  .edge-row,
  .matrix-row,
  .matrix-include-row,
  .policy-row,
  .workspace-map-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
    gap: 8px;
    align-items: center;
  }

  .param-row,
  .when-row,
  .edge-row {
    grid-template-columns: repeat(auto-fit, minmax(170px, 1fr));
  }

  .workspace-map-row {
    grid-template-columns: minmax(160px, 1fr) minmax(200px, 1.3fr) minmax(160px, 1fr) auto;
  }

  .workspace-subpath-field {
    display: grid;
    gap: 4px;

    span {
      color: var(--el-text-color-secondary);
      font-size: 12px;
      line-height: 1;
    }
  }

  .pipeline-summary-list {
    display: grid;
    gap: 8px;
  }

  .pipeline-meta-list {
    display: grid;
    gap: 8px;
  }

  .pipeline-meta-row {
    display: grid;
    grid-template-columns: 86px minmax(180px, 0.9fr) minmax(220px, 1.1fr) auto;
    gap: 8px;
    align-items: center;
    padding: 8px 10px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .pipeline-result-list {
    display: grid;
    gap: 8px;
  }

  .pipeline-param-row {
    grid-template-columns: minmax(180px, 1fr) minmax(180px, 1fr) minmax(160px, 1.2fr);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    padding: 8px;
  }

  .pipeline-result-row {
    grid-template-columns: minmax(130px, 0.8fr) 110px minmax(260px, 1.4fr) minmax(160px, 1fr) auto;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    padding: 8px;
  }

  .result-actions {
    display: inline-flex;
    align-items: center;
    justify-content: flex-end;
    gap: 4px;
  }

  .pipeline-summary-main {
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
      font-size: 13px;
    }

    span {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .pipeline-summary-tags,
  .workspace-policy-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    min-width: 0;
  }

  .pipeline-summary-desc {
    min-width: 0;
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .workspace-row {
    grid-template-columns:
      minmax(170px, 1.2fr) minmax(150px, 0.8fr) minmax(140px, 0.8fr) minmax(180px, 1fr)
      auto;
  }

  .workspace-item {
    display: grid;
    gap: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    padding: 8px;
  }

  .workspace-card-actions {
    display: inline-flex;
    justify-content: flex-end;
  }

  .workspace-limit-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    min-width: 0;
  }

  .workspace-editor-form,
  .workspace-editor-grid {
    display: grid;
    gap: 12px;
  }

  .workspace-editor-title {
    display: grid;
    gap: 4px;
    padding: 10px 12px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);

    strong,
    span {
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
      color: var(--el-text-color-secondary);
      font-size: 12px;
    }
  }

  .workspace-editor-grid {
    grid-template-columns: repeat(2, minmax(260px, 1fr));
    align-items: end;

    :deep(.el-form-item) {
      margin-bottom: 0;
    }
  }

  .workspace-editor-wide {
    grid-column: 1 / -1;
  }

  .node-param-list {
    display: grid;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-bg-color);
  }

  .node-param-item {
    display: grid;
    grid-template-columns: minmax(260px, 1.4fr) minmax(220px, 1fr) auto;
    gap: 10px;
    align-items: center;
    min-width: 0;
    padding: 10px 12px;
    border-bottom: 1px solid var(--el-border-color-lighter);

    &:last-child {
      border-bottom: 0;
    }

    &.required {
      border-color: var(--el-color-primary-light-5);
      background: var(--el-color-primary-light-9);
    }
  }

  .node-param-main {
    display: grid;
    gap: 5px;
    min-width: 0;
  }

  .node-param-item-head {
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto auto;
    align-items: center;
    gap: 8px;
    min-width: 0;
  }

  .node-param-title {
    display: grid;
    grid-template-columns: auto minmax(0, max-content) minmax(80px, 1fr);
    min-width: 0;
    align-items: center;
    gap: 5px;

    i {
      color: var(--el-color-danger);
      font-style: normal;
    }

    strong {
      overflow: hidden;
      color: var(--el-text-color-primary);
      font-size: 13px;
      text-overflow: ellipsis;
      white-space: nowrap;
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

  .node-param-head-tags {
    display: inline-flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
    min-width: 0;
  }

  .node-param-actions {
    display: inline-flex;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
  }

  .node-param-code {
    overflow: hidden;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .node-param-meta {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    min-height: 22px;
    color: var(--el-text-color-secondary);
    font-size: 12px;

    span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .node-param-binding-summary {
    display: grid;
    grid-template-columns: minmax(160px, 0.8fr) minmax(0, 1.2fr);
    gap: 10px;
    min-width: 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;

    span {
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  @media (max-width: 960px) {
    .node-param-item {
      grid-template-columns: 1fr;
      align-items: stretch;
    }

    .node-param-head-tags,
    .node-param-actions {
      justify-content: flex-start;
    }
  }

  .node-param-editor-form {
    display: grid;
    gap: 0;
  }

  .node-param-editor-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(260px, 1fr));
    column-gap: 18px;

    :deep(.el-form-item) {
      margin-bottom: 14px;
    }
  }

  .node-param-editor-wide {
    grid-column: 1 / -1;
  }

  .node-param-mode-row {
    display: grid;
    grid-template-columns: 96px minmax(0, 1fr);
    align-items: center;
    margin-bottom: 16px;

    span {
      padding-right: 12px;
      color: var(--el-text-color-regular);
      font-size: 14px;
      line-height: 32px;
      text-align: right;
    }

    :deep(.el-segmented) {
      width: 100%;
    }
  }

  .node-param-editor-dialog {
    :deep(.el-segmented) {
      width: 100%;
    }

    :deep(.el-segmented__item-label) {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      width: 100%;
      min-height: 32px;
      line-height: 1;
      text-align: center;
    }

    :deep(.el-tag) {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      line-height: 1;
    }
  }

  .pipeline-param-contract {
    display: grid;
    gap: 14px;
    padding: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .pipeline-param-contract-head,
  .pipeline-param-platform-head,
  .pipeline-param-options-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;
  }

  .pipeline-param-contract-head {
    padding-bottom: 10px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .pipeline-param-contract-head strong,
  .pipeline-param-contract-head span,
  .pipeline-param-platform-head strong,
  .pipeline-param-platform-head span {
    display: block;
    min-width: 0;
  }

  .pipeline-param-contract-head strong,
  .pipeline-param-platform-head strong {
    font-size: 14px;
    color: var(--el-text-color-primary);
  }

  .pipeline-param-contract-head span,
  .pipeline-param-platform-head span {
    margin-top: 4px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .pipeline-param-contract-tags {
    display: inline-flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    justify-content: flex-end;
  }

  .pipeline-param-contract-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 18px;

    :deep(.el-form-item) {
      margin-bottom: 14px;
    }
  }

  .pipeline-param-switch-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(145px, 1fr));
    gap: 14px;
    align-items: stretch;
  }

  .pipeline-param-platform {
    display: grid;
    gap: 12px;
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px dashed var(--el-border-color-light);
    border-radius: 8px;
  }

  .pipeline-param-platform-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 18px;

    :deep(.el-form-item) {
      margin-bottom: 14px;
    }

    :deep(.wide-label-field) {
      grid-column: 1 / -1;
    }

    :deep(.wide-label-field .el-form-item__label) {
      width: 168px;
      flex: 0 0 168px;
    }
  }

  .param-switch-field {
    display: flex;
    flex-direction: column;
    justify-content: center;
    min-height: 66px;
    padding: 10px 12px;
    margin-bottom: 0;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .param-switch-field :deep(.el-form-item__label) {
    display: flex;
    align-items: center;
    justify-content: center;
    margin-bottom: 8px;
    padding: 0;
    line-height: 20px;
    text-align: center;
  }

  .param-switch-field :deep(.el-form-item__content) {
    align-items: center;
    justify-content: center;
    min-height: 32px;
  }

  .pipeline-param-options {
    display: grid;
    gap: 8px;
    padding: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px dashed var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .pipeline-param-options-head {
    color: var(--el-text-color-primary);
    font-size: 13px;
  }

  .pipeline-param-option-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 56px;
    gap: 8px;
    align-items: center;
  }

  .node-param-description {
    margin-bottom: 0;
  }

  @media (max-width: 860px) {
    .pipeline-meta-row,
    .node-param-editor-grid,
    .pipeline-param-contract-grid,
    .pipeline-param-platform-grid,
    .pipeline-param-switch-grid {
      grid-template-columns: 1fr;
    }

    .pipeline-param-contract-head,
    .pipeline-param-platform-head {
      align-items: flex-start;
      flex-direction: column;
    }

    .pipeline-param-contract-tags {
      justify-content: flex-start;
    }

    .node-param-mode-row {
      grid-template-columns: 1fr;

      span {
        text-align: left;
      }
    }
  }

  .workspace-map-item {
    display: grid;
    gap: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-lighter);
    padding: 8px;

    &.required {
      border-color: var(--el-color-primary-light-5);
      background: linear-gradient(
        90deg,
        var(--el-color-primary-light-9),
        var(--el-fill-color-lighter)
      );
    }
  }

  .workspace-policy-editor {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
    gap: 8px;
    align-items: end;
    padding-top: 4px;
  }

  .workspace-runtime-hint {
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.4;
  }

  .workspace-binding-summary {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .workspace-extra-row,
  .matrix-include-row {
    grid-template-columns: repeat(auto-fit, minmax(120px, 1fr));
  }

  .policy-row {
    grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  }

  .policy-row :deep(.el-input-number) {
    width: 100%;
  }

  .matrix-row {
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) auto;
  }

  .section-subtitle {
    color: var(--el-text-color-secondary);
    font-size: 12px;
  }

  .config-note {
    margin: 0;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }

  .switch-row {
    display: flex;
    flex-wrap: wrap;
    gap: 10px 16px;
  }

  :deep(.tekton-dag-dialog) {
    border-radius: 8px;
  }

  :deep(.tekton-dag-dialog .el-dialog__header) {
    margin: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
    padding: 18px 24px 14px;
  }

  :deep(.tekton-dag-dialog .el-dialog__body) {
    padding: 16px 24px 18px;
  }

  :deep(.tekton-dag-dialog .el-dialog__footer) {
    border-top: 1px solid var(--el-border-color-lighter);
    padding: 12px 24px 16px;
  }

  :deep(.tekton-task-library-dialog .el-dialog__header) {
    padding: 18px 22px;
  }

  :deep(.tekton-task-library-dialog .el-dialog__title) {
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--el-text-color-primary);
  }

  :deep(.tekton-task-library-dialog .el-dialog__headerbtn) {
    top: 12px;
    right: 14px;
  }

  :deep(.tekton-task-library-dialog .el-dialog__body) {
    padding: 0;
    overflow: hidden;
  }

  .dialog-library {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 16px;
    height: min(560px, calc(86vh - 168px));
    padding: 24px 40px 28px;
    overflow: hidden;
    border: 0;
  }

  .step-select-filter {
    display: grid;
    grid-template-columns: minmax(220px, 260px) minmax(320px, 1fr) 108px;
    gap: 12px;
    align-items: center;
  }

  .step-select-filter :deep(.el-button) {
    justify-self: end;
    width: 108px;
  }

  .step-choice-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    align-content: start;
    min-height: 0;
    padding: 2px 4px 8px;
    overflow: auto;
    scrollbar-gutter: stable;
  }

  .step-choice-grid.is-empty {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: 360px;
  }

  .step-choice-empty {
    width: 100%;
  }

  .step-choice-card {
    display: grid;
    grid-template-columns: 40px minmax(0, 1fr);
    gap: 12px;
    align-items: center;
    min-height: 86px;
    padding: 12px 14px;
    text-align: left;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .step-choice-card:hover {
    border-color: var(--el-color-primary);
    box-shadow: 0 8px 24px rgb(37 99 235 / 10%);
  }

  .step-choice-card:disabled {
    cursor: not-allowed;
    box-shadow: none;
    opacity: 0.55;
  }

  .step-choice-card:disabled:hover {
    border-color: var(--el-border-color-light);
    box-shadow: none;
  }

  .library-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 40px;
    height: 40px;
    font-size: 22px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .library-info {
    min-width: 0;
  }

  .library-info strong,
  .library-info em,
  .library-info small {
    display: block;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .library-info em {
    margin-top: 5px;
    font-style: normal;
    color: var(--el-text-color-secondary);
  }

  .library-info small {
    margin-top: 4px;
    color: var(--el-text-color-placeholder);
    font-size: 12px;
  }

  .dialog-inspector {
    height: min(64vh, 620px);
    border: 0;
    padding: 0;
  }

  .dialog-inspector .panel-head {
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
    background: var(--el-fill-color-light);
    padding: 10px 12px;
  }

  .dialog-inspector .inspector-form {
    padding: 0 4px 0 0;
  }

  .validation-dialog-body {
    min-height: 160px;
    max-height: min(56vh, 480px);
    overflow: auto;
  }

  .issue-list {
    display: grid;
    gap: 8px;
  }

  .issue-item {
    display: flex;
    gap: 8px;
    border-left: 3px solid var(--el-color-warning);
    background: var(--el-fill-color-light);
    padding: 8px;
  }

  .issue-item.error {
    border-left-color: var(--el-color-danger);
  }
</style>
