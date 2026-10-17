<template>
  <div class="devops-workflow-designer" @click="closeContextMenu">
    <section class="workflow-canvas">
      <div class="canvas-head">
        <div v-if="$slots['left-actions']" class="canvas-left-actions">
          <slot name="left-actions" />
        </div>
        <div class="canvas-title">
          <strong>配置流水线</strong>
          <span>{{ nodes.length }} 个节点{{ locked ? '，画布已锁定' : '' }}</span>
        </div>
        <div class="canvas-actions">
          <ElButton @click.stop="zoomCanvas('out')">缩小</ElButton>
          <ElButton @click.stop="zoomCanvas('in')">放大</ElButton>
          <ElButton @click.stop="fitCurrentView">适应视图</ElButton>
          <ElButton
            v-if="enablePostEntry && !postEntryVisible && !readonly"
            @click.stop="showPostEntry"
          >
            添加post阶段
          </ElButton>
          <slot name="actions" :locked="locked" :fit-view="fitCurrentView" />
        </div>
      </div>

      <div ref="canvasBodyRef" class="canvas-body">
        <VueFlow
          ref="vueFlowRef"
          v-model:nodes="flowNodes"
          :edges="flowEdges"
          :nodes-draggable="!editingDisabled"
          :nodes-connectable="!editingDisabled"
          :edges-updatable="!editingDisabled"
          :elements-selectable="!editingDisabled"
          :connection-mode="ConnectionMode.Loose"
          :connection-line-options="connectionLineOptions"
          :is-valid-connection="isValidWorkflowConnection"
          :min-zoom="minCanvasZoom"
          :max-zoom="1.35"
          :edge-updater-radius="14"
          :pan-on-drag="true"
          :pan-on-scroll="true"
          :pan-on-scroll-mode="PanOnScrollMode.Free"
          :zoom-on-scroll="true"
          :zoom-on-pinch="true"
          :zoom-on-double-click="true"
          :fit-view-on-init="false"
          class="flow-view"
          @nodes-initialized="handleNodesInitialized"
          @node-click="handleNodeClick"
          @node-drag-start="handleNodeDragStart"
          @node-drag="handleNodeDrag"
          @node-drag-stop="handleNodeDragStop"
          @pane-click="handlePaneClick"
          @connect="handleConnect"
          @edge-update="handleEdgeUpdate"
          @viewport-change="handleViewportChange"
          @viewport-change-end="handleViewportChange"
        >
          <Controls
            :show-interactive="!readonly"
            :fit-view-params="fitViewParams"
            @fit-view="fitCurrentView"
            @interaction-change="handleInteractionChange"
          />
          <MiniMap pannable zoomable />

          <template #node-parallelGroup="{ data }">
            <div
              class="parallel-group-node"
              :style="{
                width: `${data.width || parallelGroupMinWidth}px`,
                height: `${data.height || parallelGroupMinHeight}px`
              }"
            >
              <div class="parallel-group-title">
                <strong>{{ data.groupTitle || '并行执行' }}</strong>
                <span>{{ data.branchCount || 0 }} 个分支</span>
              </div>
              <Handle
                id="in"
                type="target"
                :position="Position.Left"
                :connectable="false"
                class="parallel-group-handle parallel-group-handle-target parallel-group-handle-in"
              />
              <Handle
                id="branch"
                type="source"
                :position="Position.Left"
                :connectable="false"
                class="parallel-group-handle parallel-group-handle-source parallel-group-handle-branch"
              />
              <Handle
                id="exit"
                type="source"
                :position="Position.Right"
                :connectable="false"
                class="parallel-group-handle parallel-group-handle-source parallel-group-handle-exit"
              />
            </div>
          </template>

          <template #node-start="{ data }">
            <div
              class="start-node"
              :class="{ post: data.entryType === 'post' }"
              @contextmenu.prevent.stop="openContextMenu($event, data.nodeId || startNodeId)"
            >
              <span class="start-mark">{{ data.entryType === 'post' ? 'Post' : 'Start' }}</span>
              <span class="start-text">
                <strong>{{ data.entryType === 'post' ? 'Post 入口' : '流水线入口' }}</strong>
                <em>{{
                  data.entryType === 'post' ? 'Jenkins post' : `${engineTypeName} Pipeline`
                }}</em>
              </span>
              <button
                v-if="!editingDisabled"
                type="button"
                class="node-tail-add"
                :title="data.entryType === 'post' ? '添加 post 步骤' : '添加起始步骤'"
                @click.stop="openAddDialog(data.nodeId || startNodeId, 'next')"
              >
                +
              </button>
              <Handle
                type="source"
                :position="Position.Right"
                :connectable="!editingDisabled"
                class="workflow-handle workflow-handle-source"
              />
              <Handle
                v-if="data.entryType !== 'post'"
                id="post-out"
                type="source"
                :position="Position.Bottom"
                :connectable="true"
                :connectable-start="false"
                :connectable-end="false"
                class="workflow-handle workflow-handle-post-source"
              />
              <Handle
                v-if="data.entryType === 'post'"
                id="post-in"
                type="target"
                :position="Position.Top"
                :connectable="true"
                :connectable-start="false"
                :connectable-end="false"
                class="workflow-handle workflow-handle-post-target"
              />
            </div>
          </template>

          <template #node-pipeline="{ data }">
            <div
              class="pipeline-node"
              :class="{
                active: selectedNodeId === data.nodeId,
                disabled: !data.enabled,
                approval: data.stepType === 'manual_approval',
                parallel: data.branchType === 'parallel',
                'post-step': data.rootType === 'post',
                invalid: data.missingRequired
              }"
              :style="{ width: `${data.width || nodeDefaultWidth}px` }"
              @contextmenu.prevent.stop="openContextMenu($event, data.nodeId)"
            >
              <button
                v-if="!editingDisabled"
                type="button"
                class="node-head-add"
                title="在此步骤前插入"
                @click.stop="openAddDialog(data.nodeId, 'before')"
              >
                +
              </button>
              <span class="node-icon" :style="{ color: data.iconColor || '#64748b' }">
                <ArtSvgIcon :icon="data.icon || defaultIcon" />
              </span>
              <span class="node-main">
                <span class="node-title" :title="data.nodeName">{{ data.nodeName }}</span>
                <span
                  v-if="data.missingRequired"
                  class="node-warning"
                  :title="data.missingRequiredText"
                >
                  {{ data.missingRequiredText }}
                </span>
                <span v-else-if="data.stepCode" class="node-code" :title="data.stepCode">
                  {{ data.stepCode }}
                </span>
              </span>
              <span class="node-badges">
                <ElTag
                  v-if="data.rootType === 'post'"
                  class="node-type-tag node-post-tag"
                  size="small"
                  type="warning"
                >
                  Post
                </ElTag>
                <ElTag
                  v-if="data.branchType === 'parallel'"
                  class="node-type-tag node-branch-tag"
                  size="small"
                  type="info"
                >
                  并行
                </ElTag>
                <ElTag
                  v-if="data.childCount && data.childCount > 1"
                  class="node-type-tag node-fanout-tag"
                  size="small"
                  type="info"
                >
                  分支{{ data.childCount }}
                </ElTag>
                <ElTag
                  class="node-type-tag"
                  size="small"
                  :type="data.stepType === 'manual_approval' ? 'warning' : 'success'"
                >
                  {{ stepTypeName(data.stepType) }}
                </ElTag>
              </span>
              <button
                v-if="!editingDisabled"
                type="button"
                class="node-tail-add"
                title="添加下一步"
                @click.stop="openAddDialog(data.nodeId, 'next')"
              >
                +
              </button>
              <Handle
                type="target"
                :position="Position.Left"
                :connectable="!editingDisabled"
                class="workflow-handle workflow-handle-target"
              />
              <Handle
                type="source"
                :position="Position.Right"
                :connectable="!editingDisabled"
                class="workflow-handle workflow-handle-source"
              />
              <span
                v-if="!editingDisabled"
                class="node-resize-grip"
                title="拖拽调整卡片宽度"
                @pointerdown.stop.prevent="startNodeResize($event, data.nodeId)"
              />
            </div>
          </template>
        </VueFlow>

        <svg
          v-if="postEntryLink"
          class="post-entry-link-layer"
          :width="postEntryLink.canvasWidth"
          :height="postEntryLink.canvasHeight"
          :viewBox="`0 0 ${postEntryLink.canvasWidth} ${postEntryLink.canvasHeight}`"
          aria-hidden="true"
        >
          <defs>
            <marker
              id="post-entry-link-arrow"
              markerWidth="7"
              markerHeight="7"
              refX="6"
              refY="3.5"
              orient="auto"
              markerUnits="strokeWidth"
            >
              <path d="M 0 0 L 7 3.5 L 0 7 z" fill="#f97316" />
            </marker>
          </defs>
          <path
            class="post-entry-link-path"
            :d="postEntryLink.path"
            marker-end="url(#post-entry-link-arrow)"
          />
        </svg>

        <div v-if="nodes.length === 0" class="canvas-empty-tip">
          <strong>{{ readonly ? '暂无流水线步骤' : '从 Start 节点添加第一个步骤' }}</strong>
          <span>
            {{ readonly ? '查看模式仅展示模板编排。' : '右键步骤可显式添加并行分支。' }}
          </span>
        </div>

        <div
          v-if="contextMenu.visible && !editingDisabled"
          class="workflow-context-menu"
          :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
          @click.stop
        >
          <button type="button" @click="openAddDialog(contextMenu.nodeId, 'next')">
            {{ contextMenuAddText }}
          </button>
          <button
            v-if="!isRootNodeId(contextMenu.nodeId)"
            type="button"
            @click="openAddDialog(contextMenu.nodeId, 'before')"
          >
            在前面插入
          </button>
          <button
            v-if="!isRootNodeId(contextMenu.nodeId)"
            type="button"
            @click="openAddDialog(contextMenu.nodeId, 'parallelSibling')"
          >
            添加并行步骤
          </button>
          <button
            v-if="!isRootNodeId(contextMenu.nodeId)"
            type="button"
            @click="openEditDialog(contextMenu.nodeId)"
          >
            配置步骤
          </button>
          <button
            v-if="!isRootNodeId(contextMenu.nodeId)"
            type="button"
            class="danger"
            @click="removeNode(contextMenu.nodeId)"
          >
            删除步骤
          </button>
        </div>
      </div>
    </section>

    <ElDialog
      v-model="stepDialogVisible"
      :title="stepDialogMode === 'edit' ? (readonly ? '查看步骤' : '配置步骤') : '添加步骤'"
      width="900px"
      align-center
      class="workflow-polish-dialog workflow-step-dialog"
      :close-on-click-modal="false"
      @close="resetStepDialog"
    >
      <div v-if="stepDialogStage === 'select'" class="step-select-panel">
        <div class="step-select-filter">
          <ElSelect
            :model-value="dialogCategoryId"
            class="step-category-select"
            popper-class="devops-platform-param-dropdown"
            :clearable="false"
            :disabled="isPostDialog"
            :placeholder="isPostDialog ? 'Post 步骤分组' : '步骤分类'"
            @update:model-value="handleDialogCategoryChange"
          >
            <ElOption
              v-for="item in selectableCategoryOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
          <ElInput v-model="stepKeyword" clearable placeholder="搜索步骤名称或编码" />
          <ElButton :loading="libraryLoading" @click="refreshLibraryForDialog()">刷新</ElButton>
        </div>

        <div
          class="step-choice-grid"
          :class="{ 'is-empty': !libraryLoading && filteredLibraryItems.length === 0 }"
        >
          <template v-if="libraryLoading">
            <div v-for="index in 9" :key="index" class="step-choice-card is-skeleton">
              <span class="library-icon" />
              <span class="library-info">
                <strong />
                <em />
              </span>
            </div>
          </template>
          <ElEmpty
            v-else-if="filteredLibraryItems.length === 0"
            class="step-choice-empty"
            :description="emptyText"
          />
          <template v-else>
            <button
              v-for="item in filteredLibraryItems"
              :key="item.id"
              type="button"
              class="step-choice-card"
              :class="{
                approval: item.type === 'manual_approval',
                post: isPostDialog,
                'is-disabled': isPostChoiceDisabled(item)
              }"
              :disabled="isPostChoiceDisabled(item)"
              :title="postChoiceDisabledReason(item)"
              @click="chooseStep(item)"
            >
              <span class="library-icon" :style="{ color: item.iconColor || '#64748b' }">
                <ArtSvgIcon :icon="item.icon || defaultIcon" />
              </span>
              <span class="library-info">
                <strong :title="item.name">{{ item.name }}</strong>
                <em>{{ item.paramsCount || 0 }} 个变量</em>
              </span>
            </button>
          </template>
        </div>
      </div>

      <ElForm v-else class="step-config-form" label-width="92px" :disabled="readonly || locked">
        <div class="chosen-step-head">
          <span class="library-icon" :style="{ color: stepForm.iconColor || '#64748b' }">
            <ArtSvgIcon :icon="stepForm.icon || defaultIcon" />
          </span>
          <div>
            <strong>{{ stepForm.stepName }}</strong>
            <span>{{ stepForm.stepCode }}</span>
          </div>
          <ElTag
            class="node-type-tag"
            :type="stepForm.stepType === 'manual_approval' ? 'warning' : 'success'"
          >
            {{ stepTypeName(stepForm.stepType) }}
          </ElTag>
        </div>
        <ElFormItem label="节点名称">
          <ElInput v-model="stepForm.nodeName" placeholder="请输入节点名称" />
        </ElFormItem>
        <ElFormItem label="容器节点">
          <ElSelect
            v-model="stepForm.containerName"
            clearable
            filterable
            :allow-create="props.containerEditable"
            default-first-option
            :disabled="readonly || locked || !props.containerEditable"
            :placeholder="props.containerEditable ? '默认不指定容器' : '仅动态节点支持容器节点'"
            style="width: 100%"
          >
            <ElOption
              v-for="item in props.containerOptions"
              :key="item"
              :label="item"
              :value="item"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="节点状态">
          <ElSwitch v-model="stepForm.enabled" />
        </ElFormItem>
        <div v-if="enableStepParamConfig && currentStepParams.length > 0" class="step-param-panel">
          <div class="step-param-head">
            <strong>步骤参数</strong>
            <span>{{ currentStepParams.length }} 个参数，{{ requiredStepParamCount }} 个必填</span>
          </div>
          <div class="step-param-grid">
            <div
              v-for="param in currentStepParams"
              :key="param.code"
              class="step-param-item"
              :class="{
                'is-missing': isStepParamMissing(param),
                'is-full-row': isFullRowParam(param)
              }"
            >
              <div class="step-param-item-head">
                <span class="step-param-title">
                  <i v-if="param.required">*</i>
                  <strong :title="param.name || param.code">{{ param.name || param.code }}</strong>
                </span>
                <ElTag size="small" :type="stepParamModeTagType(param)">
                  {{ stepParamModeLabel(param) }}
                </ElTag>
                <ElTag v-if="param.readonly" size="small" type="info">只读</ElTag>
                <ElTag
                  v-if="param.paramType === channelCredentialParamType"
                  size="small"
                  type="info"
                >
                  自动派生
                </ElTag>
              </div>
              <div class="step-param-code" :title="param.code">{{ param.code }}</div>
              <div class="step-param-control">
                <div
                  v-if="param.paramType === channelCredentialParamType"
                  class="derived-param-preview"
                >
                  <span>来源：{{ param.config?.credentialSourceParamCode || '-' }}</span>
                  <span>{{ channelCredentialOutputLabel(param) }}</span>
                </div>
                <template v-else-if="param.paramType === 'objectList'">
                  <div class="object-list-editor">
                    <div class="object-list-head">
                      <span>{{ (param.config?.compoundFields || []).length }} 个字段</span>
                      <ElButton
                        v-if="!isStepParamReadonly(param)"
                        size="small"
                        type="primary"
                        plain
                        @click="addObjectListRow(param)"
                      >
                        新增一行
                      </ElButton>
                    </div>
                    <ElEmpty
                      v-if="!(param.config?.compoundFields || []).length"
                      description="未配置对象列表字段"
                    />
                    <div v-else class="object-list-table">
                      <ElTable
                        :data="objectListRows(param)"
                        size="small"
                        border
                        empty-text="暂无数据"
                      >
                        <ElTableColumn
                          v-for="field in param.config?.compoundFields || []"
                          :key="field.code"
                          :label="field.name"
                          :min-width="160"
                        >
                          <template #default="{ row, $index }">
                            <div
                              v-if="isDynamicObjectListField(field)"
                              class="dynamic-select-control is-object-list"
                            >
                              <ElSelect
                                :model-value="objectListFieldValue(row, field)"
                                v-bind="objectListFieldProps(param, row, $index, field)"
                                popper-class="devops-platform-param-dropdown"
                                :teleported="false"
                                fit-input-width
                                @visible-change="
                                  (visible: boolean) =>
                                    handleObjectListFieldVisibleChange(
                                      param,
                                      row,
                                      $index,
                                      field,
                                      visible
                                    )
                                "
                                @update:model-value="
                                  (value: any) =>
                                    updateObjectListFieldValue(param, $index, field.code, value)
                                "
                              >
                                <template #loading>
                                  <div class="devops-select-loading">
                                    <span class="devops-select-loading__icon"></span>
                                    <span>加载中...</span>
                                  </div>
                                </template>
                                <div
                                  v-if="
                                    objectListDynamicLoadError[
                                      objectListOptionKey(param, field, $index)
                                    ]
                                  "
                                  class="dynamic-option-tip is-error"
                                >
                                  {{
                                    objectListDynamicLoadError[
                                      objectListOptionKey(param, field, $index)
                                    ]
                                  }}
                                </div>
                                <ElOption
                                  v-for="option in objectListFieldOptions(param, field, $index)"
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
                                size="small"
                                circle
                                :icon="RefreshRight"
                                :loading="
                                  objectListDynamicLoading[
                                    objectListOptionKey(param, field, $index)
                                  ] === true
                                "
                                :disabled="isObjectListFieldReadonly(param, field)"
                                title="刷新选项"
                                aria-label="刷新选项"
                                @click.stop="
                                  refreshObjectListFieldOptions(param, row, $index, field)
                                "
                              />
                            </div>
                            <component
                              v-else
                              :is="objectListFieldComponent(field)"
                              :model-value="objectListFieldValue(row, field)"
                              v-bind="objectListFieldProps(param, row, $index, field)"
                              @visible-change="
                                (visible: boolean) =>
                                  handleObjectListFieldVisibleChange(
                                    param,
                                    row,
                                    $index,
                                    field,
                                    visible
                                  )
                              "
                              @update:model-value="
                                (value: any) =>
                                  updateObjectListFieldValue(param, $index, field.code, value)
                              "
                            >
                              <template v-if="objectListFieldIsSelect(field)" #default>
                                <ElOption
                                  v-for="option in objectListFieldOptions(param, field, $index)"
                                  :key="option.value"
                                  :label="option.label"
                                  :value="option.value"
                                />
                              </template>
                            </component>
                          </template>
                        </ElTableColumn>
                        <ElTableColumn
                          v-if="!isStepParamReadonly(param)"
                          label="操作"
                          width="86"
                          fixed="right"
                          align="center"
                        >
                          <template #default="{ $index }">
                            <ElButton
                              text
                              type="danger"
                              @click="removeObjectListRow(param, $index)"
                            >
                              删除
                            </ElButton>
                          </template>
                        </ElTableColumn>
                      </ElTable>
                    </div>
                  </div>
                </template>
                <DevopsKubeNovaDeployConfigEditor
                  v-else-if="isKubeNovaDeployConfigParam(param)"
                  :model-value="stringStepParamValue(param.code)"
                  :project-id="dynamicProjectId"
                  :system-id="dynamicSystemId"
                  :environment-id="dynamicEnvironmentId"
                  :channel-group-code="param.config?.channelGroupCode || ''"
                  :channel-type="param.config?.channelTypeFilter || ''"
                  :mapping-field="param.config?.mappingField || ''"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                  @change="() => clearStepDependentParams(param.code)"
                />
                <DevopsKubernetesDeployConfigEditor
                  v-else-if="isKubernetesDeployConfigParam(param)"
                  :model-value="stringStepParamValue(param.code)"
                  :project-id="dynamicProjectId"
                  :system-id="dynamicSystemId"
                  :environment-id="dynamicEnvironmentId"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                  @change="() => clearStepDependentParams(param.code)"
                />
                <ElInput
                  v-else-if="param.paramType === 'text'"
                  :model-value="stringStepParamValue(param.code)"
                  type="textarea"
                  :rows="isFullRowParam(param) ? 5 : 4"
                  :placeholder="param.defaultValue || '请输入参数值'"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                />
                <ElSwitch
                  v-else-if="param.paramType === 'booleanParam'"
                  :model-value="booleanStepParamValue(param.code)"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="
                    (value: string | number | boolean) =>
                      setStepParamValue(param.code, value === true ? 'true' : 'false')
                  "
                />
                <ElInputNumber
                  v-else-if="param.paramType === 'number'"
                  :model-value="numberStepParamValue(param.code)"
                  controls-position="right"
                  style="width: 100%"
                  :placeholder="param.defaultValue || '请输入数字'"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="
                    (value: number | undefined) => setNumberStepParamValue(param.code, value)
                  "
                />
                <DevopsPlatformAddressPicker
                  v-else-if="isCompositeStepParam(param)"
                  :model-value="stringStepParamValue(param.code)"
                  :param-type="platformStepAddressParamType(param)"
                  :project-id="dynamicProjectId"
                  :system-id="dynamicSystemId"
                  :environment-id="dynamicEnvironmentId"
                  :channel-group-code="param.config?.channelGroupCode || ''"
                  :channel-type="param.config?.channelTypeFilter || ''"
                  :mapping-field="param.config?.mappingField || ''"
                  :disabled="isStepParamReadonly(param)"
                  :placeholder="param.defaultValue || ''"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                  @change="() => clearStepDependentParams(param.code)"
                />
                <div v-else-if="isDynamicStepParam(param)" class="dynamic-select-control">
                  <ElSelect
                    :model-value="stringStepParamValue(param.code)"
                    class="platform-param-select"
                    popper-class="devops-platform-param-dropdown"
                    :teleported="false"
                    fit-input-width
                    filterable
                    clearable
                    allow-create
                    default-first-option
                    remote
                    remote-show-suffix
                    :remote-method="
                      (keyword: string) => queryStepDynamicParamOptions(param, keyword)
                    "
                    :loading="stepDynamicLoading[param.code] === true"
                    loading-text="加载中..."
                    :no-data-text="dynamicEmptyText(param.paramType)"
                    :disabled="isStepParamReadonly(param)"
                    :placeholder="stepDynamicPlaceholder(param)"
                    @visible-change="
                      (visible: boolean) => handleStepDynamicParamVisible(param, visible)
                    "
                    @update:model-value="(value: string) => updateStepParamValue(param, value)"
                  >
                    <template #loading>
                      <div class="devops-select-loading">
                        <span class="devops-select-loading__icon"></span>
                        <span>加载中...</span>
                      </div>
                    </template>
                    <div
                      v-if="stepDynamicLoadError[param.code]"
                      class="dynamic-option-tip is-error"
                    >
                      {{ stepDynamicLoadError[param.code] }}
                    </div>
                    <div
                      v-else-if="
                        stepDynamicLoaded[param.code] &&
                        !stepDynamicLoading[param.code] &&
                        !(stepDynamicOptions[param.code] || []).length
                      "
                      class="dynamic-option-tip"
                    >
                      {{ dynamicEmptyText(param.paramType) }}
                    </div>
                    <ElOption
                      v-for="option in stepDynamicOptions[param.code] || []"
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
                    :loading="stepDynamicLoading[param.code] === true"
                    :disabled="isStepParamReadonly(param)"
                    title="刷新选项"
                    aria-label="刷新选项"
                    @click.stop="refreshStepDynamicParamOptions(param)"
                  >
                    刷新
                  </ElButton>
                </div>
                <ElSelect
                  v-else-if="param.paramType === 'select'"
                  :model-value="arrayStepParamValue(param.code)"
                  class="platform-param-select"
                  popper-class="devops-platform-param-dropdown"
                  :teleported="false"
                  fit-input-width
                  multiple
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  placeholder="请选择"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string[]) => setStepParamValue(param.code, value)"
                >
                  <ElOption
                    v-for="option in param.selectList || []"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </ElSelect>
                <ElSelect
                  v-else-if="param.paramType === 'choice' || param.paramType === 'extendedString'"
                  :model-value="stringStepParamValue(param.code)"
                  class="platform-param-select"
                  popper-class="devops-platform-param-dropdown"
                  :teleported="false"
                  fit-input-width
                  filterable
                  clearable
                  allow-create
                  default-first-option
                  placeholder="请选择或输入"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                >
                  <ElOption
                    v-for="option in param.selectList || []"
                    :key="option.value"
                    :label="option.label"
                    :value="option.value"
                  />
                </ElSelect>
                <ElInput
                  v-else
                  :model-value="stringStepParamValue(param.code)"
                  :type="param.paramType === 'password' ? 'password' : 'text'"
                  :show-password="param.paramType === 'password'"
                  :placeholder="param.defaultValue || '请输入参数值'"
                  :disabled="isStepParamReadonly(param)"
                  @update:model-value="(value: string) => setStepParamValue(param.code, value)"
                />
              </div>
              <div v-if="param.description" class="step-param-help">{{ param.description }}</div>
            </div>
          </div>
        </div>
        <ElFormItem v-else-if="enableStepParamConfig" label="参数配置">
          <ElInput
            v-model="stepForm.paramValues"
            type="textarea"
            :rows="8"
            placeholder="请输入 JSON，模板阶段可以留空"
          />
        </ElFormItem>
      </ElForm>

      <template #footer>
        <div class="workflow-dialog-footer">
          <ElButton
            v-if="stepDialogStage === 'config' && stepDialogMode === 'add'"
            @click="backToStepSelect"
          >
            上一步
          </ElButton>
          <ElButton @click="stepDialogVisible = false">取消</ElButton>
          <ElButton
            v-if="stepDialogStage === 'config' && !readonly && !locked"
            type="primary"
            @click="saveStepDialog"
          >
            保存步骤
          </ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, ref, watch } from 'vue'
  import { ElInput, ElInputNumber, ElOption, ElSelect, ElSwitch } from 'element-plus'
  import { RefreshRight } from '@element-plus/icons-vue'
  import {
    ConnectionMode,
    ConnectionLineType,
    Handle,
    MarkerType,
    PanOnScrollMode,
    Position,
    VueFlow,
    type Connection,
    type ConnectionLineOptions,
    type Edge,
    type EdgeUpdateEvent,
    type FitViewParams,
    type Node,
    type NodeDragEvent
  } from '@vue-flow/core'
  import { Controls } from '@vue-flow/controls'
  import { MiniMap } from '@vue-flow/minimap'
  import { ElMessage } from 'element-plus'
  import DevopsKubeNovaDeployConfigEditor from '@/components/devops-kube-nova-deploy-config-editor/index.vue'
  import DevopsKubernetesDeployConfigEditor from '@/components/devops-kubernetes-deploy-config-editor/index.vue'
  import DevopsPlatformAddressPicker from '@/components/devops-platform-address-picker/index.vue'
  import { isCompositePlatformParamType } from '@/components/devops-platform-address-picker/defs'
  import { listDevopsDynamicParamOptionsApi } from '@/api/devops/pipeline'
  import { listDevopsTektonPvcApi, listDevopsTektonSecretApi } from '@/api/devops/project'
  import type {
    WorkflowCategoryOption,
    WorkflowCompoundParamField,
    WorkflowLibraryItem,
    WorkflowNode,
    WorkflowParamDefinition,
    WorkflowParamOption
  } from './types'
  import { normalizeWorkflowBranches } from './workflow-utils'
  import '@vue-flow/core/dist/style.css'
  import '@vue-flow/core/dist/theme-default.css'
  import '@vue-flow/controls/dist/style.css'
  import '@vue-flow/minimap/dist/style.css'

  type AddRelation = 'next' | 'parallelChild' | 'parallelSibling' | 'before'
  const channelParamTypes = ['repositoryChannel', 'nexusChannel', 'harborChannel', 'jfrogChannel']
  const credentialParamType = 'voucherModel'
  const channelCredentialParamType = 'channelCredential'
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
    { projectParamType?: string; projectParamTypes?: string[]; channelParamType?: string }
  > = {
    gitProject: { channelParamType: 'repositoryChannel' },
    gitBranch: {
      channelParamType: 'repositoryChannel',
      projectParamType: 'gitProject',
      projectParamTypes: ['gitProject', 'gitRepositoryUrl']
    },
    gitTag: {
      channelParamType: 'repositoryChannel',
      projectParamType: 'gitProject',
      projectParamTypes: ['gitProject', 'gitRepositoryUrl']
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
      ]
    },
    nexusArtifactVersion: {
      channelParamType: 'nexusChannel',
      projectParamType: 'nexusRepository',
      projectParamTypes: [
        'nexusRepository',
        'nexusRepositoryUrl',
        'nexusArtifactUrl',
        'nexusArtifactVersionUrl'
      ]
    },
    nexusArtifact: {
      channelParamType: 'nexusChannel',
      projectParamType: 'nexusRepository',
      projectParamTypes: [
        'nexusRepository',
        'nexusRepositoryUrl',
        'nexusArtifactUrl',
        'nexusArtifactVersionUrl'
      ]
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
      ]
    },
    harborImageTag: {
      channelParamType: 'harborChannel',
      projectParamType: 'harborProject',
      projectParamTypes: [
        'harborProject',
        'harborProjectUrl',
        'harborImageUrl',
        'harborImageTagUrl'
      ]
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
      ]
    },
    registryImageTag: {
      channelParamType: 'channelGroupCode',
      projectParamType: 'registryRepository',
      projectParamTypes: [
        'registryRepository',
        'registryRepositoryUrl',
        'registryImageUrl',
        'registryImageTagUrl'
      ]
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
      ]
    },
    jfrogArtifactVersion: {
      channelParamType: 'jfrogChannel',
      projectParamType: 'jfrogRepository',
      projectParamTypes: [
        'jfrogRepository',
        'jfrogRepositoryUrl',
        'jfrogArtifactUrl',
        'jfrogArtifactVersionUrl'
      ]
    },
    jfrogArtifact: {
      channelParamType: 'jfrogChannel',
      projectParamType: 'jfrogRepository',
      projectParamTypes: [
        'jfrogRepository',
        'jfrogRepositoryUrl',
        'jfrogArtifactUrl',
        'jfrogArtifactVersionUrl'
      ]
    },
    kubeNovaProject: {},
    kubeNovaCluster: { projectParamType: 'kubeNovaProject' },
    kubeNovaWorkspace: { projectParamType: 'kubeNovaCluster' },
    kubeNovaApplication: { projectParamType: 'kubeNovaWorkspace' },
    kubeNovaVersion: { projectParamType: 'kubeNovaApplication' },
    kubeNovaContainer: { projectParamType: 'kubeNovaVersion' },
    kubernetesNamespace: {},
    kubernetesWorkloadType: {},
    kubernetesResource: { projectParamType: 'kubernetesNamespace' },
    kubernetesContainer: { projectParamType: 'kubernetesResource' },
    kubernetesImage: { projectParamType: 'kubernetesContainer' },
    configCenter: {},
    mavenConfig: {},
    sonarAddress: {},
    sonarProjectName: {},
    sonarProjectKey: {},
    hostGroupTargets: {},
    hostConfig: {},
    hostGroupConfig: {}
  }
  const tektonPlatformParamTypes = [
    'kubernetesSecretName',
    'kubernetesPvcName',
    ...channelParamTypes,
    ...dynamicParamTypes,
    credentialParamType,
    kubeNovaDeployConfigParamType,
    kubernetesDeployConfigParamType,
    'channelGroupCode'
  ]

  type FlowNodeData = {
    nodeId: string
    nodeName: string
    stepName: string
    stepCode: string
    stepType: string
    enabled: boolean
    entryType?: 'start' | 'post'
    branchType?: 'next' | 'parallel'
    rootType?: 'main' | 'post'
    childCount?: number
    branchCount?: number
    missingRequired?: boolean
    missingRequiredText?: string
    icon?: string
    iconColor?: string
    width?: number
    height?: number
    groupTitle?: string
  }

  type StepLibraryRefreshPayload = {
    categoryId?: string
  }

  type ParallelGroupDragState = {
    groupId: string
    parentId: string
    startX: number
    startY: number
    branchStartPositions: Record<string, { x: number; y: number }>
  }

  const props = withDefaults(
    defineProps<{
      modelValue: WorkflowNode[]
      libraryItems: WorkflowLibraryItem[]
      templateItems?: WorkflowLibraryItem[]
      categoryOptions?: WorkflowCategoryOption[]
      categoryId?: string
      postCategoryId?: string
      libraryLoading?: boolean
      engineType?: string
      emptyText?: string
      readonly?: boolean
      containerOptions?: string[]
      containerEditable?: boolean
      autoFitOnMount?: boolean
      dynamicProjectId?: string
      dynamicSystemId?: string
      dynamicEnvironmentId?: string
      dynamicBuildChannelBindingId?: string
      enableStepParamConfig?: boolean
      enablePostEntry?: boolean
    }>(),
    {
      templateItems: () => [],
      categoryOptions: () => [],
      categoryId: '',
      postCategoryId: '',
      libraryLoading: false,
      engineType: 'jenkins',
      emptyText: '暂无可用步骤',
      readonly: false,
      containerOptions: () => [],
      containerEditable: false,
      autoFitOnMount: true,
      dynamicProjectId: '',
      dynamicSystemId: '',
      dynamicEnvironmentId: '',
      dynamicBuildChannelBindingId: '',
      enableStepParamConfig: true,
      enablePostEntry: true
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: WorkflowNode[]]
    'update:categoryId': [value: string]
    'refresh-library': [payload?: StepLibraryRefreshPayload]
    save: []
  }>()

  const startNodeId = 'workflow-start'
  const postStartNodeId = 'workflow-post'
  const postConditionLabels: Record<string, string> = {
    always: 'always',
    success: 'success',
    failure: 'failure',
    changed: 'changed',
    aborted: 'aborted',
    unstable: 'unstable'
  }
  const postConditionPattern = /\b(always|success|failure|changed|aborted|unstable)\s*\{/g
  const defaultIcon = 'ri:flow-chart'
  const minCanvasZoom = 0.48
  const readableFitMinZoom = 0.58
  const nodeDefaultWidth = 246
  const nodeMinWidth = 230
  const nodeMaxWidth = 380
  const nodeGapX = 320
  const nodeGapY = 118
  const startY = 92
  const postStartY = 278
  const entryViewportX = 44
  const entryViewportY = 92
  const nodeEstimatedHeight = 76
  const parallelGroupPaddingX = 28
  const parallelGroupPaddingY = 54
  const parallelGroupMinWidth = 300
  const parallelGroupMinHeight = 190
  const fitViewParams: FitViewParams = {
    padding: { x: '8%', y: '12%' },
    minZoom: readableFitMinZoom,
    maxZoom: 1,
    duration: 260
  }
  const connectionLineOptions: ConnectionLineOptions = {
    type: ConnectionLineType.SmoothStep,
    style: { stroke: '#2563eb', strokeWidth: 2.4 },
    markerEnd: MarkerType.ArrowClosed
  }
  const vueFlowRef = ref<InstanceType<typeof VueFlow>>()
  const canvasBodyRef = ref<HTMLElement>()
  const selectedNodeId = ref('')
  const flowNodes = ref<Node<FlowNodeData>[]>([])
  const nodeWidthMap = ref<Record<string, number>>({})
  const viewportState = ref({ x: 0, y: 0, zoom: 1 })
  const parallelGroupDragState = ref<ParallelGroupDragState | null>(null)
  const locked = ref(false)
  const postEntryShown = ref(false)
  const stepDialogVisible = ref(false)
  const stepDialogMode = ref<'add' | 'edit'>('add')
  const stepDialogStage = ref<'select' | 'config'>('select')
  const stepKeyword = ref('')
  const dialogAnchorId = ref(startNodeId)
  const dialogRelation = ref<AddRelation>('next')
  const chosenStep = ref<WorkflowLibraryItem | null>(null)
  const contextMenu = ref({
    visible: false,
    x: 0,
    y: 0,
    nodeId: startNodeId
  })
  const stepForm = ref<WorkflowNode>(emptyStepForm())
  const stepParamMap = ref<Record<string, any>>({})
  const stepDynamicOptions = ref<Record<string, WorkflowParamOption[]>>({})
  const stepDynamicLoading = ref<Record<string, boolean>>({})
  const stepDynamicLoaded = ref<Record<string, boolean>>({})
  const stepDynamicLoadError = ref<Record<string, string>>({})
  const stepDynamicQueryKey = ref<Record<string, string>>({})
  const objectListDynamicOptions = ref<Record<string, WorkflowParamOption[]>>({})
  const objectListDynamicLoading = ref<Record<string, boolean>>({})
  const objectListDynamicLoadError = ref<Record<string, string>>({})
  const objectListDynamicQueryKey = ref<Record<string, string>>({})
  let syncingParallelGroupDrag = false

  const readonly = computed(() => props.readonly)
  const editingDisabled = computed(() => readonly.value || locked.value)
  const enableStepParamConfig = computed(() => props.enableStepParamConfig)
  const enablePostEntry = computed(() => props.enablePostEntry !== false)
  const nodes = computed(() => normalizeNodes(props.modelValue || []))
  const postEntryVisible = computed(
    () => enablePostEntry.value && (postEntryShown.value || hasPostNode(nodes.value))
  )
  const engineTypeName = computed(() => (props.engineType === 'tekton' ? 'Tekton' : 'Jenkins'))
  const isPostDialog = computed(() => isPostAnchor(dialogAnchorId.value))
  const postCategoryIds = computed(() => {
    const ids = new Set<string>()
    if (props.postCategoryId) ids.add(props.postCategoryId)
    props.categoryOptions.forEach((item) => {
      if (item.code === 'post' && item.value) ids.add(String(item.value))
    })
    return ids
  })
  const resolvedPostCategoryId = computed(() => {
    if (props.postCategoryId) return props.postCategoryId
    return Array.from(postCategoryIds.value)[0] || ''
  })
  const isRedundantStepCategory = (item: WorkflowCategoryOption) => {
    const label = String(item.label || '').replace(/（.*$/, '')
    return ['全部', '普通步骤', '人工审核'].includes(label)
  }
  const normalCategoryOptions = computed(() =>
    props.categoryOptions.filter((item) => item.code !== 'post' && !isRedundantStepCategory(item))
  )
  const selectableCategoryOptions = computed(() =>
    isPostDialog.value
      ? props.categoryOptions.filter((item) => item.code === 'post')
      : normalCategoryOptions.value
  )
  const defaultNormalCategoryId = computed(() => normalCategoryOptions.value[0]?.value || '')
  const dialogCategoryId = computed(() =>
    isPostDialog.value
      ? resolvedPostCategoryId.value
      : props.categoryId || defaultNormalCategoryId.value
  )
  const contextMenuAddText = computed(() => {
    if (contextMenu.value.nodeId === postStartNodeId) return '添加 post 步骤'
    if (contextMenu.value.nodeId === startNodeId) return '添加起始步骤'
    return '添加下一步'
  })
  const filteredLibraryItems = computed(() =>
    props.libraryItems.filter((item) => {
      if (
        !isPostDialog.value &&
        dialogCategoryId.value &&
        item.categoryId !== dialogCategoryId.value
      ) {
        return false
      }
      const isPostItem =
        item.categoryCode === 'post' ||
        (item.categoryId ? postCategoryIds.value.has(item.categoryId) : false)
      if (isPostDialog.value ? !isPostItem : isPostItem) return false
      const keyword = stepKeyword.value.trim().toLowerCase()
      if (!keyword) return true
      return `${item.name} ${item.code}`.toLowerCase().includes(keyword)
    })
  )
  function postConditionsFromContent(content?: string) {
    const normalized = String(content || '').trim()
    if (!normalized) return ['always']
    const conditions: string[] = []
    postConditionPattern.lastIndex = 0
    let match = postConditionPattern.exec(normalized)
    while (match) {
      if (!conditions.includes(match[1])) {
        conditions.push(match[1])
      }
      match = postConditionPattern.exec(normalized)
    }
    return conditions.length > 0 ? conditions : ['always']
  }

  function postConditionsForStep(stepId?: string, stepCode?: string) {
    const item = findLibraryItem(stepId, stepCode)
    return postConditionsFromContent(item?.stageContent)
  }

  function usedPostConditions(ignoreNodeId = '') {
    const used = new Set<string>()
    const byID = new Map(nodes.value.map((item) => [item.id || '', item]))
    nodes.value.forEach((node) => {
      if (!node.id || node.id === ignoreNodeId) return
      if (resolveRootNodeId(node.id, byID) !== postStartNodeId) return
      postConditionsForStep(node.stepId, node.stepCode).forEach((condition) => used.add(condition))
    })
    return used
  }

  function duplicatePostConditionForItem(item: WorkflowLibraryItem) {
    if (!isPostDialog.value) return ''
    const used = usedPostConditions()
    return (
      postConditionsFromContent(item.stageContent).find((condition) => used.has(condition)) || ''
    )
  }

  function isPostChoiceDisabled(item: WorkflowLibraryItem) {
    return Boolean(duplicatePostConditionForItem(item))
  }

  function postChoiceDisabledReason(item: WorkflowLibraryItem) {
    const condition = duplicatePostConditionForItem(item)
    if (!condition) return item.name
    return `post 条件 ${postConditionLabels[condition] || condition} 已存在`
  }

  const currentStepTemplate = computed(() =>
    findLibraryItem(stepForm.value.stepId, stepForm.value.stepCode)
  )
  const currentStepParams = computed<WorkflowParamDefinition[]>(() =>
    enableStepParamConfig.value ? currentStepTemplate.value?.params || [] : []
  )
  const requiredStepParamCount = computed(
    () => currentStepParams.value.filter((item) => item.required).length
  )
  const postEntryLink = computed(() => {
    if (!postEntryVisible.value) return null
    const body = canvasBodyRef.value
    if (!body) return null
    const startNode = flowNodes.value.find((item) => item.id === startNodeId)
    const postNode = flowNodes.value.find((item) => item.id === postStartNodeId)
    if (!startNode || !postNode) return null

    const viewport = viewportState.value
    const toCanvas = (x: number, y: number) => ({
      x: Math.round(viewport.x + x * viewport.zoom),
      y: Math.round(viewport.y + y * viewport.zoom)
    })
    const startWidth = flowNodeWidth(startNode)
    const postWidth = flowNodeWidth(postNode)
    const source = toCanvas(startNode.position.x + startWidth / 2, startNode.position.y + 76)
    const target = toCanvas(postNode.position.x + postWidth / 2, postNode.position.y)
    return {
      canvasWidth: Math.max(1, body.clientWidth),
      canvasHeight: Math.max(1, body.clientHeight),
      path: `M ${source.x} ${source.y} L ${target.x} ${target.y}`
    }
  })
  const flowEdges = computed<Edge[]>(() => {
    const byID = new Map(nodes.value.map((item) => [item.id || '', item]))
    const parallelGroups = buildParallelGroupMap(nodes.value)
    const nodeIds = new Set(nodes.value.map((item) => item.id || '').filter(Boolean))
    const hasExpectedNode = (nodeId: string) =>
      nodeId === startNodeId || nodeId === postStartNodeId || nodeIds.has(nodeId)
    const edges: Edge[] = []
    parallelGroups.forEach((branches, parentId) => {
      if (!hasExpectedNode(parentId)) return
      const nextAfterGroup = nodes.value.filter(
        (item) => (item.parentNodeId || startNodeId) === parentId && item.branchType !== 'parallel'
      )
      branches.forEach((branch) => {
        if (!branch.id) return
        const branchRootType =
          resolveRootNodeId(branch.id || '', byID) === postStartNodeId ? 'post' : 'main'
        edges.push({
          id: `edge-${parentId}-${branch.id}`,
          source: parentId,
          target: branch.id || '',
          type: 'smoothstep',
          animated: !locked.value,
          updatable: !editingDisabled.value,
          interactionWidth: 24,
          markerEnd: MarkerType.ArrowClosed,
          class: ['workflow-edge-parallel', branchRootType === 'post' ? 'workflow-edge-post' : '']
            .filter(Boolean)
            .join(' '),
          pathOptions: { offset: 18, borderRadius: 10 },
          style: {
            stroke: branchRootType === 'post' ? '#f97316' : '#7c3aed',
            strokeWidth: 2.4,
            strokeDasharray: '7 5'
          }
        })
        nextAfterGroup.forEach((nextNode) => {
          if (!nextNode.id) return
          edges.push({
            id: `edge-${branch.id}-${nextNode.id}`,
            source: branch.id || '',
            target: nextNode.id || '',
            type: 'smoothstep',
            animated: !locked.value,
            updatable: !editingDisabled.value,
            interactionWidth: 24,
            markerEnd: MarkerType.ArrowClosed,
            class: [
              'workflow-edge-parallel-exit',
              branchRootType === 'post' ? 'workflow-edge-post' : ''
            ]
              .filter(Boolean)
              .join(' '),
            pathOptions: { offset: 24, borderRadius: 10 },
            style: {
              stroke: branchRootType === 'post' ? '#f97316' : '#7c3aed',
              strokeWidth: 2.4,
              strokeDasharray: '7 5'
            }
          })
        })
      })
    })

    nodes.value.forEach((item) => {
      const parentId = item.parentNodeId || startNodeId
      if (item.branchType === 'parallel' && parallelGroups.has(parentId)) return
      if (parallelGroups.has(parentId)) return
      const source = parentId
      if (!item.id || !hasExpectedNode(source)) return
      const rootType = resolveRootNodeId(item.id || '', byID) === postStartNodeId ? 'post' : 'main'
      edges.push({
        id: `edge-${source}-${item.id}`,
        source,
        target: item.id || '',
        type: 'smoothstep',
        animated: !locked.value,
        updatable: !editingDisabled.value,
        interactionWidth: 24,
        markerEnd: MarkerType.ArrowClosed,
        class: ['workflow-edge-next', rootType === 'post' ? 'workflow-edge-post' : '']
          .filter(Boolean)
          .join(' '),
        pathOptions: {
          offset: 22,
          borderRadius: 10
        },
        style: {
          stroke: rootType === 'post' ? '#f97316' : source === startNodeId ? '#10b981' : '#2563eb',
          strokeWidth: 2.4,
          strokeDasharray: undefined
        }
      })
    })
    return edges
  })

  function emptyStepForm(): WorkflowNode {
    return {
      stepId: '',
      stepCode: '',
      stepName: '',
      nodeName: '',
      containerName: '',
      stepType: 'task',
      icon: defaultIcon,
      iconColor: '#64748b',
      parentNodeId: startNodeId,
      branchType: 'next',
      enabled: true,
      paramValues: ''
    }
  }

  function stepTypeName(type?: string) {
    return type === 'manual_approval' ? '人工审核' : '普通步骤'
  }

  function stepParamModeLabel(param: WorkflowParamDefinition) {
    if (isPlatformStepParam(param)) {
      return normalizeStepRuntimeMode(param.runtimeMode, param.mode) === 'params'
        ? '平台执行变量'
        : '平台静态变量'
    }
    if (param.mode === 'env') return '环境变量'
    return isRuntimeConfigParam(param) ? '执行变量，可运行时配置' : '执行变量'
  }

  function stepParamModeTagType(param: WorkflowParamDefinition) {
    if (isPlatformStepParam(param)) return 'primary'
    return param.mode === 'env' ? 'warning' : 'success'
  }

  function normalizeStepRuntimeMode(runtimeMode?: string, mode?: string) {
    if (runtimeMode === 'params' || runtimeMode === 'env') return runtimeMode
    if (mode === 'env' || mode === 'paas') return 'env'
    return 'params'
  }

  function findLibraryItem(stepId?: string, stepCode?: string) {
    const match = (item: WorkflowLibraryItem) =>
      (stepId && item.id === stepId) || (stepCode && item.code === stepCode)
    return props.libraryItems.find(match) || props.templateItems.find(match)
  }

  function parseParamValues(raw?: string) {
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

  function stringifyParamValues(values: Record<string, any>) {
    const cleaned = Object.entries(values).reduce<Record<string, any>>((acc, [key, value]) => {
      if (Array.isArray(value)) {
        if (value.length > 0) acc[key] = value
        return acc
      }
      if (String(value ?? '').trim() !== '') acc[key] = value
      return acc
    }, {})
    return Object.keys(cleaned).length > 0 ? JSON.stringify(cleaned, null, 2) : ''
  }

  function initStepParamMap(node: WorkflowNode) {
    const params = findLibraryItem(node.stepId, node.stepCode)?.params || []
    const parsed = parseParamValues(node.paramValues)
    stepDynamicOptions.value = {}
    stepDynamicLoading.value = {}
    stepDynamicLoaded.value = {}
    stepDynamicLoadError.value = {}
    stepDynamicQueryKey.value = {}
    objectListDynamicOptions.value = {}
    objectListDynamicLoading.value = {}
    objectListDynamicLoadError.value = {}
    objectListDynamicQueryKey.value = {}
    stepParamMap.value = params.reduce<Record<string, any>>((acc, param) => {
      const value = parsed[param.code]
      if (param.paramType === 'objectList') {
        if (Array.isArray(value)) {
          acc[param.code] = value
          return acc
        }
        if (String(param.defaultValue || '').trim()) {
          try {
            const defaultRows = JSON.parse(param.defaultValue || '[]')
            acc[param.code] = Array.isArray(defaultRows) ? defaultRows : []
          } catch {
            acc[param.code] = []
          }
          return acc
        }
        acc[param.code] = []
        return acc
      }
      if (param.paramType === channelCredentialParamType) {
        acc[param.code] = ''
        return acc
      }
      acc[param.code] =
        value !== undefined && value !== null && value !== '' ? value : param.defaultValue || ''
      return acc
    }, {})
  }

  function isEmptyParamValue(value: any) {
    if (Array.isArray(value)) return value.length === 0
    return String(value ?? '').trim() === ''
  }

  function missingRequiredParams(node: WorkflowNode) {
    if (!enableStepParamConfig.value) return []
    const params = findLibraryItem(node.stepId, node.stepCode)?.params || []
    if (params.length === 0) return []
    const values = parseParamValues(node.paramValues)
    return params.filter(
      (param) =>
        param.required &&
        param.paramType !== channelCredentialParamType &&
        !isRuntimeConfigParam(param) &&
        isEmptyParamValue(values[param.code]) &&
        isEmptyParamValue(param.defaultValue)
    )
  }

  function stringStepParamValue(code: string) {
    const value = stepParamMap.value[code]
    return Array.isArray(value) ? value.join(',') : String(value ?? '')
  }

  function booleanStepParamValue(code: string) {
    const value = stepParamMap.value[code]
    return value === true || String(value).toLowerCase() === 'true'
  }

  function numberStepParamValue(code: string) {
    const value = stepParamMap.value[code]
    if (value === undefined || value === null || value === '') return undefined
    const parsed = Number(value)
    return Number.isNaN(parsed) ? undefined : parsed
  }

  function arrayStepParamValue(code: string) {
    const value = stepParamMap.value[code]
    if (Array.isArray(value)) return value
    return String(value || '')
      .split(',')
      .map((item) => item.trim())
      .filter(Boolean)
  }

  function setStepParamValue(code: string, value: any) {
    const param = currentStepParams.value.find((item) => item.code === code)
    if (param && isStepParamReadonly(param)) return
    stepParamMap.value = { ...stepParamMap.value, [code]: value }
  }

  function setNumberStepParamValue(code: string, value?: number) {
    setStepParamValue(code, value === undefined || value === null ? '' : String(value))
  }

  function isKnownDynamicOptionValue(value: unknown, options: WorkflowParamOption[] | undefined) {
    const text = String(value ?? '').trim()
    return !!text && !!options?.some((option) => String(option.value ?? '').trim() === text)
  }

  function stepDependencyValue(code?: string) {
    if (!code) return ''
    return String(stepParamMap.value[code] ?? '').trim()
  }

  function parseDynamicOptionMetadata(raw?: string) {
    if (!raw) return undefined
    try {
      return JSON.parse(raw)
    } catch {
      return undefined
    }
  }

  function dynamicOptionSecondary(option: WorkflowParamOption) {
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

  function stepDynamicProjectValue(param: WorkflowParamDefinition) {
    return dynamicProjectDependencyValue(param, stepDependencyValue)
  }

  function isGitRepositoryURLValue(value: string) {
    const text = value.trim()
    return /^https?:\/\//.test(text) || /^ssh:\/\//.test(text) || /^[^@\s]+@[^:\s]+:.+/.test(text)
  }

  function isImageRegistryAddressValue(value: string) {
    const text = value.trim()
    if (!text || !text.includes('/')) return false
    return /^https?:\/\//.test(text) || text.includes('.') || text.includes(':')
  }

  function isArtifactRepositoryAddressValue(value: string) {
    const text = value.trim()
    if (!text || !text.includes('/')) return false
    return /^https?:\/\//.test(text) || text.includes('.') || text.includes(':')
  }

  function isGitRefDynamicType(paramType: string) {
    return ['gitBranch', 'gitTag'].includes(paramType)
  }

  function isImageRegistryDynamicType(paramType: string) {
    return ['harborImage', 'harborImageTag', 'registryImage', 'registryImageTag'].includes(
      paramType
    )
  }

  const imageRegistryProjectDependencyFields = [
    'address.imageTagUrl',
    'address.imageUrl',
    'address.projectUrl',
    'address.registryUrl',
    'dynamic.project',
    'dynamic.repository',
    'dynamic.registry'
  ]

  function isArtifactRepositoryDynamicType(paramType: string) {
    return [
      'nexusArtifactName',
      'nexusArtifactVersion',
      'jfrogArtifactName',
      'jfrogArtifactVersion'
    ].includes(paramType)
  }

  function gitRefProjectDependencyParamCode(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    field?: string
  ) {
    if (!isGitRefDynamicType(effectiveDynamicParamType(item))) return ''
    const fields = field ? [field] : ['address.projectUrl', 'dynamic.project']
    return (
      (item.config?.dependencyParamCodes || []).find(
        (dep: { field?: string; paramCode?: string }) =>
          fields.includes(String(dep.field || '').trim())
      )?.paramCode || ''
    )
  }

  function gitRefProjectDependencyValue(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    readValue: (code?: string) => string
  ) {
    for (const field of ['address.projectUrl', 'dynamic.project']) {
      const value = readValue(gitRefProjectDependencyParamCode(item, field))
      if (value) return value
    }
    return item.config?.projectParamCode ? readValue(item.config.projectParamCode) : ''
  }

  function dynamicProjectDependencyValue(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    readValue: (code?: string) => string
  ) {
    if (isGitRefDynamicType(effectiveDynamicParamType(item))) {
      return gitRefProjectDependencyValue(item, readValue)
    }
    if (isImageRegistryDynamicType(effectiveDynamicParamType(item))) {
      for (const field of imageRegistryProjectDependencyFields) {
        const paramCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) => String(dep.field || '').trim() === field
        )?.paramCode
        const value = readValue(paramCode)
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
        const value = readValue(paramCode)
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
      const value = readValue(namespaceParamCode)
      if (value) return value
    }
    const projectCode = item.config?.projectParamCode || gitRefProjectDependencyParamCode(item)
    return projectCode ? readValue(projectCode) : ''
  }

  function channelDynamicParamType(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>
  ) {
    if (String(item.paramType || '').trim() !== 'channelGroupCode') return ''
    const field = String(item.config?.mappingField || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    const groupCode = String(item.config?.channelGroupCode || '').trim()
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
    if (field === 'dynamic.registry') {
      if (channelType === 'harbor') return 'harborProject'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepository'
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
    if (field === 'dynamic.artifact' || field === 'dynamic.artifactName')
      return channelType === 'jfrog' ? 'jfrogArtifactName' : 'nexusArtifactName'
    if (field === 'dynamic.artifactVersion' || field === 'dynamic.artifactTag')
      return channelType === 'jfrog' ? 'jfrogArtifactVersion' : 'nexusArtifactVersion'
    if (field === 'address.repositoryUrl')
      return channelType === 'jfrog' ? 'jfrogRepositoryUrl' : 'nexusRepositoryUrl'
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

  function isDynamicChannelAddressParam(param: WorkflowParamDefinition) {
    return isCompositePlatformParamType(channelDynamicParamType(param))
  }

  function isDynamicChannelAddressField(field: WorkflowCompoundParamField) {
    return isCompositePlatformParamType(channelDynamicParamType(field))
  }

  function platformStepAddressParamType(param: WorkflowParamDefinition) {
    return isDynamicChannelAddressParam(param) ? channelDynamicParamType(param) : param.paramType
  }

  function isChannelDeployConfigParam(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    channelType: string
  ) {
    return (
      String(item.paramType || '').trim() === 'channelGroupCode' &&
      ['dynamic.deployConfig', 'address.deployConfig', 'address.deploymentConfig'].includes(
        String(item.config?.mappingField || '').trim()
      ) &&
      String(item.config?.channelTypeFilter || '').trim() === channelType
    )
  }

  function isKubeNovaDeployConfigParam(param: WorkflowParamDefinition) {
    return (
      param.paramType === kubeNovaDeployConfigParamType ||
      isChannelDeployConfigParam(param, 'kube-nova')
    )
  }

  function isKubernetesDeployConfigParam(param: WorkflowParamDefinition) {
    return (
      param.paramType === kubernetesDeployConfigParamType ||
      isChannelDeployConfigParam(param, 'kubernetes')
    )
  }

  function effectiveDynamicParamType(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>
  ) {
    return channelDynamicParamType(item) || item.paramType
  }

  function isChannelVariableParam(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType'>
  ) {
    return String(item.paramType || '').trim() === 'channelGroupCode'
  }

  function dynamicComponentValue(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    readValue?: (code?: string) => string
  ) {
    if (item.paramType === credentialParamType) return item.config?.voucherModel || ''
    if (isChannelVariableParam(item)) {
      if (
        (isImageRegistryDynamicType(effectiveDynamicParamType(item)) ||
          isArtifactRepositoryDynamicType(effectiveDynamicParamType(item))) &&
        item.config?.componentParamCode
      ) {
        return readValue?.(item.config.componentParamCode) || ''
      }
      if (
        ['kubernetesResource', 'kubernetesContainer', 'kubernetesImage'].includes(
          effectiveDynamicParamType(item)
        )
      ) {
        const workloadParamCode = (item.config?.dependencyParamCodes || []).find(
          (dep: { field?: string; paramCode?: string }) =>
            String(dep.field || '').trim() === 'dynamic.workloadType'
        )?.paramCode
        return readValue?.(workloadParamCode) || item.config?.channelTypeFilter || ''
      }
      return item.config?.channelTypeFilter || ''
    }
    return ''
  }

  function dynamicChannelBindingValue(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    readValue: (code?: string) => string,
    projectValue: string,
    effectiveType = effectiveDynamicParamType(item)
  ) {
    const bindingValue =
      String(item.config?.channelBindingId || '').trim() || readValue(item.config?.channelParamCode)
    if (bindingValue) return bindingValue
    if (isGitRefDynamicType(effectiveType) && isGitRepositoryURLValue(projectValue)) return ''
    if (isImageRegistryDynamicType(effectiveType) && isImageRegistryAddressValue(projectValue))
      return ''
    if (
      isArtifactRepositoryDynamicType(effectiveType) &&
      isArtifactRepositoryAddressValue(projectValue)
    )
      return ''
    return bindingValue
  }

  function channelDependencyValues(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'config'>,
    readValue: (code?: string) => string
  ) {
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

  function channelDependencyValuesText(values: Record<string, string>) {
    return Object.keys(values).length ? JSON.stringify(values) : ''
  }

  function hasMissingChannelDependency(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'config'>,
    readValue: (code?: string) => string
  ) {
    if (isGitRefDynamicType(effectiveDynamicParamType(item))) {
      const hasGitProjectDependency = (item.config?.dependencyParamCodes || []).some(
        (dep: { field?: string }) =>
          ['dynamic.project', 'address.projectUrl'].includes(String(dep.field || '').trim())
      )
      if (hasGitProjectDependency) return !gitRefProjectDependencyValue(item, readValue)
    }
    return (item.config?.dependencyParamCodes || []).some(
      (dep: { paramCode?: string }) =>
        !!String(dep.paramCode || '').trim() && !readValue(dep.paramCode)
    )
  }

  function isDynamicStepParam(param: WorkflowParamDefinition) {
    return (
      isPlatformStepParam(param) &&
      ((isChannelVariableParam(param) &&
        !isDynamicChannelAddressParam(param) &&
        !isKubeNovaDeployConfigParam(param) &&
        !isKubernetesDeployConfigParam(param)) ||
        channelParamTypes.includes(param.paramType) ||
        dynamicParamTypes.includes(param.paramType) ||
        param.paramType === 'kubernetesSecretName' ||
        param.paramType === 'kubernetesPvcName' ||
        param.paramType === credentialParamType)
    )
  }

  function isCompositeStepParam(param: WorkflowParamDefinition) {
    return (
      isPlatformStepParam(param) &&
      isCompositePlatformParamType(platformStepAddressParamType(param))
    )
  }

  function isPlatformStepParam(param: WorkflowParamDefinition) {
    return (
      param.mode === 'paas' ||
      (props.engineType === 'tekton' &&
        (tektonPlatformParamTypes.includes(param.paramType) ||
          isCompositePlatformParamType(param.paramType)))
    )
  }

  function isFullRowParam(param: WorkflowParamDefinition) {
    return (
      param.config?.fullRow === true ||
      isKubeNovaDeployConfigParam(param) ||
      isKubernetesDeployConfigParam(param)
    )
  }

  function isStepParamReadonly(param: WorkflowParamDefinition) {
    return readonly.value || locked.value || param.readonly === true
  }

  function isObjectListFieldReadonly(
    param: WorkflowParamDefinition,
    field: WorkflowCompoundParamField
  ) {
    return isStepParamReadonly(param) || field.readonly === true
  }

  function stepDynamicPlaceholder(param: WorkflowParamDefinition) {
    if (stepDynamicLoadError.value[param.code]) return stepDynamicLoadError.value[param.code]
    if (
      stepDynamicLoaded.value[param.code] &&
      !stepDynamicLoading.value[param.code] &&
      !(stepDynamicOptions.value[param.code] || []).length
    ) {
      return dynamicEmptyText(param.paramType)
    }
    const effectiveType = effectiveDynamicParamType(param)
    const spec = dynamicParamSpec[effectiveType]
    if (
      isChannelVariableParam(param) &&
      hasMissingChannelDependency(param, stepDependencyValue) &&
      !isGitRepositoryURLValue(stepDynamicProjectValue(param)) &&
      !isImageRegistryAddressValue(stepDynamicProjectValue(param)) &&
      !isArtifactRepositoryAddressValue(stepDynamicProjectValue(param))
    ) {
      return param.defaultValue || '可手动输入，选择依赖后可自动获取'
    }
    if (spec?.projectParamType && !stepDynamicProjectValue(param)) {
      return param.defaultValue || '可手动输入，选择依赖后可自动获取'
    }
    if (
      (dynamicParamTypes.includes(effectiveType) || isChannelVariableParam(param)) &&
      spec?.channelParamType &&
      !dynamicChannelBindingValue(param, stepDependencyValue, stepDynamicProjectValue(param)) &&
      !isGitRepositoryURLValue(stepDynamicProjectValue(param)) &&
      !isImageRegistryAddressValue(stepDynamicProjectValue(param)) &&
      !isArtifactRepositoryAddressValue(stepDynamicProjectValue(param))
    ) {
      return param.defaultValue || '可手动输入，选择依赖后可自动获取'
    }
    if (channelParamTypes.includes(param.paramType) || isChannelVariableParam(param)) {
      return param.defaultValue || '请选择渠道实例'
    }
    if (param.paramType === 'kubernetesSecretName') {
      return param.defaultValue || '请选择 Kubernetes Secret'
    }
    if (param.paramType === 'kubernetesPvcName') {
      return param.defaultValue || '请选择 Kubernetes PVC'
    }
    if (param.paramType === credentialParamType) {
      return param.defaultValue || '请选择凭证'
    }
    return param.defaultValue || '请选择'
  }

  function dynamicEmptyText(paramType: string) {
    if (paramType === credentialParamType) return '暂无该类型平台凭证'
    if (paramType === 'kubernetesSecretName') return '暂无 Kubernetes Secret'
    if (paramType === 'kubernetesPvcName') return '暂无 Kubernetes PVC'
    if (paramType === 'gitBranch') return '该 Git 项目暂无可选分支'
    if (paramType === 'gitTag') return '该 Git 项目暂无可选 Tag'
    return '暂无可选项'
  }

  function dynamicErrorFallback(paramType: string) {
    if (paramType === credentialParamType) return '获取平台凭证失败'
    if (paramType === 'gitBranch') return '获取 Git 分支失败'
    if (paramType === 'gitTag') return '获取 Git Tag 失败'
    if (paramType === 'gitProject') return '获取 Git 项目失败'
    if (paramType === 'sonarProjectKey') return '获取 Sonar 项目 Key 失败'
    if (paramType === 'kubernetesSecretName') return '获取 Kubernetes Secret 失败'
    if (paramType === 'kubernetesPvcName') return '获取 Kubernetes PVC 失败'
    return '获取选项失败'
  }

  function extractErrorMessage(error: unknown) {
    if (error instanceof Error) return error.message
    if (typeof error === 'object' && error && 'message' in error) {
      return String((error as { message?: unknown }).message || '')
    }
    return String(error || '')
  }

  function formatDynamicError(paramType: string, error: unknown) {
    const fallback = dynamicErrorFallback(paramType)
    const message = extractErrorMessage(error).trim()
    if (!message || message === '服务器内部错误') {
      return `${fallback}，请检查渠道凭证或项目权限`
    }
    return message.includes(fallback) ? message : `${fallback}：${message}`
  }

  function dynamicQueryKey(
    item: Pick<WorkflowParamDefinition | WorkflowCompoundParamField, 'paramType' | 'config'>,
    channelBindingId: string,
    projectValue: string,
    keyword: string,
    componentValue = '',
    dependencyValues = ''
  ) {
    const configTypeValue =
      item.paramType === 'configCenter'
        ? item.config?.configTypeId || item.config?.configTypeCode || ''
        : ''
    return [
      effectiveDynamicParamType(item),
      componentValue,
      configTypeValue,
      channelBindingId,
      projectValue,
      keyword.trim(),
      dependencyValues
    ].join('|')
  }

  async function queryStepDynamicParamOptions(
    param: WorkflowParamDefinition,
    keyword = '',
    force = false
  ) {
    if (!isDynamicStepParam(param) || !props.dynamicProjectId) return
    if (param.paramType === 'kubernetesSecretName' || param.paramType === 'kubernetesPvcName') {
      const bindingId =
        String(param.config?.channelBindingId || '').trim() ||
        String(props.dynamicBuildChannelBindingId || '').trim()
      if (!bindingId) {
        stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
        stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: false }
        stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
        return
      }
      if (stepDynamicLoading.value[param.code]) return
      const queryKey = dynamicQueryKey(param, bindingId, '', keyword)
      if (
        !force &&
        !keyword.trim() &&
        stepDynamicLoaded.value[param.code] &&
        stepDynamicQueryKey.value[param.code] === queryKey &&
        !stepDynamicLoadError.value[param.code]
      ) {
        return
      }
      stepDynamicQueryKey.value = { ...stepDynamicQueryKey.value, [param.code]: queryKey }
      stepDynamicLoading.value = { ...stepDynamicLoading.value, [param.code]: true }
      stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
      try {
        if (param.paramType === 'kubernetesPvcName') {
          const res = await listDevopsTektonPvcApi({
            projectId: props.dynamicProjectId,
            bindingId,
            keyword
          })
          stepDynamicOptions.value = {
            ...stepDynamicOptions.value,
            [param.code]: (res.items || []).map((pvc) => ({
              label: `${pvc.name}（${pvc.storage || pvc.status || 'PVC'}）`,
              value: pvc.name,
              metadata: {
                namespace: pvc.namespace,
                storageClassName: pvc.storageClassName,
                accessModes: pvc.accessModes || []
              }
            }))
          }
        } else {
          const res = await listDevopsTektonSecretApi({
            projectId: props.dynamicProjectId,
            bindingId,
            keyword
          })
          stepDynamicOptions.value = {
            ...stepDynamicOptions.value,
            [param.code]: (res.items || []).map((secret) => ({
              label: `${secret.name}（${secret.type || 'Opaque'}）`,
              value: secret.name,
              metadata: { namespace: secret.namespace, keys: secret.keys || [] }
            }))
          }
        }
        stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: true }
      } catch (error) {
        stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
        stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: true }
        stepDynamicLoadError.value = {
          ...stepDynamicLoadError.value,
          [param.code]: formatDynamicError(param.paramType, error)
        }
      } finally {
        stepDynamicLoading.value = { ...stepDynamicLoading.value, [param.code]: false }
      }
      return
    }
    const projectValue = stepDynamicProjectValue(param)
    const effectiveType = effectiveDynamicParamType(param)
    const spec = dynamicParamSpec[effectiveType]
    if (dynamicParamSpec[effectiveType]?.projectParamType && !projectValue) {
      stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
      stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: false }
      stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
      return
    }
    const channelBindingId = dynamicChannelBindingValue(
      param,
      stepDependencyValue,
      projectValue,
      effectiveType
    )
    if (
      (dynamicParamTypes.includes(effectiveType) || isChannelVariableParam(param)) &&
      spec?.channelParamType &&
      !channelBindingId &&
      !isGitRepositoryURLValue(projectValue) &&
      !isImageRegistryAddressValue(projectValue) &&
      !isArtifactRepositoryAddressValue(projectValue)
    ) {
      stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
      stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: false }
      stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
      return
    }
    if (stepDynamicLoading.value[param.code]) return
    const componentValue = dynamicComponentValue(param, stepDependencyValue)
    const dependencyValues = channelDependencyValuesText(
      channelDependencyValues(param, stepDependencyValue)
    )
    const queryKey = dynamicQueryKey(
      param,
      channelBindingId,
      projectValue,
      keyword,
      componentValue,
      dependencyValues
    )
    if (
      !force &&
      !keyword.trim() &&
      stepDynamicLoaded.value[param.code] &&
      stepDynamicQueryKey.value[param.code] === queryKey &&
      !stepDynamicLoadError.value[param.code]
    ) {
      return
    }
    stepDynamicQueryKey.value = { ...stepDynamicQueryKey.value, [param.code]: queryKey }
    stepDynamicLoading.value = { ...stepDynamicLoading.value, [param.code]: true }
    stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
    try {
      const res = await listDevopsDynamicParamOptionsApi({
        projectId: props.dynamicProjectId,
        systemId: props.dynamicSystemId,
        environmentId: props.dynamicEnvironmentId,
        paramType: effectiveType,
        channelBindingId,
        projectValue,
        componentValue,
        configTypeId: param.config?.configTypeId || '',
        configTypeCode: isChannelVariableParam(param)
          ? param.config?.channelGroupCode || ''
          : param.config?.configTypeCode || '',
        mappingField: param.config?.mappingField || '',
        dependencyValues,
        keyword,
        page: 1,
        pageSize: 50
      })
      stepDynamicOptions.value = {
        ...stepDynamicOptions.value,
        [param.code]: (res.items || []).map((item) => ({
          label: item.label,
          value: item.value,
          metadata: parseDynamicOptionMetadata(item.metadata)
        }))
      }
      stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: true }
    } catch (error) {
      stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
      stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: true }
      stepDynamicLoadError.value = {
        ...stepDynamicLoadError.value,
        [param.code]: formatDynamicError(effectiveType, error)
      }
    } finally {
      stepDynamicLoading.value = { ...stepDynamicLoading.value, [param.code]: false }
    }
  }

  function clearStepDependentParams(code: string) {
    const next = { ...stepParamMap.value }
    currentStepParams.value.forEach((param) => {
      const dependsOn =
        param.config?.channelParamCode === code ||
        param.config?.projectParamCode === code ||
        param.config?.credentialSourceParamCode === code ||
        (param.config?.dependencyParamCodes || []).some(
          (dep: { paramCode?: string }) => dep.paramCode === code
        )
      if (!dependsOn) return
      if (isStepParamReadonly(param)) return
      if (isKnownDynamicOptionValue(next[param.code], stepDynamicOptions.value[param.code])) {
        next[param.code] = ''
      }
      stepDynamicOptions.value = { ...stepDynamicOptions.value, [param.code]: [] }
      stepDynamicLoaded.value = { ...stepDynamicLoaded.value, [param.code]: false }
      stepDynamicLoadError.value = { ...stepDynamicLoadError.value, [param.code]: '' }
      stepDynamicQueryKey.value = { ...stepDynamicQueryKey.value, [param.code]: '' }
    })
    stepParamMap.value = next
  }

  function updateStepParamValue(param: WorkflowParamDefinition, value: string) {
    if (isStepParamReadonly(param)) return
    setStepParamValue(param.code, value)
    clearStepDependentParams(param.code)
  }

  function handleStepDynamicParamVisible(param: WorkflowParamDefinition, visible: boolean) {
    if (visible && !isStepParamReadonly(param)) queryStepDynamicParamOptions(param, '', true)
  }

  function refreshStepDynamicParamOptions(param: WorkflowParamDefinition) {
    if (isStepParamReadonly(param)) return
    queryStepDynamicParamOptions(param, '', true)
  }

  function objectListRows(param: WorkflowParamDefinition) {
    const value = stepParamMap.value[param.code]
    return Array.isArray(value) ? value : []
  }

  function objectListFields(param: WorkflowParamDefinition): WorkflowCompoundParamField[] {
    const fields = param.config?.compoundFields
    return Array.isArray(fields) ? (fields as WorkflowCompoundParamField[]) : []
  }

  function objectListFieldValue(row: Record<string, any>, field: WorkflowCompoundParamField) {
    const value = row?.[field.code]
    if (field.paramType === 'booleanParam') {
      return value === true || String(value).toLowerCase() === 'true'
    }
    if (field.paramType === 'number' && value !== undefined && value !== '') {
      return Number(value)
    }
    return value ?? ''
  }

  function objectListFieldIsSelect(field: WorkflowCompoundParamField) {
    return (
      field.paramType === 'choice' ||
      field.paramType === 'list' ||
      field.mode === 'paas' ||
      isTektonPlatformObjectListField(field)
    )
  }

  function isDynamicObjectListField(field: WorkflowCompoundParamField) {
    return (
      (field.mode === 'paas' || isTektonPlatformObjectListField(field)) &&
      ((isChannelVariableParam(field) && !isDynamicChannelAddressField(field)) ||
        channelParamTypes.includes(field.paramType) ||
        dynamicParamTypes.includes(field.paramType) ||
        field.paramType === credentialParamType)
    )
  }

  function isTektonPlatformObjectListField(field: WorkflowCompoundParamField) {
    return props.engineType === 'tekton' && tektonPlatformParamTypes.includes(field.paramType)
  }

  function objectListOptionKey(
    param: WorkflowParamDefinition,
    field: WorkflowCompoundParamField,
    rowIndex: number
  ) {
    return `${param.code}.${rowIndex}.${field.code}`
  }

  function objectListFieldOptions(
    param: WorkflowParamDefinition,
    field: WorkflowCompoundParamField,
    rowIndex: number
  ) {
    if (isDynamicObjectListField(field)) {
      return objectListDynamicOptions.value[objectListOptionKey(param, field, rowIndex)] || []
    }
    return field.config?.options || field.selectList || []
  }

  function objectListFieldComponent(field: WorkflowCompoundParamField) {
    if (field.paramType === 'booleanParam') return ElSwitch
    if (field.paramType === 'number') return ElInputNumber
    if (field.paramType === 'text') return ElInput
    if (objectListFieldIsSelect(field)) return ElSelect
    return ElInput
  }

  function objectListFieldProps(
    param: WorkflowParamDefinition,
    row: Record<string, any>,
    rowIndex: number,
    field: WorkflowCompoundParamField
  ) {
    const disabled = isObjectListFieldReadonly(param, field)
    if (field.paramType === 'booleanParam') {
      return {
        disabled
      }
    }
    if (field.paramType === 'number') {
      return {
        disabled,
        controlsPosition: 'right',
        placeholder: field.defaultValue || '请输入数字'
      }
    }
    if (field.paramType === 'text') {
      return {
        type: 'textarea',
        rows: 2,
        disabled,
        placeholder: field.defaultValue || '请输入'
      }
    }
    if (objectListFieldIsSelect(field)) {
      const dynamicField = isDynamicObjectListField(field)
      const key = objectListOptionKey(param, field, rowIndex)
      return {
        class: 'platform-param-select',
        popperClass: 'devops-platform-param-dropdown',
        teleported: false,
        fitInputWidth: true,
        filterable: true,
        clearable: true,
        remote: dynamicField,
        remoteShowSuffix: dynamicField,
        remoteMethod: dynamicField
          ? (keyword: string) => queryObjectListFieldOptions(param, row, rowIndex, field, keyword)
          : undefined,
        loading: objectListDynamicLoading.value[key] === true,
        loadingText: '加载中...',
        noDataText: dynamicEmptyText(field.paramType),
        allowCreate: true,
        defaultFirstOption: true,
        disabled,
        placeholder: objectListDynamicLoadError.value[key] || field.defaultValue || '请选择'
      }
    }
    return {
      disabled,
      type: field.paramType === 'password' ? 'password' : 'text',
      showPassword: field.paramType === 'password',
      placeholder: field.defaultValue || '请输入'
    }
  }

  function normalizeObjectListFieldValue(field: WorkflowCompoundParamField, value: any) {
    if (field.paramType === 'booleanParam') {
      return value === true || String(value).toLowerCase() === 'true'
    }
    if (field.paramType === 'number') {
      return value === undefined || value === null || value === '' ? '' : String(value)
    }
    return value
  }

  function objectListDependencyValue(row: Record<string, any>, code?: string) {
    if (!code) return ''
    return String(row?.[code] ?? '').trim()
  }

  function objectListProjectValue(row: Record<string, any>, field: WorkflowCompoundParamField) {
    return dynamicProjectDependencyValue(field, (code?: string) =>
      objectListDependencyValue(row, code)
    )
  }

  async function queryObjectListFieldOptions(
    param: WorkflowParamDefinition,
    row: Record<string, any>,
    rowIndex: number,
    field: WorkflowCompoundParamField,
    keyword = '',
    force = false
  ) {
    if (!isDynamicObjectListField(field)) return
    const key = objectListOptionKey(param, field, rowIndex)
    const projectValue = objectListProjectValue(row, field)
    const effectiveType = effectiveDynamicParamType(field)
    const channelBindingId = dynamicChannelBindingValue(
      field,
      (code?: string) => objectListDependencyValue(row, code),
      projectValue,
      effectiveType
    )
    if (dynamicParamSpec[effectiveType]?.projectParamType && !projectValue) {
      objectListDynamicOptions.value = { ...objectListDynamicOptions.value, [key]: [] }
      objectListDynamicLoadError.value = { ...objectListDynamicLoadError.value, [key]: '' }
      return
    }
    if (
      !props.dynamicProjectId ||
      (dynamicParamTypes.includes(effectiveType) &&
        dynamicParamSpec[effectiveType]?.channelParamType &&
        !channelBindingId &&
        !isGitRepositoryURLValue(projectValue) &&
        !isImageRegistryAddressValue(projectValue) &&
        !isArtifactRepositoryAddressValue(projectValue))
    ) {
      objectListDynamicOptions.value = { ...objectListDynamicOptions.value, [key]: [] }
      objectListDynamicLoadError.value = { ...objectListDynamicLoadError.value, [key]: '' }
      return
    }
    if (
      !props.dynamicProjectId ||
      (isChannelVariableParam(field) &&
        dynamicParamSpec[effectiveType]?.channelParamType &&
        !channelBindingId &&
        !isGitRepositoryURLValue(projectValue) &&
        !isImageRegistryAddressValue(projectValue) &&
        !isArtifactRepositoryAddressValue(projectValue))
    ) {
      objectListDynamicOptions.value = { ...objectListDynamicOptions.value, [key]: [] }
      objectListDynamicLoadError.value = { ...objectListDynamicLoadError.value, [key]: '' }
      return
    }
    if (objectListDynamicLoading.value[key]) return
    const componentValue = dynamicComponentValue(field, (code?: string) =>
      objectListDependencyValue(row, code)
    )
    const dependencyValues = channelDependencyValuesText(
      channelDependencyValues(field, (code?: string) => objectListDependencyValue(row, code))
    )
    const queryKey = dynamicQueryKey(
      field,
      channelBindingId,
      projectValue,
      keyword,
      componentValue,
      dependencyValues
    )
    if (
      !force &&
      !keyword.trim() &&
      objectListDynamicQueryKey.value[key] === queryKey &&
      !objectListDynamicLoadError.value[key]
    ) {
      return
    }
    objectListDynamicQueryKey.value = { ...objectListDynamicQueryKey.value, [key]: queryKey }
    objectListDynamicLoading.value = { ...objectListDynamicLoading.value, [key]: true }
    objectListDynamicLoadError.value = { ...objectListDynamicLoadError.value, [key]: '' }
    try {
      const res = await listDevopsDynamicParamOptionsApi({
        projectId: props.dynamicProjectId,
        systemId: props.dynamicSystemId,
        environmentId: props.dynamicEnvironmentId,
        paramType: effectiveType,
        channelBindingId,
        projectValue,
        componentValue,
        configTypeId: field.config?.configTypeId || '',
        configTypeCode: isChannelVariableParam(field)
          ? field.config?.channelGroupCode || ''
          : field.config?.configTypeCode || '',
        mappingField: field.config?.mappingField || '',
        dependencyValues,
        keyword,
        page: 1,
        pageSize: 50
      })
      objectListDynamicOptions.value = {
        ...objectListDynamicOptions.value,
        [key]: (res.items || []).map((item) => ({
          label: item.label,
          value: item.value,
          metadata: parseDynamicOptionMetadata(item.metadata)
        }))
      }
    } catch (error) {
      objectListDynamicOptions.value = { ...objectListDynamicOptions.value, [key]: [] }
      objectListDynamicLoadError.value = {
        ...objectListDynamicLoadError.value,
        [key]: formatDynamicError(effectiveType, error)
      }
    } finally {
      objectListDynamicLoading.value = { ...objectListDynamicLoading.value, [key]: false }
    }
  }

  function handleObjectListFieldVisibleChange(
    param: WorkflowParamDefinition,
    row: Record<string, any>,
    rowIndex: number,
    field: WorkflowCompoundParamField,
    visible: boolean
  ) {
    if (visible && !isObjectListFieldReadonly(param, field)) {
      queryObjectListFieldOptions(param, row, rowIndex, field, '', true)
    }
  }

  function refreshObjectListFieldOptions(
    param: WorkflowParamDefinition,
    row: Record<string, any>,
    rowIndex: number,
    field: WorkflowCompoundParamField
  ) {
    if (isObjectListFieldReadonly(param, field)) {
      return
    }
    queryObjectListFieldOptions(param, row, rowIndex, field, '', true)
  }

  function clearObjectListDependentFields(
    param: WorkflowParamDefinition,
    row: Record<string, any>,
    rowIndex: number,
    fieldCode: string
  ) {
    objectListFields(param).forEach((field) => {
      const dependsOnField =
        field.config?.channelParamCode === fieldCode ||
        field.config?.projectParamCode === fieldCode ||
        (field.config?.dependencyParamCodes || []).some(
          (dep: { paramCode?: string }) => dep.paramCode === fieldCode
        )
      if (!dependsOnField) return
      if (isObjectListFieldReadonly(param, field)) return
      row[field.code] = ''
      const key = objectListOptionKey(param, field, rowIndex)
      objectListDynamicOptions.value = { ...objectListDynamicOptions.value, [key]: [] }
      objectListDynamicLoadError.value = { ...objectListDynamicLoadError.value, [key]: '' }
      objectListDynamicQueryKey.value = { ...objectListDynamicQueryKey.value, [key]: '' }
    })
  }

  function blankObjectListRow(param: WorkflowParamDefinition) {
    const row: Record<string, any> = {}
    objectListFields(param).forEach((field) => {
      if (field.paramType === 'booleanParam') {
        row[field.code] = String(field.defaultValue || '').toLowerCase() === 'true'
        return
      }
      row[field.code] = field.defaultValue || ''
    })
    return row
  }

  function addObjectListRow(param: WorkflowParamDefinition) {
    if (isStepParamReadonly(param)) return
    const rows = [...objectListRows(param), blankObjectListRow(param)]
    setStepParamValue(param.code, rows)
  }

  function removeObjectListRow(param: WorkflowParamDefinition, index: number) {
    if (isStepParamReadonly(param)) return
    const rows = [...objectListRows(param)]
    rows.splice(index, 1)
    setStepParamValue(param.code, rows)
  }

  function updateObjectListFieldValue(
    param: WorkflowParamDefinition,
    rowIndex: number,
    fieldCode: string,
    value: any
  ) {
    const rows = [...objectListRows(param)]
    const field = objectListFields(param).find((item) => item.code === fieldCode)
    if (isStepParamReadonly(param) || (field && isObjectListFieldReadonly(param, field))) return
    if (!rows[rowIndex]) rows[rowIndex] = {}
    rows[rowIndex] = {
      ...rows[rowIndex],
      [fieldCode]: field ? normalizeObjectListFieldValue(field, value) : value
    }
    clearObjectListDependentFields(param, rows[rowIndex], rowIndex, fieldCode)
    setStepParamValue(param.code, rows)
  }

  function isStepParamMissing(param: WorkflowParamDefinition) {
    return Boolean(
      param.required &&
        param.paramType !== channelCredentialParamType &&
        !isRuntimeConfigParam(param) &&
        isEmptyParamValue(stepParamMap.value[param.code])
    )
  }

  function isRuntimeConfigParam(param: WorkflowParamDefinition) {
    return (
      param.paramType !== channelCredentialParamType &&
      param.runtimeConfig === true &&
      normalizeStepRuntimeMode(param.runtimeMode, param.mode) === 'params'
    )
  }

  function channelCredentialOutputLabel(param: WorkflowParamDefinition) {
    const mode = normalizeCredentialMode(param.config?.credentialMode, param.config?.mappingField)
    if (mode === 'jenkinsCredentialId') return '输出：Jenkins credentialId'
    if (mode === 'jsonData') return '输出：JSON'
    if (mode === 'jsonBase64') return '输出：JSON Base64'
    return `输出：${credentialMappingFieldLabel(param.config?.mappingField)}`
  }

  function normalizeCredentialMode(mode?: string, mappingField?: string) {
    const value = String(mode || '').trim()
    if (['field', 'jsonData', 'jsonBase64', 'jenkinsCredentialId'].includes(value)) return value
    const legacyField = String(mappingField || '').trim()
    if (legacyField === 'credentialId') return 'jenkinsCredentialId'
    if (legacyField === 'jsonData' || legacyField === 'jsonBase64') return legacyField
    return 'field'
  }

  function credentialMappingFieldLabel(field?: string) {
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

  function syncStepFormParamValues() {
    stepForm.value.paramValues = stringifyParamValues(stepParamMap.value)
  }

  function parallelGroupNodeId(parentId: string) {
    return `parallel-group-${parentId}`
  }

  function parallelGroupParentId(groupNodeId: string) {
    const prefix = 'parallel-group-'
    return groupNodeId.startsWith(prefix) ? groupNodeId.slice(prefix.length) : ''
  }

  function buildParallelGroupMap(items: WorkflowNode[]) {
    const groups = new Map<string, WorkflowNode[]>()
    items.forEach((item) => {
      if (item.branchType !== 'parallel') return
      const parentId = item.parentNodeId || startNodeId
      if (!groups.has(parentId)) groups.set(parentId, [])
      groups.get(parentId)?.push(item)
    })
    Array.from(groups.entries()).forEach(([parentId, children]) => {
      if (children.length < 2) groups.delete(parentId)
    })
    groups.forEach((children) => {
      children.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0))
    })
    return groups
  }

  function parallelGroupLayout(branches: WorkflowNode[]) {
    const minX = Math.min(...branches.map((item) => item.x ?? nodeGapX))
    const minY = Math.min(...branches.map((item) => item.y ?? startY))
    const maxX = Math.max(
      ...branches.map(
        (item) => (item.x ?? nodeGapX) + (nodeWidthMap.value[item.id || ''] || nodeDefaultWidth)
      )
    )
    const maxY = Math.max(...branches.map((item) => (item.y ?? startY) + nodeEstimatedHeight))
    return {
      x: minX - parallelGroupPaddingX,
      y: minY - parallelGroupPaddingY,
      width: Math.max(parallelGroupMinWidth, maxX - minX + parallelGroupPaddingX * 2),
      height: Math.max(
        parallelGroupMinHeight,
        maxY - minY + parallelGroupPaddingY + parallelGroupPaddingY / 2
      )
    }
  }

  function normalizeParallelChildren(items: WorkflowNode[]) {
    return normalizeWorkflowBranches(items, startNodeId)
  }

  function normalizeNodes(items: WorkflowNode[]) {
    const base = normalizeParallelChildren(items).map((item, index) => ({
      ...item,
      id: item.id || `node-${index + 1}`,
      nodeName: item.nodeName || item.stepName || '未命名步骤',
      stepType: item.stepType || 'task',
      branchType: item.branchType || 'next',
      enabled: item.enabled !== false,
      sortOrder: item.sortOrder || index + 1
    }))
    const ids = new Set(base.map((item) => item.id || ''))
    const hasTree = items.some((item) => item.parentNodeId || item.branchType === 'parallel')
    if (!hasTree) {
      let parentId = startNodeId
      base.forEach((item) => {
        item.parentNodeId = parentId
        item.branchType = 'next'
        parentId = item.id || parentId
      })
    } else {
      base.forEach((item) => {
        const parentId = item.parentNodeId || startNodeId
        item.parentNodeId =
          parentId === item.id ||
          (parentId !== startNodeId && parentId !== postStartNodeId && !ids.has(parentId))
            ? startNodeId
            : parentId
      })
    }
    return compactNodes(base, true)
  }

  function compactNodes(items: WorkflowNode[], preservePosition = true) {
    const childrenMap = new Map<string, WorkflowNode[]>()
    const ordered = [...items].sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0))
    ordered.forEach((item) => {
      if (!item.id) return
      const parent = item.parentNodeId || startNodeId
      if (!childrenMap.has(parent)) childrenMap.set(parent, [])
      childrenMap.get(parent)?.push(item)
    })
    childrenMap.forEach((children) => {
      children.sort((left, right) => (left.sortOrder || 0) - (right.sortOrder || 0))
    })

    const placed: WorkflowNode[] = []
    const placedIds = new Set<string>()
    const spanCache = new Map<string, number>()
    const nodeChildren = (nodeId: string) => childrenMap.get(nodeId) || []
    const parallelChildren = (nodeId: string) =>
      nodeChildren(nodeId).filter((child) => child.branchType === 'parallel')
    const nextChildren = (nodeId: string) =>
      nodeChildren(nodeId).filter((child) => child.branchType !== 'parallel')
    const rowSpan = (nodeId: string, stack = new Set<string>()): number => {
      if (!nodeId || stack.has(nodeId)) return 1
      if (spanCache.has(nodeId)) return spanCache.get(nodeId) || 1
      const nextStack = new Set(stack)
      nextStack.add(nodeId)
      const children = nodeChildren(nodeId).filter((child) => child.id && !nextStack.has(child.id))
      const parallel = children.filter((child) => child.branchType === 'parallel')
      const next = children.filter((child) => child.branchType !== 'parallel')
      const parallelSpan =
        parallel.length > 0
          ? parallel.reduce((total, child) => total + rowSpan(child.id || '', nextStack), 0)
          : 0
      const nextSpan =
        next.length > 0
          ? next.reduce((total, child) => total + rowSpan(child.id || '', nextStack), 0)
          : 0
      const span = Math.max(1, parallelSpan, nextSpan)
      spanCache.set(nodeId, Math.max(1, span))
      return Math.max(1, span)
    }
    const placeNode = (item: WorkflowNode, depth: number, row: number) => {
      if (!item.id || placedIds.has(item.id)) return
      placedIds.add(item.id)
      const autoX = depth * nodeGapX
      const autoY = startY + row * nodeGapY
      placed.push({
        ...item,
        x: preservePosition && typeof item.x === 'number' ? item.x : autoX,
        y: preservePosition && typeof item.y === 'number' ? item.y : autoY
      })
      const parallel = parallelChildren(item.id || '').filter(
        (child) => !placedIds.has(child.id || '')
      )
      const next = nextChildren(item.id || '').filter((child) => !placedIds.has(child.id || ''))
      let cursor = row
      parallel.forEach((child) => {
        placeNode(child, depth + 1, cursor)
        cursor += rowSpan(child.id || '')
      })
      const nextDepth = parallel.length > 0 ? depth + 2 : depth + 1
      next.forEach((child, index) => {
        placeNode(child, nextDepth + index, row)
      })
    }

    const placeRoots = (rootId: string, baseRow: number) => {
      const roots = childrenMap.get(rootId) || []
      let cursor = baseRow
      roots.forEach((item) => {
        placeNode(item, 1, cursor)
        cursor += rowSpan(item.id || '')
      })
      return cursor
    }

    const afterMainRow = placeRoots(startNodeId, 0)
    const afterPostRow = placeRoots(postStartNodeId, Math.max(3, afterMainRow + 1))

    let orphanRow = Math.max(afterMainRow, afterPostRow)
    ordered.forEach((item) => {
      if (item.id && !placedIds.has(item.id)) {
        placeNode(item, 1, orphanRow)
        orphanRow += rowSpan(item.id || '')
      }
    })
    return alignPostNodesBelowMain(placed).map((item, index) => ({
      ...item,
      sortOrder: index + 1
    }))
  }

  function alignPostNodesBelowMain(items: WorkflowNode[]) {
    const byID = new Map(items.map((item) => [item.id || '', item]))
    const mainNodes = items.filter(
      (item) => item.id && resolveRootNodeId(item.id, byID) !== postStartNodeId
    )
    const postNodes = items.filter(
      (item) => item.id && resolveRootNodeId(item.id, byID) === postStartNodeId
    )
    if (mainNodes.length === 0 || postNodes.length === 0) return items

    const minPostY = Math.min(...postNodes.map((item) => item.y ?? postStartY))
    const maxMainY = Math.max(...mainNodes.map((item) => item.y ?? startY))
    const expectedPostY = Math.max(postStartY, maxMainY + nodeGapY)
    const offsetY = expectedPostY - minPostY
    if (offsetY <= 0) return items

    const postIds = new Set(postNodes.map((item) => item.id || '').filter(Boolean))
    return items.map((item) =>
      item.id && postIds.has(item.id)
        ? {
            ...item,
            y: Math.round((item.y ?? postStartY) + offsetY)
          }
        : item
    )
  }

  function emitNodes(items: WorkflowNode[], preservePosition = true) {
    emit('update:modelValue', compactNodes(normalizeParallelChildren(items), preservePosition))
  }

  function getPostEntryY() {
    const byID = new Map(nodes.value.map((item) => [item.id || '', item]))
    const postNodes = nodes.value.filter(
      (item) => resolveRootNodeId(item.id || '', byID) === postStartNodeId
    )
    if (postNodes.length > 0) {
      return Math.max(postStartY, Math.min(...postNodes.map((item) => item.y ?? postStartY)))
    }
    const maxMainY = nodes.value.reduce((maxY, item) => {
      const rootId = resolveRootNodeId(item.id || '', byID)
      if (rootId === postStartNodeId) return maxY
      return Math.max(maxY, item.y ?? startY)
    }, startY)
    return Math.max(postStartY, maxMainY + nodeGapY)
  }

  function buildParallelGroupNodes(parallelGroups = buildParallelGroupMap(nodes.value)) {
    const orderedGroups = Array.from(parallelGroups.entries()).sort((left, right) => {
      const leftOrder = Math.min(...left[1].map((item) => item.sortOrder || 0))
      const rightOrder = Math.min(...right[1].map((item) => item.sortOrder || 0))
      if (leftOrder !== rightOrder) return leftOrder - rightOrder
      const leftX = Math.min(...left[1].map((item) => item.x ?? 0))
      const rightX = Math.min(...right[1].map((item) => item.x ?? 0))
      return leftX - rightX
    })
    return orderedGroups.map(([parentId, branches], index) => {
      const groupTitle = `并行组-${index + 1}`
      const layout = parallelGroupLayout(branches)
      return {
        id: parallelGroupNodeId(parentId),
        type: 'parallelGroup',
        position: {
          x: layout.x,
          y: layout.y
        },
        targetPosition: Position.Left,
        sourcePosition: Position.Right,
        draggable: !editingDisabled.value,
        selectable: false,
        data: {
          nodeId: parallelGroupNodeId(parentId),
          nodeName: groupTitle,
          stepName: '',
          stepCode: '',
          stepType: 'parallel_group',
          enabled: true,
          width: layout.width,
          height: layout.height,
          branchCount: branches.length,
          groupTitle
        } as FlowNodeData
      }
    })
  }

  function rebuildFlowNodes() {
    const parallelGroups = buildParallelGroupMap(nodes.value)
    const parallelCountByParent = new Map<string, number>()
    parallelGroups.forEach((children, parentId) =>
      parallelCountByParent.set(parentId, children.length)
    )
    const byID = new Map(nodes.value.map((item) => [item.id || '', item]))
    const postEntryY = getPostEntryY()
    flowNodes.value = [
      {
        id: startNodeId,
        type: 'start',
        position: { x: 0, y: startY },
        sourcePosition: Position.Right,
        draggable: false,
        selectable: false,
        data: { nodeId: startNodeId, entryType: 'start' } as FlowNodeData
      },
      ...(postEntryVisible.value
        ? [
            {
              id: postStartNodeId,
              type: 'start',
              position: { x: 0, y: postEntryY },
              targetPosition: Position.Top,
              sourcePosition: Position.Right,
              draggable: false,
              selectable: false,
              data: { nodeId: postStartNodeId, entryType: 'post' } as FlowNodeData
            }
          ]
        : []),
      ...buildParallelGroupNodes(parallelGroups),
      ...nodes.value.map((item) => {
        const missing = missingRequiredParams(item)
        const isParallel = item.branchType === 'parallel'
        const branchType: FlowNodeData['branchType'] = isParallel ? 'parallel' : 'next'
        const rootType: FlowNodeData['rootType'] =
          resolveRootNodeId(item.id || '', byID) === postStartNodeId ? 'post' : 'main'
        return {
          id: item.id || '',
          type: 'pipeline',
          position: { x: item.x ?? nodeGapX, y: item.y ?? startY },
          targetPosition: Position.Left,
          sourcePosition: Position.Right,
          draggable: !editingDisabled.value,
          data: {
            nodeId: item.id || '',
            nodeName: item.nodeName || item.stepName || '未命名步骤',
            stepName: item.stepName || '',
            stepCode: item.stepCode || '',
            stepType: item.stepType || 'task',
            branchType,
            rootType,
            childCount: parallelCountByParent.get(item.id || '') || 0,
            icon: item.icon || defaultIcon,
            iconColor: item.iconColor || '#64748b',
            width: nodeWidthMap.value[item.id || ''] || nodeDefaultWidth,
            enabled: item.enabled !== false,
            missingRequired: missing.length > 0,
            missingRequiredText:
              missing.length > 0 ? `缺少必填参数：${missing[0].name || missing[0].code}` : ''
          }
        }
      })
    ]
    scheduleFlowNodeInternalsRefresh()
  }

  function getStructureSignature(items: WorkflowNode[]) {
    return items
      .map((item) => [item.id, item.parentNodeId, item.branchType, item.stepId].join(':'))
      .join('|')
  }

  function handleNodeClick(event: { node: Node<FlowNodeData> }) {
    if (isRootNodeId(event.node.id) || event.node.type === 'parallelGroup') {
      handlePaneClick()
      return
    }
    selectedNodeId.value = event.node.id
    if (readonly.value) {
      openEditDialog(event.node.id)
      return
    }
    if (!locked.value) openEditDialog(event.node.id)
  }

  function handlePaneClick() {
    selectedNodeId.value = ''
    closeContextMenu()
  }

  function findParallelGroupBranches(parentId: string) {
    return buildParallelGroupMap(nodes.value).get(parentId) || []
  }

  function handleNodeDragStart(event: NodeDragEvent) {
    if (editingDisabled.value || event.node.type !== 'parallelGroup') return
    const parentId = parallelGroupParentId(event.node.id)
    const branches = findParallelGroupBranches(parentId)
    if (!parentId || branches.length < 2) return

    const flowPositionMap = new Map(flowNodes.value.map((item) => [item.id, item.position]))
    const branchStartPositions = branches.reduce<Record<string, { x: number; y: number }>>(
      (acc, branch) => {
        if (!branch.id) return acc
        const position = flowPositionMap.get(branch.id)
        acc[branch.id] = {
          x: position?.x ?? branch.x ?? nodeGapX,
          y: position?.y ?? branch.y ?? startY
        }
        return acc
      },
      {}
    )
    parallelGroupDragState.value = {
      groupId: event.node.id,
      parentId,
      startX: event.node.position.x,
      startY: event.node.position.y,
      branchStartPositions
    }
    syncingParallelGroupDrag = true
  }

  function syncParallelGroupDragNodes(event: NodeDragEvent) {
    const dragState = parallelGroupDragState.value
    if (!dragState || dragState.groupId !== event.node.id) return null
    const deltaX = event.node.position.x - dragState.startX
    const deltaY = event.node.position.y - dragState.startY
    flowNodes.value = flowNodes.value.map((node) => {
      if (node.id === dragState.groupId) {
        return { ...node, position: { x: event.node.position.x, y: event.node.position.y } }
      }
      const startPosition = dragState.branchStartPositions[node.id]
      if (!startPosition) return node
      return {
        ...node,
        position: {
          x: startPosition.x + deltaX,
          y: startPosition.y + deltaY
        }
      }
    })
    return { deltaX, deltaY, dragState }
  }

  function handleNodeDrag(event: NodeDragEvent) {
    if (event.node.type !== 'parallelGroup') return
    syncingParallelGroupDrag = true
    syncParallelGroupDragNodes(event)
  }

  function handleNodeDragStop(event: NodeDragEvent) {
    if (event.node.type !== 'parallelGroup') return
    const result = syncParallelGroupDragNodes(event)
    if (!result) {
      syncingParallelGroupDrag = false
      return
    }
    const { deltaX, deltaY, dragState } = result
    const next = nodes.value.map((item) => {
      if (!item.id) return item
      const startPosition = dragState.branchStartPositions[item.id]
      if (!startPosition) return item
      return {
        ...item,
        x: Math.round(startPosition.x + deltaX),
        y: Math.round(startPosition.y + deltaY)
      }
    })
    emit('update:modelValue', compactNodes(next, true))
    parallelGroupDragState.value = null
    nextTick(() => {
      syncingParallelGroupDrag = false
    })
  }

  function updateNode(nodeId: string, patch: Partial<WorkflowNode>) {
    if (editingDisabled.value || !nodeId) return
    emitNodes(nodes.value.map((item) => (item.id === nodeId ? { ...item, ...patch } : item)))
  }

  function startNodeResize(event: PointerEvent, nodeId: string) {
    if (editingDisabled.value || !nodeId) return
    const startX = event.clientX
    const startWidth = nodeWidthMap.value[nodeId] || nodeDefaultWidth
    const updateWidth = (nextWidth: number) => {
      const width = Math.min(nodeMaxWidth, Math.max(nodeMinWidth, Math.round(nextWidth)))
      nodeWidthMap.value = { ...nodeWidthMap.value, [nodeId]: width }
      flowNodes.value = flowNodes.value.map((node) => {
        if (node.id !== nodeId) return node
        const data = node.data as FlowNodeData
        return {
          ...node,
          data: {
            ...data,
            nodeId: data.nodeId || nodeId,
            nodeName: data.nodeName || data.stepName || '未命名步骤',
            stepName: data.stepName || '',
            stepCode: data.stepCode || '',
            stepType: data.stepType || 'task',
            enabled: data.enabled !== false,
            width
          }
        }
      })
    }
    const handleMove = (moveEvent: PointerEvent) => {
      updateWidth(startWidth + moveEvent.clientX - startX)
    }
    const handleUp = () => {
      document.removeEventListener('pointermove', handleMove)
      document.removeEventListener('pointerup', handleUp)
    }
    document.addEventListener('pointermove', handleMove)
    document.addEventListener('pointerup', handleUp, { once: true })
  }

  function removeNode(nodeId: string) {
    if (editingDisabled.value || !nodeId || nodeId === startNodeId || nodeId === postStartNodeId) {
      return
    }
    const removed = nodes.value.find((item) => item.id === nodeId)
    if (!removed) return
    const parentId = removed.parentNodeId || startNodeId
    const next = nodes.value
      .filter((item) => item.id !== nodeId)
      .map((item) =>
        item.parentNodeId === nodeId
          ? { ...item, parentNodeId: parentId, branchType: 'next' }
          : item
      )
    emitNodes(next, false)
    selectedNodeId.value = next[0]?.id || ''
    closeContextMenu()
    fitCurrentViewDeferred()
  }

  function flowViewport() {
    return vueFlowRef.value as
      | {
          setViewport?: (
            transform: { x: number; y: number; zoom: number },
            options?: { duration?: number }
          ) => Promise<boolean>
          getViewport?: () => { x: number; y: number; zoom: number }
          fitView?: (options?: FitViewParams) => Promise<boolean>
          updateNodeInternals?: (ids?: string[]) => void
        }
      | undefined
  }

  function syncViewportState() {
    const viewport = flowViewport()?.getViewport?.()
    if (!viewport) return
    viewportState.value = {
      x: viewport.x,
      y: viewport.y,
      zoom: viewport.zoom
    }
  }

  function handleViewportChange(viewport?: { x: number; y: number; zoom: number }) {
    if (!viewport) {
      syncViewportState()
      return
    }
    viewportState.value = {
      x: viewport.x,
      y: viewport.y,
      zoom: viewport.zoom
    }
  }

  function flowNodeWidth(node: Node<FlowNodeData>) {
    if (node.type === 'start') return 240
    if (node.type === 'parallelGroup') return node.data?.width || parallelGroupMinWidth
    return node.data?.width || nodeDefaultWidth
  }

  function flowNodeHeight(node: Node<FlowNodeData>) {
    if (node.type === 'parallelGroup') return node.data?.height || parallelGroupMinHeight
    return nodeEstimatedHeight
  }

  function flowBounds() {
    if (flowNodes.value.length === 0) {
      return { minX: 0, minY: startY, width: 240, height: nodeEstimatedHeight }
    }
    const boxes = flowNodes.value.map((node) => ({
      x: node.position.x,
      y: node.position.y,
      width: flowNodeWidth(node),
      height: flowNodeHeight(node)
    }))
    const minX = Math.min(...boxes.map((item) => item.x))
    const minY = Math.min(...boxes.map((item) => item.y))
    const maxX = Math.max(...boxes.map((item) => item.x + item.width))
    const maxY = Math.max(...boxes.map((item) => item.y + item.height))
    return {
      minX,
      minY,
      width: Math.max(240, maxX - minX),
      height: Math.max(nodeEstimatedHeight, maxY - minY)
    }
  }

  function readableZoom() {
    const body = canvasBodyRef.value
    if (!body) return 1
    const bounds = flowBounds()
    const availableWidth = Math.max(520, body.clientWidth - entryViewportX - 56)
    const availableHeight = Math.max(360, body.clientHeight - entryViewportY - 72)
    const zoomByWidth = availableWidth / bounds.width
    const zoomByHeight = availableHeight / bounds.height
    const targetZoom = Math.min(1, zoomByWidth, zoomByHeight)
    return Math.max(readableFitMinZoom, Math.min(1, targetZoom || 1))
  }

  function focusEntryNode(entryNodeId = startNodeId) {
    const flow = flowViewport()
    const entryNode = flowNodes.value.find((node) => node.id === entryNodeId) || flowNodes.value[0]
    if (!flow?.setViewport || !entryNode) {
      return flow?.fitView?.(fitViewParams) || Promise.resolve(false)
    }
    const zoom = readableZoom()
    return flow.setViewport(
      {
        x: entryViewportX - entryNode.position.x * zoom,
        y: entryViewportY - entryNode.position.y * zoom,
        zoom
      },
      { duration: fitViewParams.duration }
    )
  }

  function fitCurrentView() {
    void arrangeAndFitCurrentView()
  }

  function zoomCanvas(direction: 'in' | 'out') {
    const flow = flowViewport()
    const currentZoom = flow?.getViewport?.().zoom || 1
    const factor = direction === 'in' ? 1.16 : 1 / 1.16
    const nextZoom = Math.min(
      1.35,
      Math.max(minCanvasZoom, Number((currentZoom * factor).toFixed(3)))
    )
    const entryNode = flowNodes.value.find((node) => node.id === startNodeId)
    if (!entryNode || !flow?.setViewport) return
    void flow
      .setViewport(
        {
          x: entryViewportX - entryNode.position.x * nextZoom,
          y: entryViewportY - entryNode.position.y * nextZoom,
          zoom: nextZoom
        },
        { duration: 160 }
      )
      .then(() => {
        syncViewportState()
        return refreshFlowNodeInternalsRepeated(2)
      })
  }

  const waitFrame = () =>
    new Promise<void>((resolve) => {
      window.requestAnimationFrame(() => resolve())
    })

  const waitDelay = (delay: number) =>
    new Promise<void>((resolve) => {
      window.setTimeout(() => resolve(), delay)
    })

  async function waitRenderedFlowNodes() {
    const expected = Math.max(flowNodes.value.length, 1)
    for (let index = 0; index < 12; index += 1) {
      await nextTick()
      await waitFrame()
      const rendered = canvasBodyRef.value?.querySelectorAll('.vue-flow__node').length || 0
      if (rendered >= expected) return
      await waitDelay(40)
    }
  }

  function refreshFlowNodeInternals() {
    const ids = flowNodes.value.map((node) => node.id).filter(Boolean)
    flowViewport()?.updateNodeInternals?.(ids)
  }

  async function refreshFlowNodeInternalsRepeated(times = 3) {
    for (let index = 0; index < times; index += 1) {
      await nextTick()
      await waitFrame()
      refreshFlowNodeInternals()
    }
  }

  function scheduleFlowNodeInternalsRefresh() {
    void refreshFlowNodeInternalsRepeated(3)
  }

  function fitCurrentViewDeferred() {
    void fitCurrentViewAfterLayout()
  }

  async function arrangeAndFitCurrentView() {
    closeContextMenu()
    selectedNodeId.value = ''
    if (!readonly.value && nodes.value.length > 0) {
      emit('update:modelValue', compactNodes(nodes.value, false))
      await nextTick()
      rebuildFlowNodes()
    }
    await fitCurrentViewAfterLayout()
  }

  async function fitCurrentViewAfterLayout() {
    await waitRenderedFlowNodes()
    await refreshFlowNodeInternalsRepeated(2)
    await waitFrame()
    await focusEntryNode(startNodeId)
    syncViewportState()
    await waitDelay(80)
    await refreshFlowNodeInternalsRepeated(2)
    await focusEntryNode(startNodeId)
    syncViewportState()
    await waitDelay(80)
    await refreshFlowNodeInternalsRepeated(2)
  }

  function handleNodesInitialized() {
    syncViewportState()
    if (props.autoFitOnMount) fitCurrentViewDeferred()
  }

  function handleInteractionChange(active: boolean) {
    locked.value = !active
    closeContextMenu()
    nextTick(() => rebuildFlowNodes())
  }

  function showPostEntry() {
    if (!enablePostEntry.value) return
    postEntryShown.value = true
    rebuildFlowNodes()
    nextTick(async () => {
      await refreshFlowNodeInternalsRepeated(4)
      await focusEntryNode(startNodeId)
      await refreshFlowNodeInternalsRepeated(4)
    })
  }

  function hasPostNode(items: WorkflowNode[]) {
    const byID = new Map(items.map((item) => [item.id || '', item]))
    return items.some((item) => resolveRootNodeId(item.id || '', byID) === postStartNodeId)
  }

  function isPostAnchor(nodeId: string) {
    return resolveRootNodeId(nodeId) === postStartNodeId
  }

  function isRootNodeId(nodeId: string) {
    return nodeId === startNodeId || nodeId === postStartNodeId
  }

  function isDescendantOf(nodeId: string, possibleAncestorId: string) {
    const byID = new Map(nodes.value.map((item) => [item.id || '', item]))
    let parentId = byID.get(nodeId)?.parentNodeId || ''
    const visited = new Set<string>()
    while (parentId && parentId !== startNodeId && parentId !== postStartNodeId) {
      if (parentId === possibleAncestorId) return true
      if (visited.has(parentId)) return false
      visited.add(parentId)
      parentId = byID.get(parentId)?.parentNodeId || ''
    }
    return parentId === possibleAncestorId
  }

  function isValidWorkflowConnection(connection: Connection) {
    if (editingDisabled.value) return false
    const source = connection.source
    const target = connection.target
    if (!source || !target || source === target || isRootNodeId(target)) return false
    if (source.startsWith('parallel-group-') || target.startsWith('parallel-group-')) return false
    const sourceExists = isRootNodeId(source) || nodes.value.some((item) => item.id === source)
    const targetExists = nodes.value.some((item) => item.id === target)
    if (!sourceExists || !targetExists) return false
    return !isDescendantOf(source, target)
  }

  function reconnectNode(targetNodeId: string, sourceNodeId: string) {
    if (!isValidWorkflowConnection({ source: sourceNodeId, target: targetNodeId })) return
    const targetNode = nodes.value.find((item) => item.id === targetNodeId)
    if (!targetNode) return
    const currentParentId = targetNode.parentNodeId || startNodeId
    emitNodes(
      nodes.value.map((item) => {
        if (item.id === targetNodeId) {
          return {
            ...item,
            parentNodeId: sourceNodeId,
            branchType: 'next'
          }
        }
        if (
          item.id !== targetNodeId &&
          (item.parentNodeId || startNodeId) === sourceNodeId &&
          item.branchType !== 'parallel'
        ) {
          return { ...item, parentNodeId: targetNodeId, branchType: 'next' }
        }
        if (
          item.parentNodeId === targetNodeId &&
          currentParentId === sourceNodeId &&
          targetNode.branchType !== 'parallel'
        ) {
          return { ...item, parentNodeId: currentParentId, branchType: item.branchType || 'next' }
        }
        return item
      }),
      false
    )
    selectedNodeId.value = targetNodeId
    fitCurrentViewDeferred()
  }

  function handleConnect(connection: Connection) {
    reconnectNode(connection.target, connection.source)
  }

  function handleEdgeUpdate(event: EdgeUpdateEvent) {
    reconnectNode(event.connection.target, event.connection.source)
  }

  function resolveRootNodeId(nodeId: string, sourceMap?: Map<string, WorkflowNode>) {
    if (nodeId === postStartNodeId) return postStartNodeId
    if (nodeId === startNodeId || !nodeId) return startNodeId

    const byID = sourceMap || new Map(nodes.value.map((item) => [item.id || '', item]))
    let parentID = byID.get(nodeId)?.parentNodeId || startNodeId
    const visited = new Set<string>()
    while (parentID && parentID !== startNodeId && parentID !== postStartNodeId) {
      if (visited.has(parentID)) return startNodeId
      visited.add(parentID)
      parentID = byID.get(parentID)?.parentNodeId || startNodeId
    }
    return parentID === postStartNodeId ? postStartNodeId : startNodeId
  }

  function openContextMenu(event: MouseEvent, nodeId: string) {
    if (editingDisabled.value) return
    const rect = canvasBodyRef.value?.getBoundingClientRect()
    contextMenu.value = {
      visible: true,
      x: rect ? event.clientX - rect.left + 8 : event.offsetX + 8,
      y: rect ? event.clientY - rect.top + 8 : event.offsetY + 8,
      nodeId
    }
    if (nodeId !== startNodeId && nodeId !== postStartNodeId) selectedNodeId.value = nodeId
  }

  function closeContextMenu() {
    contextMenu.value.visible = false
  }

  function handleDialogCategoryChange(value: string | number | boolean | Record<string, any>) {
    if (isPostDialog.value) {
      const categoryId = resolvedPostCategoryId.value
      if (categoryId) emit('update:categoryId', categoryId)
      nextTick(() => refreshLibraryForDialog(categoryId))
      return
    }
    const categoryId = typeof value === 'string' ? value : ''
    emit('update:categoryId', categoryId)
    nextTick(() => refreshLibraryForDialog(categoryId))
  }

  function refreshLibraryForDialog(categoryId = dialogCategoryId.value) {
    const nextCategoryId = categoryId || dialogCategoryId.value
    if (nextCategoryId && nextCategoryId !== props.categoryId) {
      emit('update:categoryId', nextCategoryId)
    }
    emit('refresh-library', { categoryId: nextCategoryId })
  }

  function parallelParentOf(nodeId: string) {
    const node = nodes.value.find((item) => item.id === nodeId)
    return node?.branchType === 'parallel' ? node.parentNodeId || startNodeId : ''
  }

  function uniqueNodeName(baseName: string, ignoreNodeId = '') {
    const base = String(baseName || '未命名步骤').trim() || '未命名步骤'
    const usedNames = nodes.value
      .filter((item) => item.id !== ignoreNodeId)
      .map((item) => String(item.nodeName || item.stepName || '').trim())
      .filter(Boolean)
    if (!usedNames.includes(base)) {
      return base
    }
    const usedSet = new Set(usedNames)
    for (let index = 1; ; index++) {
      const name = `${base}-${index}`
      if (!usedSet.has(name)) return name
    }
  }

  function resolveAddRelation(_anchorId: string, relation: AddRelation): AddRelation {
    if (relation === 'before') return relation
    if (relation === 'parallelSibling') return relation
    if (relation === 'parallelChild') return relation
    return 'next'
  }

  function openAddDialog(anchorId: string, relation: AddRelation) {
    if (editingDisabled.value) return
    closeContextMenu()
    dialogAnchorId.value = anchorId || startNodeId
    dialogRelation.value = resolveAddRelation(dialogAnchorId.value, relation)
    stepDialogMode.value = 'add'
    stepDialogStage.value = 'select'
    stepKeyword.value = ''
    chosenStep.value = null
    stepForm.value = emptyStepForm()
    stepDialogVisible.value = true
    let categoryId = props.categoryId || defaultNormalCategoryId.value
    if (isPostAnchor(dialogAnchorId.value) && resolvedPostCategoryId.value) {
      categoryId = resolvedPostCategoryId.value
      emit('update:categoryId', categoryId)
    } else if (resolvedPostCategoryId.value && props.categoryId === resolvedPostCategoryId.value) {
      categoryId = defaultNormalCategoryId.value
      if (categoryId) emit('update:categoryId', categoryId)
    } else if (categoryId && categoryId !== props.categoryId) {
      emit('update:categoryId', categoryId)
    }
    nextTick(() => refreshLibraryForDialog(categoryId))
  }

  function chooseStep(item: WorkflowLibraryItem) {
    if (isPostChoiceDisabled(item)) {
      ElMessage.warning(postChoiceDisabledReason(item))
      return
    }
    chosenStep.value = item
    stepForm.value = {
      ...emptyStepForm(),
      stepId: item.id,
      stepCode: item.code,
      stepName: item.name,
      nodeName: uniqueNodeName(item.name),
      stepType: item.type || 'task',
      icon: item.icon || defaultIcon,
      iconColor: item.iconColor || '#64748b'
    }
    initStepParamMap(stepForm.value)
    stepDialogStage.value = 'config'
  }

  function openEditDialog(nodeId: string) {
    if (locked.value || !nodeId || nodeId === startNodeId || nodeId === postStartNodeId) return
    const node = nodes.value.find((item) => item.id === nodeId)
    if (!node) return
    closeContextMenu()
    selectedNodeId.value = node.id || ''
    stepDialogMode.value = 'edit'
    stepDialogStage.value = 'config'
    stepForm.value = { ...node, nodeName: node.nodeName || node.stepName || '未命名步骤' }
    initStepParamMap(stepForm.value)
    stepDialogVisible.value = true
  }

  function backToStepSelect() {
    stepDialogStage.value = 'select'
  }

  function saveStepDialog() {
    if (editingDisabled.value) return
    if (!enableStepParamConfig.value) {
      stepForm.value.paramValues = ''
    } else if (currentStepParams.value.length > 0) {
      syncStepFormParamValues()
    } else if (String(stepForm.value.paramValues || '').trim()) {
      const parsed = parseParamValues(stepForm.value.paramValues)
      if (Object.keys(parsed).length === 0) {
        ElMessage.warning('参数配置必须是 JSON 对象')
        return
      }
    }
    if (stepDialogMode.value === 'edit') {
      const nodeName = uniqueNodeName(
        stepForm.value.nodeName || stepForm.value.stepName || '未命名步骤',
        stepForm.value.id || ''
      )
      updateNode(stepForm.value.id || '', { ...stepForm.value, nodeName })
      stepDialogVisible.value = false
      return
    }
    if (!chosenStep.value) return
    const nodeId = `node-${Date.now()}`
    const anchorId = dialogAnchorId.value || startNodeId
    const relation = resolveAddRelation(anchorId, dialogRelation.value)
    const anchorNode = nodes.value.find((item) => item.id === anchorId)
    const parallelParentId = parallelParentOf(anchorId)
    const isBeforeParallelGroup = Boolean(parallelParentId && relation === 'before')
    const isAfterParallelGroup = Boolean(parallelParentId && relation === 'next')
    const effectiveAnchorId =
      isBeforeParallelGroup || isAfterParallelGroup ? parallelParentId : anchorId
    const effectiveAnchorNode =
      effectiveAnchorId === anchorId
        ? anchorNode
        : nodes.value.find((item) => item.id === effectiveAnchorId)
    const parallelSiblingParentId =
      relation === 'parallelSibling' && !isRootNodeId(anchorId)
        ? anchorNode?.parentNodeId || startNodeId
        : ''
    const shouldLiftAnchorChildren = Boolean(
      parallelSiblingParentId && anchorNode?.branchType !== 'parallel'
    )
    const parentNodeId = isBeforeParallelGroup
      ? effectiveAnchorId
      : relation === 'before' && effectiveAnchorNode
        ? effectiveAnchorNode.parentNodeId || startNodeId
        : parallelSiblingParentId || effectiveAnchorId
    const branchType =
      relation === 'parallelSibling' || relation === 'parallelChild'
        ? 'parallel'
        : relation === 'before' && effectiveAnchorNode && !isBeforeParallelGroup
          ? effectiveAnchorNode.branchType || 'next'
          : 'next'
    const nodeName = uniqueNodeName(stepForm.value.nodeName || chosenStep.value.name)
    const node: WorkflowNode = {
      ...stepForm.value,
      nodeName,
      id: nodeId,
      parentNodeId,
      branchType
    }
    const hasExistingParallelChildren = nodes.value.some(
      (item) => item.parentNodeId === effectiveAnchorId && item.branchType === 'parallel'
    )
    const shouldPromoteAnchorChildren = relation === 'parallelChild' && !hasExistingParallelChildren
    const next: WorkflowNode[] = nodes.value.map((item) => {
      if (isBeforeParallelGroup && item.parentNodeId === effectiveAnchorId) {
        return { ...item, parentNodeId: nodeId, branchType: item.branchType || 'next' }
      }
      if (!isBeforeParallelGroup && relation === 'before' && item.id === effectiveAnchorId) {
        return { ...item, parentNodeId: nodeId, branchType: 'next' }
      }
      if (relation === 'parallelSibling' && item.id === anchorId) {
        return { ...item, branchType: 'parallel' }
      }
      if (shouldLiftAnchorChildren && item.parentNodeId === anchorId) {
        return { ...item, parentNodeId: parallelSiblingParentId, branchType: 'next' }
      }
      if (shouldPromoteAnchorChildren && item.parentNodeId === effectiveAnchorId) {
        return { ...item, branchType: 'parallel' }
      }
      if (relation !== 'next' || item.parentNodeId !== effectiveAnchorId) return item
      if (isAfterParallelGroup && item.branchType === 'parallel') return item
      return { ...item, parentNodeId: nodeId, branchType: item.branchType || 'next' }
    })
    next.push(node)
    emitNodes(next, false)
    selectedNodeId.value = nodeId
    stepDialogVisible.value = false
    fitCurrentViewAfterLayout()
  }

  function resetStepDialog() {
    chosenStep.value = null
    stepDialogStage.value = 'select'
    stepParamMap.value = {}
  }

  let lastStructureSignature = ''
  let modelWatcherReady = false
  watch(
    () => props.modelValue,
    (value) => {
      const currentSignature = getStructureSignature(value || [])
      const shouldFit = modelWatcherReady && currentSignature !== lastStructureSignature
      lastStructureSignature = currentSignature
      rebuildFlowNodes()
      if (selectedNodeId.value && !nodes.value.some((item) => item.id === selectedNodeId.value)) {
        selectedNodeId.value = ''
      }
      if (shouldFit) fitCurrentViewDeferred()
      modelWatcherReady = true
    },
    { immediate: true, deep: true }
  )

  watch(
    [() => props.libraryItems, () => props.templateItems],
    () => {
      rebuildFlowNodes()
      if (
        stepDialogVisible.value &&
        stepDialogStage.value === 'config' &&
        Object.keys(stepParamMap.value).length === 0
      ) {
        initStepParamMap(stepForm.value)
      }
    },
    { deep: true }
  )

  watch(
    postEntryVisible,
    (visible) => {
      if (visible) void refreshFlowNodeInternalsRepeated(4)
    },
    { immediate: true }
  )

  watch(
    flowNodes,
    (items: Node<FlowNodeData>[]) => {
      if (editingDisabled.value || syncingParallelGroupDrag) return
      const positionMap = new Map<string, { x: number; y: number }>(
        items
          .filter((item) => item.type !== 'parallelGroup')
          .map((item) => [item.id, item.position])
      )
      const next = nodes.value.map((item) => {
        const position = positionMap.get(item.id || '')
        return position ? { ...item, x: Math.round(position.x), y: Math.round(position.y) } : item
      })
      const changed = next.some(
        (item, index) => item.x !== nodes.value[index]?.x || item.y !== nodes.value[index]?.y
      )
      if (changed) emit('update:modelValue', next)
    },
    { deep: true }
  )

  onMounted(() => {
    rebuildFlowNodes()
    nextTick(syncViewportState)
    if (props.autoFitOnMount) fitCurrentViewDeferred()
  })

  defineExpose({ fitCurrentView, fitCurrentViewAfterLayout })
</script>

<style scoped lang="scss">
  .devops-workflow-designer {
    width: 100%;
    height: 100%;
    min-height: 620px;
  }

  .workflow-canvas {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 620px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .canvas-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    min-height: 56px;
    padding: 10px 12px;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .canvas-left-actions {
    display: inline-flex;
    flex: none;
    align-items: center;
  }

  .canvas-title {
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
    }

    span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .canvas-actions {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    justify-content: flex-end;
  }

  .canvas-body {
    position: relative;
    flex: 1;
    min-height: 520px;
    overflow: hidden;
    background:
      linear-gradient(90deg, rgb(148 163 184 / 7%) 1px, transparent 1px),
      linear-gradient(rgb(148 163 184 / 7%) 1px, transparent 1px), #f8fafc;
    background-size: 32px 32px;

    :deep(.vue-flow__controls) {
      overflow: hidden;
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      box-shadow: 0 10px 24px rgb(15 23 42 / 10%);
    }

    :deep(.vue-flow__minimap) {
      overflow: hidden;
      background: rgb(255 255 255 / 88%);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;
      box-shadow: 0 14px 34px rgb(15 23 42 / 8%);
    }

    :deep(.vue-flow__edge-path) {
      stroke-linecap: round;
      stroke-linejoin: round;
    }

    :deep(.workflow-edge-parallel .vue-flow__edge-path),
    :deep(.workflow-edge-parallel-group .vue-flow__edge-path),
    :deep(.workflow-edge-parallel-exit .vue-flow__edge-path) {
      filter: drop-shadow(0 1px 2px rgb(124 58 237 / 16%));
    }

    :deep(.workflow-edge-post .vue-flow__edge-path) {
      filter: drop-shadow(0 1px 2px rgb(249 115 22 / 16%));
    }
  }

  .flow-view {
    width: 100%;
    height: 100%;
  }

  .post-entry-link-layer {
    position: absolute;
    inset: 0;
    z-index: 0;
    pointer-events: none;
  }

  .post-entry-link-path {
    filter: drop-shadow(0 1px 1px rgb(249 115 22 / 10%));
    fill: none;
    stroke: #f97316;
    stroke-dasharray: 4 5;
    stroke-linecap: round;
    stroke-linejoin: round;
    stroke-width: 1.6;
  }

  :deep(.vue-flow__node-parallelGroup) {
    z-index: 0 !important;
    pointer-events: auto;
  }

  :deep(.vue-flow__node-start),
  :deep(.vue-flow__node-pipeline) {
    z-index: 2 !important;
  }

  .start-node,
  .pipeline-node,
  .parallel-group-node {
    position: relative;
    min-height: 76px;
    border-radius: 8px;
    box-shadow: 0 10px 26px rgb(15 23 42 / 8%);
  }

  .parallel-group-node {
    pointer-events: auto;
    cursor: move;
    background: transparent;
    border: 1px solid #a78bfa;
    box-shadow:
      inset 0 0 0 1px rgb(255 255 255 / 70%),
      0 18px 42px rgb(124 58 237 / 10%);
  }

  .parallel-group-title {
    position: absolute;
    top: 10px;
    left: 14px;
    display: inline-flex;
    gap: 8px;
    align-items: center;
    padding: 4px 8px;
    color: #5b21b6;
    background: rgb(255 255 255 / 88%);
    border: 1px solid #ddd6fe;
    border-radius: 6px;
    box-shadow: 0 6px 14px rgb(124 58 237 / 10%);

    strong {
      font-size: 12px;
      line-height: 1;
    }

    span {
      font-size: 12px;
      line-height: 1;
      color: #7c3aed;
    }
  }

  .start-node {
    display: grid;
    grid-template-columns: 44px minmax(0, 1fr);
    column-gap: 10px;
    align-items: center;
    width: 240px;
    padding: 10px;
    color: #064e3b;
    background: #ecfdf5;
    border: 1px solid #6ee7b7;

    &.post {
      color: #7c2d12;
      background: #fff7ed;
      border-color: #fdba74;

      .start-mark {
        color: #c2410c;
        background: #ffedd5;
        border-color: #fed7aa;
      }

      .start-text em {
        color: #c2410c;
      }
    }
  }

  .start-mark {
    display: flex;
    align-items: center;
    justify-content: center;
    height: 36px;
    font-size: 12px;
    font-weight: 700;
    color: #047857;
    background: #d1fae5;
    border: 1px solid #a7f3d0;
    border-radius: 8px;
  }

  .start-text {
    min-width: 0;

    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 14px;
    }

    em {
      margin-top: 5px;
      font-style: normal;
      color: #047857;
    }
  }

  .pipeline-node {
    display: grid;
    grid-template-columns: 36px minmax(104px, 1fr) minmax(58px, auto);
    gap: 8px;
    align-items: center;
    min-width: 230px;
    max-width: 380px;
    padding: 10px 12px;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);

    &::before {
      position: absolute;
      top: 10px;
      bottom: 10px;
      left: 0;
      width: 3px;
      content: '';
      background: transparent;
      border-radius: 0 4px 4px 0;
    }

    &.active {
      border-color: var(--el-color-primary);
      box-shadow:
        0 0 0 2px var(--el-color-primary-light-8),
        0 16px 34px rgb(37 99 235 / 14%);
    }

    &.disabled {
      opacity: 0.55;
    }

    &.parallel {
      background: #faf7ff;
      border-color: #c4b5fd;

      &::before {
        background: #7c3aed;
      }

      .node-icon {
        color: #7c3aed !important;
        background: #f3e8ff;
        border-color: #ddd6fe;
      }
    }

    &.post-step {
      background: #fff7ed;
      border-color: #fdba74;

      &::before {
        background: #f97316;
      }

      .node-icon {
        color: #ea580c !important;
        background: #ffedd5;
        border-color: #fed7aa;
      }
    }

    &.approval {
      background: #fffbeb;
      border-color: #f59e0b;
      box-shadow: 0 16px 34px rgb(245 158 11 / 12%);

      &::before {
        background: #f59e0b;
      }

      .node-icon {
        color: #b45309 !important;
        background: #fef3c7;
        border-color: #fde68a;
      }

      .node-title {
        color: #92400e;
      }
    }

    &.invalid {
      background: #fff7f7;
      border-color: var(--el-color-danger);
      box-shadow:
        0 0 0 2px var(--el-color-danger-light-9),
        0 14px 34px rgb(220 38 38 / 12%);
    }
  }

  .node-icon {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    overflow: hidden;
    font-size: 21px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .node-main {
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 0;
  }

  .node-title {
    display: -webkit-box;
    overflow: hidden;
    font-size: 14px;
    font-weight: 700;
    line-height: 1.35;
    color: var(--el-text-color-primary);
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .node-warning {
    overflow: hidden;
    font-size: 12px;
    line-height: 1.2;
    color: var(--el-color-danger);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .node-code {
    overflow: hidden;
    font-size: 12px;
    line-height: 1.2;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .node-type-tag {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    max-width: 72px;
    height: 22px;
    padding: 0 6px;
    font-size: 12px;
    line-height: 20px;
    white-space: nowrap;
  }

  .node-badges {
    display: inline-flex;
    flex: none;
    flex-wrap: wrap;
    gap: 4px;
    align-items: center;
    justify-content: flex-end;
    max-width: 88px;
  }

  .node-branch-tag {
    color: #6d28d9;
    background: #f5f3ff;
    border-color: #c4b5fd;
  }

  .node-post-tag {
    color: #c2410c;
    background: #ffedd5;
    border-color: #fdba74;
  }

  .node-fanout-tag {
    color: #1d4ed8;
    background: #eff6ff;
    border-color: #bfdbfe;
  }

  .node-head-add,
  .node-tail-add {
    position: absolute;
    top: -11px;
    z-index: 2;
    display: flex;
    align-items: center;
    justify-content: center;
    width: 22px;
    height: 22px;
    font-size: 14px;
    font-weight: 700;
    color: var(--el-color-primary);
    pointer-events: none;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-color-primary-light-5);
    border-radius: 50%;
    box-shadow: 0 8px 18px rgb(37 99 235 / 14%);
    opacity: 0;
    transition:
      color 0.16s ease,
      background 0.16s ease,
      border-color 0.16s ease,
      opacity 0.16s ease,
      transform 0.16s ease;
    transform: translateY(-4px);

    &:hover {
      color: #fff;
      background: var(--el-color-primary);
      border-color: var(--el-color-primary);
    }
  }

  .start-node:hover .node-tail-add,
  .pipeline-node:hover .node-head-add,
  .pipeline-node:hover .node-tail-add,
  .pipeline-node.active .node-head-add,
  .pipeline-node.active .node-tail-add {
    pointer-events: auto;
    opacity: 1;
    transform: translateY(0);
  }

  .node-head-add {
    left: 12px;
  }

  .node-tail-add {
    right: 12px;
  }

  .node-resize-grip {
    position: absolute;
    right: 6px;
    bottom: 5px;
    z-index: 3;
    width: 12px;
    height: 12px;
    cursor: ew-resize;
    background: linear-gradient(135deg, transparent 50%, var(--el-border-color) 50%) bottom right /
      8px 8px no-repeat;
    opacity: 0;
    transition: opacity 0.16s ease;
  }

  .pipeline-node:hover .node-resize-grip,
  .pipeline-node.active .node-resize-grip {
    opacity: 1;
  }

  .workflow-handle {
    width: 10px;
    height: 10px;
    background: #fff;
    border: 2px solid var(--el-color-primary);
    opacity: 0;
    transition:
      opacity 0.16s ease,
      box-shadow 0.16s ease;
  }

  .start-node:hover .workflow-handle,
  .pipeline-node:hover .workflow-handle,
  .pipeline-node.active .workflow-handle {
    opacity: 1;
  }

  .workflow-handle-source {
    right: -6px;
  }

  .workflow-handle-target {
    left: -6px;
  }

  .workflow-handle-post-source {
    bottom: -6px;
    left: 50%;
    pointer-events: none;
    border-color: #f97316;
    transform: translateX(-50%);
  }

  .workflow-handle-post-target {
    top: -6px;
    left: 50%;
    pointer-events: none;
    border-color: #f97316;
    transform: translateX(-50%);
  }

  .workflow-handle:hover {
    box-shadow: 0 0 0 5px var(--el-color-primary-light-8);
  }

  .parallel-group-handle {
    width: 10px;
    height: 10px;
    pointer-events: none;
    background: #fff;
    border: 2px solid #7c3aed;
    opacity: 0;
  }

  .parallel-group-handle-in {
    left: -6px;
  }

  .parallel-group-handle-branch {
    top: 50%;
    left: -6px;
  }

  .parallel-group-handle-exit {
    right: -6px;
  }

  .canvas-empty-tip {
    position: absolute;
    top: 32px;
    left: 50%;
    z-index: 5;
    display: flex;
    flex-direction: column;
    gap: 4px;
    min-width: 260px;
    padding: 10px 12px;
    pointer-events: none;
    background: rgb(255 255 255 / 92%);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 14px 36px rgb(15 23 42 / 10%);
    transform: translateX(-50%);

    span {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .workflow-context-menu {
    position: absolute;
    z-index: 20;
    display: flex;
    flex-direction: column;
    min-width: 144px;
    padding: 6px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 16px 36px rgb(15 23 42 / 16%);

    button {
      height: 32px;
      padding: 0 10px;
      color: var(--el-text-color-regular);
      text-align: left;
      cursor: pointer;
      background: transparent;
      border: 0;
      border-radius: 6px;

      &:hover {
        color: var(--el-color-primary);
        background: var(--el-color-primary-light-9);
      }

      &.danger {
        color: var(--el-color-danger);

        &:hover {
          background: var(--el-color-danger-light-9);
        }
      }
    }
  }

  :deep(.workflow-polish-dialog) {
    --el-dialog-padding-primary: 0;

    display: flex;
    flex-direction: column;
    max-height: min(86vh, 780px);
    overflow: hidden;
    border-radius: 8px;
  }

  :deep(.workflow-polish-dialog .el-dialog__header) {
    flex: none;
    padding: 18px 22px;
    margin-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  :deep(.workflow-polish-dialog .el-dialog__title) {
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--el-text-color-primary);
  }

  :deep(.workflow-polish-dialog .el-dialog__headerbtn) {
    top: 12px;
    right: 14px;
  }

  :deep(.workflow-polish-dialog .el-dialog__body) {
    flex: none;
    min-height: 0;
    padding: 0;
    overflow: hidden;
  }

  :deep(.workflow-polish-dialog .el-dialog__footer) {
    flex: none;
    padding: 12px 22px 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .workflow-dialog-footer {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .step-select-panel {
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
    gap: 16px;
    height: min(560px, calc(86vh - 168px));
    padding: 24px 40px 28px;
    overflow: hidden;
  }

  .step-select-filter {
    display: grid;
    grid-template-columns: minmax(220px, 260px) minmax(320px, 1fr) 108px;
    gap: 12px;
    align-items: center;

    :deep(.el-button) {
      justify-self: end;
      width: 108px;
    }
  }

  .step-choice-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    align-content: start;
    min-height: 0;
    max-height: none;
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
    min-height: 74px;
    padding: 12px 14px;
    text-align: left;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;

    &:hover {
      border-color: var(--el-color-primary);
      box-shadow: 0 8px 24px rgb(37 99 235 / 10%);
    }

    &:disabled,
    &.is-disabled {
      cursor: not-allowed;
      box-shadow: none;
      opacity: 0.55;
    }

    &:disabled:hover,
    &.is-disabled:hover {
      border-color: var(--el-border-color-light);
      box-shadow: none;
    }

    &.approval {
      background: #fffbeb;
      border-color: #fde68a;

      .library-icon {
        color: #b45309 !important;
        background: #fef3c7;
        border-color: #fde68a;
      }
    }

    &.post {
      background: #fff7ed;
      border-color: #fed7aa;

      .library-icon {
        color: #ea580c !important;
        background: #ffedd5;
        border-color: #fed7aa;
      }
    }

    &.is-skeleton {
      cursor: default;
      background: linear-gradient(135deg, #fbfdff 0%, #f7faff 100%);
      border-color: #edf2f8;
      box-shadow: none;

      .library-icon,
      strong,
      em {
        position: relative;
        overflow: hidden;
        color: transparent;
        background: #eef3f8;
        border-color: #eef3f8;
      }

      .library-icon::after,
      strong::after,
      em::after {
        position: absolute;
        inset: 0;
        content: '';
        background: linear-gradient(
          90deg,
          transparent 0%,
          rgb(255 255 255 / 78%) 48%,
          transparent 100%
        );
        transform: translateX(-100%);
        animation: step-skeleton-shimmer 1.25s ease-in-out infinite;
      }

      strong {
        width: 76%;
        height: 16px;
        border-radius: 4px;
      }

      em {
        width: 46%;
        height: 12px;
        border-radius: 4px;
      }
    }
  }

  @keyframes step-skeleton-shimmer {
    100% {
      transform: translateX(100%);
    }
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

    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    em {
      margin-top: 5px;
      font-style: normal;
      color: var(--el-text-color-secondary);
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

  .dynamic-select-control.is-object-list {
    grid-template-columns: minmax(0, 1fr) 28px;
    gap: 6px;
  }

  .dynamic-select-refresh {
    width: 74px;
    height: 34px;
    padding: 0;
  }

  .dynamic-select-control.is-object-list .dynamic-select-refresh {
    width: 28px;
    height: 28px;
  }

  .platform-param-select {
    width: 100%;
    min-width: 0;

    :deep(.el-select__wrapper) {
      position: relative;
      min-height: 42px;
      padding: 10px 12px;
      background: #fff;
      border-radius: 14px;
      box-shadow: 0 0 0 1px #e2e8f0 inset;
      transition:
        box-shadow 0.18s ease,
        background-color 0.18s ease;
    }

    :deep(.el-select__selected-item),
    :deep(.el-select__selected-item span),
    :deep(.el-select__input) {
      min-width: 0;
      font-weight: 500;
      color: #1f2937;
      -webkit-text-fill-color: #1f2937;
    }

    :deep(.el-select__selection) {
      min-width: 0;
    }

    :deep(.el-select__selected-item span) {
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    :deep(.el-select__placeholder) {
      color: #9aa4b2;
      -webkit-text-fill-color: #9aa4b2;
    }

    :deep(.el-select__placeholder.is-transparent) {
      color: #1f2937;
      -webkit-text-fill-color: #1f2937;
      opacity: 1;
    }

    :deep(.el-select.is-disabled .el-select__selected-item),
    :deep(.el-select.is-disabled .el-select__selected-item span),
    :deep(.el-select.is-disabled .el-select__input),
    :deep(.el-select.is-disabled .el-select__placeholder.is-transparent) {
      color: #8b95a5;
      -webkit-text-fill-color: #8b95a5;
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

    max-width: min(640px, calc(100vw - 32px));
    padding: 0;
    background: #ffffff !important;
    border: 1px solid #e2e8f0;
    border-radius: 14px;
    box-shadow: 0 16px 34px rgb(15 23 42 / 12%);
    overflow: hidden !important;

    .el-select-dropdown {
      overflow: hidden !important;
      background: #ffffff !important;
      transform: none !important;
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
      min-height: 44px !important;
      padding: 10px 12px !important;
      overflow: visible !important;
      line-height: 1.35 !important;
      color: #334155 !important;
      text-overflow: clip !important;
      white-space: normal !important;
      background: #ffffff !important;
      border-radius: 0 !important;
      transform: none !important;
      text-align: left !important;
      writing-mode: horizontal-tb !important;
      transition:
        background-color 0.16s ease,
        color 0.16s ease;
    }

    .el-select-dropdown__item.is-selected {
      font-weight: 600 !important;
      color: #0f172a !important;
      background: #ffffff !important;
    }

    .el-select-dropdown__item.hover,
    .el-select-dropdown__item:hover {
      color: #1d4ed8 !important;
      background: #eff6ff !important;
      box-shadow: inset 4px 0 0 #3b82f6;
    }
  }

  :global(.el-popper.devops-platform-param-dropdown[aria-hidden='true']) {
    display: none !important;
  }

  :global(.el-popper.devops-platform-param-dropdown .el-select-dropdown__empty) {
    display: flex;
    align-items: center;
    justify-content: center;
    min-height: var(--devops-platform-dropdown-height, 260px);
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

  @media (width <= 960px) {
    .step-select-panel {
      padding: 20px;
    }

    .step-select-filter,
    .step-choice-grid {
      grid-template-columns: 1fr;
    }

    .step-select-filter :deep(.el-button) {
      justify-self: stretch;
      width: 100%;
    }
  }

  .step-config-form {
    box-sizing: border-box;
    height: min(560px, calc(86vh - 168px));
    padding: 24px 32px 28px;
    overflow: auto;
  }

  .chosen-step-head {
    display: grid;
    grid-template-columns: 40px minmax(0, 1fr) auto;
    gap: 8px;
    align-items: center;
    padding: 10px;
    margin-bottom: 12px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;

    strong,
    span {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    span {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
    }
  }

  .step-param-panel {
    padding: 12px;
    margin-top: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .step-param-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;

    strong {
      font-size: 15px;
      color: var(--el-text-color-primary);
    }

    span {
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .step-param-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 12px;

    :deep(.el-select) {
      width: 100%;
    }
  }

  .step-param-item {
    min-width: 0;
    padding: 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .step-param-item.is-full-row {
    grid-column: 1 / -1;

    .step-param-code {
      margin-bottom: 10px;
    }
  }

  .step-param-item.is-missing {
    background: var(--el-color-danger-light-9);
    border-color: var(--el-color-danger);
  }

  .step-param-item.is-missing :deep(.el-input__wrapper),
  .step-param-item.is-missing :deep(.el-textarea__inner),
  .step-param-item.is-missing :deep(.el-select__wrapper) {
    border-color: var(--el-color-danger);
    box-shadow: 0 0 0 1px var(--el-color-danger) inset;
  }

  .step-param-item-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    min-width: 0;
  }

  .step-param-title {
    display: flex;
    gap: 4px;
    align-items: center;
    min-width: 0;
    font-size: 14px;
    color: var(--el-text-color-primary);

    i {
      flex: 0 0 auto;
      font-style: normal;
      color: var(--el-color-danger);
    }

    strong {
      min-width: 0;
      overflow: hidden;
      font-weight: 600;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .step-param-code {
    margin: 4px 0 10px;
    overflow: hidden;
    font-size: 12px;
    line-height: 1.4;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .step-param-control {
    min-width: 0;

    :deep(.el-input),
    :deep(.el-select),
    :deep(.el-textarea) {
      width: 100%;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper),
    :deep(.el-textarea__inner) {
      background: #fff;
      border-radius: 14px;
      box-shadow: 0 0 0 1px #e2e8f0 inset;
      transition:
        box-shadow 0.18s ease,
        background-color 0.18s ease;
    }

    :deep(.el-input__wrapper),
    :deep(.el-select__wrapper) {
      min-height: 42px;
      padding: 0 12px;
    }

    :deep(.el-textarea__inner) {
      padding: 10px 12px;
    }

    :deep(.el-input__inner),
    :deep(.el-textarea__inner),
    :deep(.el-select__selected-item),
    :deep(.el-select__selected-item span),
    :deep(.el-select__input) {
      font-weight: 500;
      color: #1f2937;
      -webkit-text-fill-color: #1f2937;
    }

    :deep(.el-input__inner::placeholder),
    :deep(.el-textarea__inner::placeholder),
    :deep(.el-select__placeholder) {
      font-weight: 400;
      color: #9aa4b2;
      -webkit-text-fill-color: #9aa4b2;
    }

    :deep(.el-select__placeholder.is-transparent) {
      font-weight: 500;
      color: #1f2937;
      -webkit-text-fill-color: #1f2937;
      opacity: 1;
    }

    :deep(.el-input.is-disabled .el-input__inner),
    :deep(.el-textarea.is-disabled .el-textarea__inner),
    :deep(.el-select.is-disabled .el-select__selected-item),
    :deep(.el-select.is-disabled .el-select__selected-item span),
    :deep(.el-select.is-disabled .el-select__input),
    :deep(.el-select.is-disabled .el-select__placeholder.is-transparent) {
      color: #8b95a5;
      -webkit-text-fill-color: #8b95a5;
    }

    :deep(.el-select__caret) {
      color: #475569;
    }

    :deep(.el-input__wrapper:hover),
    :deep(.el-select__wrapper:hover),
    :deep(.el-textarea__inner:hover) {
      box-shadow: 0 0 0 1px #94a3b8 inset;
    }

    :deep(.el-input__wrapper.is-focus),
    :deep(.el-select__wrapper.is-focused),
    :deep(.el-textarea__inner:focus) {
      box-shadow:
        0 0 0 1px #2563eb inset,
        0 0 0 4px rgb(37 99 235 / 14%);
    }
  }

  .derived-param-preview {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 8px 10px;
    font-size: 12px;
    line-height: 1.4;
    color: var(--el-text-color-secondary);
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .step-param-help {
    margin-top: 6px;
    overflow: hidden;
    font-size: 12px;
    line-height: 1.35;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .object-list-editor {
    display: flex;
    flex-direction: column;
    gap: 8px;
  }

  .object-list-head {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .object-list-table {
    overflow: hidden;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .object-list-table :deep(.el-input),
  .object-list-table :deep(.el-select) {
    width: 100%;
  }
</style>
