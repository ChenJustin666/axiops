<template>
  <div class="devops-tekton-step-page">
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
      :data="tableData"
      :pagination="pagination"
      empty-text="暂无 Tekton 步骤"
      @add="openDialog('add')"
      @refresh="refreshAll"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #empty-actions>
        <ElButton type="primary" @click="openDialog('add')">新增</ElButton>
      </template>
      <template #body-before>
        <div class="tekton-step-overview">
          <div class="overview-metrics">
            <div class="overview-item">
              <span>片段数量</span>
              <strong>{{ pagination.total }}</strong>
            </div>
            <div class="overview-item">
              <span>镜像数量</span>
              <strong>{{ tableImageCount }}</strong>
            </div>
          </div>
          <ElButton type="primary" plain @click="openImageReplaceDialog('global')">
            镜像管理
          </ElButton>
        </div>
      </template>

      <template #default="{ item }">
        <div class="step-card">
          <div class="step-card-head">
            <div class="step-icon" :style="{ color: item.iconColor || '#2563eb' }">
              <ArtSvgIcon :icon="item.icon || 'ri:flow-chart'" />
            </div>
            <div class="step-title">
              <span class="name" :title="item.name">{{ item.name }}</span>
              <span class="code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElSwitch
              :model-value="item.status === 1"
              @change="(val) => handleStatusChange(item, Boolean(val))"
            />
          </div>
          <div class="step-card-body">
            <div class="tag-row">
              <ElTag size="small" type="info">{{ item.categoryName || '未分类' }}</ElTag>
              <ElTag size="small" type="success">Task</ElTag>
              <ElTag size="small">{{ visibleStepParamCount(item) }} 个参数</ElTag>
            </div>
            <p class="desc" :title="item.description || '暂无描述'">
              {{ item.description || '暂无描述' }}
            </p>
            <div class="resource-line" :title="resourceName(item.stageContent)">
              <span>资源名称</span>
              <strong>{{ resourceName(item.stageContent) || '-' }}</strong>
            </div>
            <div v-if="resourceImages(item.stageContent).length > 0" class="card-image-line">
              <span class="card-field-label">镜像</span>
              <div class="image-tag-list">
                <ElTooltip
                  v-for="image in resourceImages(item.stageContent)"
                  :key="image"
                  :content="image"
                  placement="top"
                  :show-after="300"
                >
                  <ElTag
                    class="image-tag"
                    size="small"
                    type="info"
                    @click.stop="openImageReplaceDialog('row', item)"
                  >
                    <span class="image-tag-text">{{ image }}</span>
                  </ElTag>
                </ElTooltip>
              </div>
            </div>
          </div>
          <div class="step-card-footer">
            <span class="created-time" :title="`创建时间：${smartFormatTime(item.createdAt)}`">
              创建：{{ smartFormatTime(item.createdAt) }}
            </span>
            <span class="footer-actions">
              <ElButton link type="primary" @click="openDialog('edit', item)">配置</ElButton>
              <ElButton link type="danger" @click="handleDelete(item)">删除</ElButton>
            </span>
          </div>
        </div>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增 Tekton 步骤' : '配置 Tekton 步骤'"
      width="1180px"
      align-center
      class="devops-polish-dialog step-dialog tekton-step-dialog"
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <div class="wizard-steps">
        <button
          v-for="(item, index) in wizardSteps"
          :key="item.title"
          type="button"
          class="wizard-step"
          :class="{ active: dialogStep === index, done: dialogStep > index }"
          @click="handleStepClick(index)"
        >
          <span class="wizard-index">{{ index + 1 }}</span>
          <span class="wizard-text">
            <strong>{{ item.title }}</strong>
            <em>{{ item.desc }}</em>
          </span>
        </button>
      </div>

      <ElForm
        ref="formRef"
        v-loading="dialogLoading"
        element-loading-text="加载中..."
        :model="formData"
        :rules="rules"
        label-width="118px"
        :disabled="submitLoading || dialogLoading"
      >
        <div v-show="dialogStep === 0" class="dialog-step-panel">
          <section class="config-panel scroll-panel">
            <div class="panel-head">
              <div>
                <strong>公共配置</strong>
                <span>片段基础信息和 Task 资源标识</span>
              </div>
            </div>
            <ElRow :gutter="16">
              <ElCol :span="12">
                <ElFormItem label="名称" prop="name">
                  <ElInput v-model="formData.name" placeholder="例如 Git Clone" />
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="编码" prop="code">
                  <ElInput
                    v-model="formData.code"
                    :disabled="dialogType === 'edit'"
                    placeholder="例如 git-clone"
                  />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElRow :gutter="16">
              <ElCol :span="12">
                <ElFormItem label="步骤分类" prop="categoryId">
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-model="formData.categoryId"
                    filterable
                    placeholder="请选择分类"
                    style="width: 100%"
                    @visible-change="(visible: boolean) => visible && fetchCategories()"
                  >
                    <ElOption
                      v-for="item in categories"
                      :key="item.id"
                      :label="`${item.name}（${item.code}）`"
                      :value="item.id"
                    />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="引擎类型">
                  <ElInput model-value="Tekton" disabled />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElRow :gutter="16">
              <ElCol :span="12">
                <ElFormItem label="Task 名称">
                  <ElInput
                    :model-value="taskResourceName()"
                    disabled
                    placeholder="按步骤编码自动生成"
                  />
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="API 版本">
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-model="taskForm.apiVersion"
                    style="width: 100%"
                  >
                    <ElOption label="tekton.dev/v1" value="tekton.dev/v1" />
                    <ElOption label="tekton.dev/v1beta1" value="tekton.dev/v1beta1" />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElFormItem label="步骤图标">
              <div class="icon-select-panel">
                <ElSegmented
                  v-model="activeIconGroup"
                  class="icon-group-tabs"
                  :options="iconGroupOptions"
                />
                <div class="icon-picker">
                  <button
                    v-for="item in currentIconGroup?.items || []"
                    :key="item.icon"
                    type="button"
                    :class="{ active: formData.icon === item.icon }"
                    :style="{ color: item.color }"
                    :title="item.label"
                    @click="pickIcon(item.icon, item.color)"
                  >
                    <ArtSvgIcon :icon="item.icon" />
                    <span>{{ item.label }}</span>
                  </button>
                </div>
              </div>
            </ElFormItem>
            <ElFormItem label="步骤描述">
              <ElInput
                v-model="formData.description"
                type="textarea"
                :rows="3"
                placeholder="请输入步骤描述"
              />
            </ElFormItem>
            <ElFormItem label="状态" prop="status">
              <ElRadioGroup v-model="formData.status">
                <ElRadio :value="1">启用</ElRadio>
                <ElRadio :value="0">停用</ElRadio>
              </ElRadioGroup>
            </ElFormItem>
          </section>
        </div>

        <div v-show="dialogStep === 1" class="dialog-step-panel">
          <section class="config-panel yaml-panel">
            <div class="panel-head yaml-panel-head">
              <div>
                <strong>YAML 片段</strong>
                <span>
                  {{ yamlParseError || `Task：${taskForm.name || '-'}，name 按编码保存` }}
                </span>
                <div class="yaml-field-hints">
                  <ElTooltip
                    content="stepTemplate 是 Task 内所有 steps 的默认容器配置，例如 env、volumeMounts、securityContext。"
                    placement="top"
                  >
                    <ElTag size="small" type="info">stepTemplate</ElTag>
                  </ElTooltip>
                  <ElTooltip
                    content="sidecars 是跟 Task steps 并行运行的辅助容器，常用于代理、缓存或本地服务。"
                    placement="top"
                  >
                    <ElTag size="small" type="info">sidecars</ElTag>
                  </ElTooltip>
                </div>
              </div>
              <div class="panel-actions">
                <ElButton text type="primary" size="small" @click="syncGeneratedYaml(true)">
                  从公共配置同步
                </ElButton>
                <ElButton text type="primary" size="small" @click="formatAdvancedYaml(true)">
                  整理 YAML
                </ElButton>
                <ElButton text type="primary" size="small" @click="handleValidateYamlClick">
                  校验 YAML
                </ElButton>
              </div>
            </div>
            <YamlEditorPro
              v-model="advancedYaml"
              height="100%"
              filename="task.yaml"
              theme="dark"
              :word-wrap="true"
              :show-toolbar="true"
            />
          </section>
        </div>

        <div v-show="dialogStep === 2" class="dialog-step-panel">
          <section class="config-panel scroll-panel">
            <div class="panel-head">
              <div>
                <strong>Task 参数契约</strong>
                <span>只维护 Task spec.params，支持 string、array、object</span>
              </div>
              <div class="panel-actions">
                <ElButton text type="primary" @click="syncParamsFromTask(true)"
                  >从 YAML 参数导入</ElButton
                >
                <ElButton text type="primary" @click="addParam">新增</ElButton>
              </div>
            </div>
            <div class="param-table-wrap">
              <ElEmpty v-if="visibleParams.length === 0" description="暂无 Task 参数">
                <ElButton type="primary" plain @click="syncParamsFromTask(true)">
                  从 YAML 参数导入
                </ElButton>
              </ElEmpty>
              <ElTable v-else class="param-table" :data="visibleParams" height="100%" border>
                <ElTableColumn type="index" label="#" width="56" align="center" />
                <ElTableColumn label="参数名称" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.name || '未配置名称' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="Task 参数名" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.code || '未配置名称' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="Tekton 类型" width="132">
                  <template #default="{ row }">
                    <ElTag size="small" effect="plain">{{ paramTypeName(row.paramType) }}</ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="默认值" min-width="150" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.defaultValue || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="限制" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ paramConstraintSummary(row) }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="必填" width="76" align="center">
                  <template #default="{ row }">
                    <ElTag size="small" :type="row.required ? 'danger' : 'info'" effect="plain">
                      {{ row.required ? '是' : '否' }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="排序" prop="sortOrder" width="76" align="center" />
                <ElTableColumn label="描述" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.description || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="操作" width="150" fixed="right" align="right">
                  <template #default="{ row }">
                    <div class="table-row-actions">
                      <ElButton text type="primary" @click="editParam(row.__index)">编辑</ElButton>
                      <ElButton text type="danger" @click="removeParam(row.__index)">删除</ElButton>
                    </div>
                  </template>
                </ElTableColumn>
              </ElTable>
            </div>
          </section>
        </div>

        <div v-show="dialogStep === 3" class="dialog-step-panel">
          <section class="config-panel scroll-panel">
            <div class="panel-head">
              <div>
                <strong>Task Result 契约</strong>
                <span>维护 Task spec.results 的名称、类型、值表达式和对象属性</span>
              </div>
              <div class="panel-actions">
                <ElButton text type="primary" @click="syncResultsFromTask(true)">
                  从 YAML Result 导入
                </ElButton>
                <ElButton text type="primary" @click="addResult">新增</ElButton>
              </div>
            </div>
            <div class="result-table-wrap">
              <ElEmpty v-if="visibleResults.length === 0" description="暂无 Task Result">
                <ElButton type="primary" plain @click="syncResultsFromTask(true)">
                  从 YAML Result 导入
                </ElButton>
              </ElEmpty>
              <ElTable v-else class="result-table" :data="visibleResults" height="100%" border>
                <ElTableColumn type="index" label="#" width="56" align="center" />
                <ElTableColumn label="Result 名称" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.name || '未配置名称' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="Tekton 类型" width="132">
                  <template #default="{ row }">
                    <ElTag size="small" effect="plain">{{ row.type || 'string' }}</ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="值表达式" min-width="220" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.value || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="限制" min-width="150" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ taskResultConstraintSummary(row) }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="描述" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.description || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="操作" width="150" fixed="right" align="right">
                  <template #default="{ row }">
                    <div class="table-row-actions">
                      <ElButton text type="primary" @click="editResult(row.__index)">编辑</ElButton>
                      <ElButton text type="danger" @click="removeResult(row.__index)">
                        删除
                      </ElButton>
                    </div>
                  </template>
                </ElTableColumn>
              </ElTable>
            </div>
          </section>
        </div>

        <div v-show="dialogStep === 4" class="dialog-step-panel">
          <section class="config-panel scroll-panel">
            <div class="panel-head">
              <div>
                <strong>Workspace 管理</strong>
                <span>只维护 Task spec.workspaces 声明</span>
              </div>
              <div class="panel-actions">
                <ElButton text type="primary" @click="syncWorkspacesFromTask(true)">
                  从 YAML 工作区导入
                </ElButton>
                <ElButton text type="primary" @click="addWorkspace">新增</ElButton>
              </div>
            </div>
            <div class="workspace-table-wrap">
              <ElEmpty v-if="taskForm.workspaces.length === 0" description="暂无 Workspace">
                <ElButton type="primary" plain @click="syncWorkspacesFromTask(true)">
                  从 YAML 工作区导入
                </ElButton>
              </ElEmpty>
              <ElTable
                v-else
                class="workspace-table"
                :data="taskForm.workspaces"
                height="100%"
                border
              >
                <ElTableColumn type="index" label="#" width="56" align="center" />
                <ElTableColumn label="Workspace 名称" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.name || '未配置名称' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="必填" width="86" align="center">
                  <template #default="{ row }">
                    <ElTag size="small" :type="row.optional ? 'info' : 'danger'" effect="plain">
                      {{ row.optional ? '否' : '是' }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="只读" width="86" align="center">
                  <template #default="{ row }">
                    <ElTag size="small" :type="row.readOnly ? 'warning' : 'info'" effect="plain">
                      {{ row.readOnly ? '是' : '否' }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="挂载路径" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.mountPath || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="描述" min-width="220" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.description || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="操作" width="150" fixed="right" align="right">
                  <template #default="{ $index }">
                    <div class="table-row-actions">
                      <ElButton text type="primary" @click="editWorkspace($index)">编辑</ElButton>
                      <ElButton text type="danger" @click="removeWorkspace($index)">删除</ElButton>
                    </div>
                  </template>
                </ElTableColumn>
              </ElTable>
            </div>
          </section>
        </div>
      </ElForm>

      <template #footer>
        <ElButton :disabled="dialogLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton v-if="dialogStep > 0" @click="() => handlePrevStep()">上一步</ElButton>
        <ElButton
          v-if="dialogStep < wizardSteps.length - 1"
          type="primary"
          :disabled="dialogLoading"
          @click="() => handleNextStep()"
        >
          下一步
        </ElButton>
        <ElButton
          v-else
          type="primary"
          :loading="submitLoading"
          :disabled="dialogLoading"
          @click="handleSubmit"
        >
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="workspaceDialogVisible"
      :title="workspaceDialogIndex === -1 ? '新增 Workspace' : '编辑 Workspace'"
      width="min(760px, calc(100vw - 32px))"
      top="5vh"
      append-to-body
      class="devops-polish-dialog workspace-edit-dialog"
      :close-on-click-modal="false"
    >
      <ElForm
        ref="workspaceFormRef"
        :model="workspaceForm"
        :rules="workspaceRules"
        label-width="112px"
        class="workspace-edit-form"
      >
        <ElFormItem prop="name" label="Workspace 名称">
          <ElInput v-model="workspaceForm.name" placeholder="例如 output、basic-auth" />
        </ElFormItem>
        <ElFormItem>
          <template #label>
            <span class="form-label-with-help">
              是否必填
              <ElTooltip
                content="写入官方 spec.workspaces[].optional；关闭后表示可选 Workspace。"
                placement="top"
              >
                <span class="field-help">?</span>
              </ElTooltip>
            </span>
          </template>
          <ElSwitch v-model="workspaceRequired" />
          <div class="workspace-form-tip">
            关闭后写入 optional: true；最终绑定由 Pipeline 和运行配置决定。
          </div>
        </ElFormItem>
        <ElFormItem label="只读挂载">
          <ElSwitch v-model="workspaceForm.readOnly" />
        </ElFormItem>
        <ElFormItem label="挂载路径">
          <ElInput v-model="workspaceForm.mountPath" placeholder="不填写则使用 Tekton 默认路径" />
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput
            v-model="workspaceForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入 Workspace 描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="workspaceDialogVisible = false">取消</ElButton>
          <ElButton type="primary" @click="saveWorkspaceDialog">确定</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="paramDialogVisible"
      :title="paramDialogIndex === -1 ? '新增参数' : '编辑参数'"
      width="min(760px, calc(100vw - 32px))"
      append-to-body
      class="devops-polish-dialog param-edit-dialog"
      :close-on-click-modal="false"
    >
      <ElForm
        ref="paramFormRef"
        :model="paramForm"
        :rules="paramRules"
        label-width="112px"
        class="param-edit-form"
      >
        <div class="param-edit-grid">
          <ElFormItem prop="name" label="参数名称">
            <ElInput v-model="paramForm.name" placeholder="例如 Git 仓库地址" />
          </ElFormItem>
          <ElFormItem prop="code" label="Task 参数名">
            <ElInput v-model="paramForm.code" placeholder="例如 url" />
          </ElFormItem>
          <ElFormItem>
            <template #label>
              <span class="form-label-with-help">
                Tekton 类型
                <ElTooltip
                  content="官方 Task 参数类型只支持 string、array、object。"
                  placement="top"
                >
                  <span class="field-help">?</span>
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              popper-class="devops-step-template-dropdown"
              v-model="paramForm.paramType"
              filterable
              style="width: 100%"
              @change="() => handlePlatformParamTypeChange(paramForm)"
            >
              <ElOption
                v-for="item in paramTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem>
            <template #label>
              <span class="form-label-with-help">
                是否必填
                <ElTooltip
                  content="平台契约字段，用于模板和运行配置校验，不写入 Task YAML。"
                  placement="top"
                >
                  <span class="field-help">?</span>
                </ElTooltip>
              </span>
            </template>
            <ElSwitch v-model="paramForm.required" @change="handleParamRequiredChange" />
          </ElFormItem>
          <ElFormItem label="默认值">
            <ElInput
              v-if="paramForm.paramType === 'object'"
              v-model="paramForm.defaultValue"
              type="textarea"
              :rows="5"
              resize="vertical"
              placeholder="请输入 YAML 对象"
            />
            <ElInput
              v-else-if="paramForm.paramType === 'array'"
              v-model="paramForm.defaultValue"
              type="textarea"
              :rows="4"
              resize="vertical"
              placeholder="请输入 YAML 数组或逗号分隔列表"
            />
            <ElInput v-else v-model="paramForm.defaultValue" placeholder="请输入默认值" />
          </ElFormItem>
          <ElFormItem label="字段说明" class="param-contract-tip">
            <div>是否必填为平台契约字段；默认值、enum、properties 会同步到 Task YAML。</div>
          </ElFormItem>
          <ElFormItem label="排序">
            <ElInputNumber v-model="paramForm.sortOrder" :min="0" style="width: 100%" />
          </ElFormItem>
          <ElFormItem label="枚举值">
            <ElInput
              v-model="paramEnumText"
              type="textarea"
              :rows="4"
              resize="vertical"
              placeholder="每行一个 enum 值，留空则不限制"
            />
          </ElFormItem>
          <ElFormItem v-if="paramForm.paramType === 'object'">
            <template #label>
              <span class="form-label-with-help">
                对象属性
                <ElTooltip
                  content="properties 用于声明 object 参数的子字段，每个子字段需要配置 type。"
                  placement="top"
                >
                  <span class="field-help">?</span>
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="paramPropertiesRaw"
              type="textarea"
              :rows="5"
              resize="vertical"
              placeholder="例如：&#10;image:&#10;  type: string&#10;tags:&#10;  type: array"
            />
          </ElFormItem>
        </div>

        <ElFormItem label="描述" class="param-description">
          <ElInput
            v-model="paramForm.description"
            type="textarea"
            :rows="3"
            placeholder="请输入参数描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="paramDialogVisible = false">取消</ElButton>
          <ElButton type="primary" @click="saveParamDialog">确定</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="resultDialogVisible"
      :title="resultDialogIndex === -1 ? '新增 Result' : '编辑 Result'"
      width="min(720px, calc(100vw - 32px))"
      append-to-body
      class="devops-polish-dialog param-edit-dialog"
      :close-on-click-modal="false"
    >
      <ElForm
        ref="resultFormRef"
        :model="resultForm"
        :rules="resultRules"
        label-width="112px"
        class="param-edit-form"
      >
        <div class="param-edit-grid">
          <ElFormItem prop="name" label="Result 名称">
            <ElInput v-model="resultForm.name" placeholder="例如 commit" />
          </ElFormItem>
          <ElFormItem>
            <template #label>
              <span class="form-label-with-help">
                Tekton 类型
                <ElTooltip
                  content="官方 Task Result 类型只支持 string、array、object。"
                  placement="top"
                >
                  <span class="field-help">?</span>
                </ElTooltip>
              </span>
            </template>
            <ElSelect
              popper-class="devops-step-template-dropdown"
              v-model="resultForm.type"
              style="width: 100%"
            >
              <ElOption label="string" value="string" />
              <ElOption label="array" value="array" />
              <ElOption label="object" value="object" />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="值表达式" class="param-edit-full">
            <ElInput
              v-model="resultForm.value"
              type="textarea"
              :rows="3"
              resize="vertical"
              placeholder="例如 $(steps.clone.results.exitCode)"
            />
          </ElFormItem>
          <ElFormItem v-if="resultForm.type === 'object'" class="param-edit-full">
            <template #label>
              <span class="form-label-with-help">
                对象属性
                <ElTooltip
                  content="properties 用于声明 object Result 的子字段，每个子字段需要配置 type。"
                  placement="top"
                >
                  <span class="field-help">?</span>
                </ElTooltip>
              </span>
            </template>
            <ElInput
              v-model="resultPropertiesRaw"
              type="textarea"
              :rows="5"
              resize="vertical"
              placeholder="例如：&#10;digest:&#10;  type: string"
            />
          </ElFormItem>
          <ElFormItem label="描述" class="param-edit-full">
            <ElInput
              v-model="resultForm.description"
              type="textarea"
              :rows="3"
              resize="vertical"
              placeholder="请输入 Result 描述"
            />
          </ElFormItem>
        </div>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="resultDialogVisible = false">取消</ElButton>
          <ElButton type="primary" @click="saveResultDialog">确定</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="imageDialogVisible"
      :title="imageDialogTitle"
      width="min(1080px, calc(100vw - 32px))"
      align-center
      class="image-replace-dialog"
      :close-on-click-modal="false"
    >
      <div class="image-dialog-layout">
        <div class="image-dialog-head">
          <div class="image-dialog-stats">
            <div class="image-stat">
              <span>镜像数量</span>
              <strong>{{ imagePagination.total }}</strong>
            </div>
            <div class="image-stat">
              <span>已修改</span>
              <strong>{{ imageChangedRows.length }}</strong>
            </div>
          </div>
          <ElButton :loading="imageListLoading" @click="fetchImageRows">刷新</ElButton>
        </div>
        <div class="image-manage-table-wrap" v-loading="imageListLoading">
          <ElTable
            class="image-manage-table"
            :data="imageRows"
            row-key="rowKey"
            height="100%"
            empty-text="暂无镜像"
          >
            <ElTableColumn label="ID" width="156" show-overflow-tooltip>
              <template #default="{ row }">
                <span class="image-row-id" :title="row.id">{{ shortImageId(row.id) }}</span>
              </template>
            </ElTableColumn>
            <ElTableColumn label="名称" min-width="210" show-overflow-tooltip>
              <template #default="{ row }">
                <div class="image-name-cell">
                  <span class="image-row-name" :title="row.name">{{ row.name || '-' }}</span>
                  <span v-if="row.code" class="image-row-code" :title="row.code">
                    {{ row.code }}
                  </span>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn label="镜像地址" min-width="480">
              <template #default="{ row }">
                <div class="image-editor-cell">
                  <ElInput
                    v-model="row.newImage"
                    class="image-input"
                    clearable
                    placeholder="请输入镜像地址"
                  />
                  <ElTag v-if="isImageRowChanged(row)" size="small" type="warning" effect="plain">
                    已修改
                  </ElTag>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </div>
        <div class="image-dialog-pagination">
          <ElPagination
            v-model:current-page="imagePagination.current"
            v-model:page-size="imagePagination.size"
            background
            small
            :page-sizes="[20, 50, 100, 200]"
            layout="total, sizes, prev, pager, next"
            :total="imagePagination.total"
            @size-change="handleImageSizeChange"
            @current-change="handleImageCurrentChange"
          />
        </div>
      </div>
      <template #footer>
        <div class="image-dialog-footer">
          <span>已修改 {{ imageChangedRows.length }} 条</span>
          <div class="dialog-footer-actions">
            <ElButton @click="imageDialogVisible = false">取消</ElButton>
            <ElButton type="primary" :loading="imageReplaceLoading" @click="handleImageReplace">
              保存修改
            </ElButton>
          </div>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import {
    computed,
    defineAsyncComponent,
    nextTick,
    onActivated,
    onMounted,
    reactive,
    ref,
    watch
  } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { parse, stringify } from 'yaml'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    batchUpdateDevopsTektonStepImageApi,
    createDevopsTektonStepApi,
    deleteDevopsTektonStepApi,
    listDevopsStepCategoryApi,
    listDevopsTektonStepApi,
    listDevopsTektonStepImageApi,
    updateDevopsTektonStepApi,
    updateDevopsTektonStepStatusApi,
    type DevopsStepCategory,
    type DevopsStepParam,
    type DevopsStepTemplate,
    type DevopsStepTemplateForm,
    type TektonStepImageItem,
    type UpdateTektonStepImageItem
  } from '@/api/devops/pipeline'
  import { smartFormatTime } from '@/utils/format'
  import {
    buildTaskResource,
    cloneValue,
    collectTaskImagesFromYaml,
    dumpTaskYaml,
    normalizeTektonName,
    normalizeTaskYamlForSave,
    parseTaskYaml as parseTektonTaskYaml,
    validateTaskForSave,
    type TaskParseResult,
    type TektonTaskParam,
    type TektonTaskForm,
    type TektonTaskResult,
    type TektonWorkspace
  } from './utils/taskYaml'

  defineOptions({ name: 'DevopsTektonStep' })

  const YamlEditorPro = defineAsyncComponent(() => import('@/components/yaml-editor-pro/index.vue'))

  type ImageDialogMode = 'global' | 'row'
  type TektonStepImageRow = TektonStepImageItem & {
    rowKey: string
    newImage: string
  }

  const loading = ref(false)
  const submitLoading = ref(false)
  const dialogLoading = ref(false)
  const showSearchBar = ref(true)
  const formRef = ref<FormInstance>()
  const paramFormRef = ref<FormInstance>()
  const resultFormRef = ref<FormInstance>()
  const workspaceFormRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const dialogStep = ref(0)
  const paramDialogVisible = ref(false)
  const paramDialogIndex = ref(-1)
  const paramEnumText = ref('')
  const paramPropertiesRaw = ref('')
  const resultDialogVisible = ref(false)
  const resultDialogIndex = ref(-1)
  const resultPropertiesRaw = ref('')
  const workspaceDialogVisible = ref(false)
  const workspaceDialogIndex = ref(-1)
  const currentId = ref('')
  const tableData = ref<DevopsStepTemplate[]>([])
  const categories = ref<DevopsStepCategory[]>([])
  const searchParams = ref<Record<string, any>>({})
  const advancedYaml = ref('')
  const yamlParseError = ref('')
  const rawTaskResource = ref<Record<string, any>>({})
  const lastAutoTaskName = ref('git-clone')
  const imageDialogVisible = ref(false)
  const imageDialogMode = ref<ImageDialogMode>('global')
  const imageListLoading = ref(false)
  const imageReplaceLoading = ref(false)
  const imageRows = ref<TektonStepImageRow[]>([])
  const currentImageStep = ref<DevopsStepTemplate | null>(null)
  let applyingFormToYaml = false

  const searchForm = ref({
    name: undefined,
    code: undefined,
    categoryId: undefined,
    status: undefined
  })
  const pagination = ref({ current: 1, size: 15, total: 0 })
  const imagePagination = ref({ current: 1, size: 50, total: 0 })

  const activeIconGroup = ref('工具平台')
  const iconGroups = [
    {
      name: '语言类型',
      items: [
        { label: 'Java', icon: 'skill-icons:java-light', color: '#b07219' },
        { label: 'Go', icon: 'skill-icons:golang', color: '#00add8' },
        { label: 'Python', icon: 'skill-icons:python-light', color: '#3776ab' },
        { label: 'Node.js', icon: 'skill-icons:nodejs-light', color: '#339933' },
        { label: 'Vue', icon: 'skill-icons:vuejs-light', color: '#42b883' },
        { label: 'React', icon: 'skill-icons:react-light', color: '#61dafb' },
        { label: 'Angular', icon: 'skill-icons:angular-light', color: '#dd0031' },
        { label: 'PHP', icon: 'skill-icons:php-light', color: '#777bb4' },
        { label: 'Rust', icon: 'skill-icons:rust', color: '#000000' },
        { label: 'JavaScript', icon: 'skill-icons:javascript', color: '#f7df1e' },
        { label: 'TypeScript', icon: 'skill-icons:typescript', color: '#3178c6' },
        { label: 'Shell', icon: 'skill-icons:bash-light', color: '#4eaa25' }
      ]
    },
    {
      name: '工具平台',
      items: [
        { label: 'Tekton', icon: 'simple-icons:tekton', color: '#2563eb' },
        { label: 'Kubernetes', icon: 'simple-icons:kubernetes', color: '#326ce5' },
        { label: 'Docker', icon: 'simple-icons:docker', color: '#0ea5e9' },
        { label: 'Harbor', icon: 'simple-icons:harbor', color: '#60b932' },
        { label: 'Git', icon: 'simple-icons:git', color: '#f05032' },
        { label: 'GitLab', icon: 'simple-icons:gitlab', color: '#fc6d26' },
        { label: 'GitHub', icon: 'simple-icons:github', color: '#24292f' },
        { label: 'Nexus', icon: 'simple-icons:sonatype', color: '#1b75bb' },
        { label: 'JFrog', icon: 'simple-icons:jfrog', color: '#41bf47' },
        { label: 'SonarQube', icon: 'simple-icons:sonarqube', color: '#4b9fd5' },
        { label: 'Jenkins', icon: 'simple-icons:jenkins', color: '#d24939' },
        { label: 'Argo CD', icon: 'simple-icons:argo', color: '#ef7b4d' },
        { label: 'Prometheus', icon: 'simple-icons:prometheus', color: '#e6522c' },
        { label: 'Grafana', icon: 'simple-icons:grafana', color: '#f46800' },
        { label: 'Terraform', icon: 'simple-icons:terraform', color: '#844fba' },
        { label: 'Ansible', icon: 'simple-icons:ansible', color: '#ee0000' }
      ]
    },
    {
      name: '构建工具',
      items: [
        { label: 'Maven', icon: 'simple-icons:apachemaven', color: '#c71a36' },
        { label: 'Gradle', icon: 'simple-icons:gradle', color: '#02303a' },
        { label: 'npm', icon: 'simple-icons:npm', color: '#cb3837' },
        { label: 'pnpm', icon: 'simple-icons:pnpm', color: '#f69220' },
        { label: 'Yarn', icon: 'simple-icons:yarn', color: '#2c8ebb' },
        { label: 'Bun', icon: 'simple-icons:bun', color: '#fbf0df' },
        { label: 'Cargo', icon: 'simple-icons:rust', color: '#000000' },
        { label: 'Helm', icon: 'simple-icons:helm', color: '#0f1689' },
        { label: 'Make', icon: 'ri:hammer-line', color: '#475569' },
        { label: 'Kaniko', icon: 'ri:box-3-line', color: '#2563eb' },
        { label: 'Buildpacks', icon: 'ri:package-line', color: '#0f766e' },
        { label: '通用工具', icon: 'ri:tools-line', color: '#64748b' }
      ]
    }
  ]
  const wizardSteps = [
    { title: '基本配置', desc: '名称、编码、分类、图标' },
    { title: 'Task YAML', desc: '维护 Tekton Task' },
    { title: '参数管理', desc: 'Task 参数契约' },
    { title: 'Result 契约', desc: 'Task 输出契约' },
    { title: 'Workspace', desc: 'Task 工作区声明' }
  ]
  const iconGroupOptions = computed(() =>
    iconGroups.map((group) => ({ label: group.name, value: group.name }))
  )
  const currentIconGroup = computed(
    () => iconGroups.find((group) => group.name === activeIconGroup.value) || iconGroups[0]
  )
  const getIconGroupName = (icon?: string) =>
    iconGroups.find((group) => group.items.some((item) => item.icon === icon))?.name || '工具平台'

  const paramTypeOptions = [
    { label: 'string', value: 'string' },
    { label: 'array', value: 'array' },
    { label: 'object', value: 'object' }
  ]
  const tektonTaskParamTypeSet = new Set(paramTypeOptions.map((item) => item.value))

  const defaultTaskForm = (): TektonTaskForm => ({
    apiVersion: 'tekton.dev/v1',
    name: 'git-clone',
    displayName: 'git clone',
    description: '克隆 Git 仓库到工作空间',
    metadataLabelsRaw: '',
    metadataAnnotationsRaw: '',
    stepTemplateRaw: '',
    sidecarsRaw: '',
    volumesRaw: '',
    params: [
      { name: 'url', type: 'string', description: 'Git 仓库地址' },
      { name: 'revision', type: 'string', default: 'main', description: 'Git 分支或提交' }
    ],
    workspaces: [{ name: 'output', description: '源码目录' }],
    results: [
      { name: 'commit', description: '当前提交 SHA' },
      { name: 'url', description: '当前仓库地址' }
    ],
    steps: [
      {
        name: 'clone',
        image: 'alpine/git:latest',
        script:
          'git clone --branch "$(params.revision)" "$(params.url)" "$(workspaces.output.path)"'
      }
    ]
  })

  const defaultParam = (): DevopsStepParam => ({
    name: '',
    code: '',
    defaultValue: '',
    paramType: 'string',
    mode: 'params',
    required: true,
    readonly: false,
    description: '',
    selectList: [],
    allowCustom: false,
    sortOrder: 0,
    runtimeMode: 'params',
    runtimeConfig: false,
    config: {
      options: [],
      channelGroupCode: '',
      channelParamCode: '',
      channelBindingId: '',
      mappingField: '',
      voucherModel: '',
      credentialId: '',
      credentialMode: 'field',
      credentialSourceParamCode: '',
      configTypeId: '',
      configTypeCode: '',
      valueMode: 'string',
      projectParamCode: '',
      componentParamCode: '',
      provider: '',
      validateRemote: false,
      fullRow: false,
      renderMode: 'normal',
      compoundFields: []
    }
  })

  const defaultWorkspace = (): TektonWorkspace => ({
    name: '',
    description: '',
    optional: false,
    readOnly: false,
    mountPath: ''
  })

  const defaultResult = (): TektonTaskResult => ({
    name: '',
    type: 'string',
    properties: {},
    description: '',
    value: ''
  })

  const defaultForm = (): DevopsStepTemplateForm => ({
    name: '',
    code: '',
    icon: 'simple-icons:tekton',
    iconColor: '#2563eb',
    description: '',
    type: 'task',
    categoryId: '',
    stageContent: '',
    params: [],
    taskParams: [],
    taskResults: [],
    taskWorkspaces: [],
    status: 1
  })

  const formData = reactive<DevopsStepTemplateForm>(defaultForm())
  const paramForm = reactive<DevopsStepParam>(defaultParam())
  const resultForm = reactive<TektonTaskResult>(defaultResult())
  const workspaceForm = reactive<TektonWorkspace>(defaultWorkspace())
  const taskForm = reactive<TektonTaskForm>(defaultTaskForm())
  const workspaceRequired = computed({
    get: () => workspaceForm.optional !== true,
    set: (value: boolean) => {
      workspaceForm.optional = !value
    }
  })

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    categoryId: [{ required: true, message: '请选择分类', trigger: 'change' }],
    stageContent: [{ required: true, message: '请输入 Task YAML', trigger: 'blur' }]
  }
  const paramRules: FormRules = {
    name: [{ required: true, message: '请输入参数名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入 Task 参数名', trigger: 'blur' }]
  }
  const workspaceRules: FormRules = {
    name: [{ required: true, message: '请输入 Workspace 名称', trigger: 'blur' }]
  }
  const resultRules: FormRules = {
    name: [{ required: true, message: '请输入 Result 名称', trigger: 'blur' }]
  }

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '分类',
      key: 'categoryId',
      type: 'select',
      props: {
        clearable: true,
        filterable: true,
        options: categories.value.map((item) => ({ label: item.name, value: item.id }))
      }
    },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: {
        clearable: true,
        options: [
          { label: '启用', value: 1 },
          { label: '停用', value: 0 }
        ]
      }
    }
  ])

  const tableImageCount = computed(() => {
    const images = new Set<string>()
    tableData.value.forEach((item) => {
      collectTaskImagesFromYaml(item.stageContent || '').forEach((image) => images.add(image))
    })
    return images.size
  })

  const isImageRowChanged = (item: TektonStepImageRow) => {
    const oldImage = String(item.image || '').trim()
    const newImage = String(item.newImage || '').trim()
    return Boolean(newImage && newImage !== oldImage)
  }

  const shortImageId = (value?: string) => {
    if (!value) return '-'
    return value.length > 18 ? `${value.slice(0, 8)}...${value.slice(-6)}` : value
  }

  const imageChangedRows = computed(() => imageRows.value.filter(isImageRowChanged))

  const imageDialogTitle = computed(() => {
    if (imageDialogMode.value === 'row') return '片段镜像管理'
    return '镜像管理'
  })

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const fetchCategories = async () => {
    const res = await listDevopsStepCategoryApi({ page: 1, pageSize: 200, status: 1 })
    categories.value = res.items || []
  }

  const fetchList = async () => {
    loading.value = true
    try {
      const res = await listDevopsTektonStepApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        engineType: 'tekton',
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    await fetchCategories()
    await fetchList()
  }

  const handleSearch = () => {
    searchParams.value = pickParams()
    pagination.value.current = 1
    fetchList()
  }

  const handleReset = () => {
    searchParams.value = {}
    pagination.value.current = 1
    fetchList()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    fetchList()
  }

  const handleCurrentChange = (page: number) => {
    pagination.value.current = page
    fetchList()
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsStepTemplate) => {
    dialogType.value = type
    currentId.value = row?.id || ''
    dialogStep.value = 0
    Object.assign(formData, defaultForm(), row || {})
    formData.type = 'task'
    formData.params = (row?.params || [])
      .filter((item) => !isLegacyWorkspaceBindingParam(item))
      .map((item) => normalizeParam(item))
    normalizeParamSortOrders(false)
    assignParamForm(defaultParam())
    paramDialogVisible.value = false
    paramDialogIndex.value = -1
    assignResultForm(defaultResult())
    resultDialogVisible.value = false
    resultDialogIndex.value = -1
    assignWorkspaceForm(defaultWorkspace())
    workspaceDialogVisible.value = false
    workspaceDialogIndex.value = -1
    Object.assign(taskForm, defaultTaskForm())
    rawTaskResource.value = {}
    yamlParseError.value = ''
    lastAutoTaskName.value = taskForm.name
    activeIconGroup.value = getIconGroupName(formData.icon)
    dialogVisible.value = true
    dialogLoading.value = true
    await nextTick()
    try {
      if (row?.stageContent) {
        advancedYaml.value = row.stageContent
        const parsed = parseTaskYaml(row.stageContent)
        if (parsed) {
          applyParsedTask(parsed)
          syncParamsFromTask(false)
        } else {
          yamlParseError.value = 'Task YAML 不合法'
        }
      } else {
        syncGeneratedYaml(true)
        advancedYaml.value = formData.stageContent
        syncParamsFromTask(false)
      }
      await fetchCategories()
      await nextTick()
      formRef.value?.clearValidate()
    } finally {
      dialogLoading.value = false
    }
  }

  const resetForm = () => {
    dialogLoading.value = false
    dialogStep.value = 0
    currentId.value = ''
    paramDialogVisible.value = false
    paramDialogIndex.value = -1
    resultDialogVisible.value = false
    resultDialogIndex.value = -1
    workspaceDialogVisible.value = false
    workspaceDialogIndex.value = -1
    Object.assign(formData, defaultForm())
    assignParamForm(defaultParam())
    assignResultForm(defaultResult())
    assignWorkspaceForm(defaultWorkspace())
    Object.assign(taskForm, defaultTaskForm())
    rawTaskResource.value = {}
    advancedYaml.value = ''
    yamlParseError.value = ''
    imageDialogVisible.value = false
    currentImageStep.value = null
    imageRows.value = []
    imagePagination.value.current = 1
    activeIconGroup.value = '工具平台'
    formRef.value?.clearValidate()
    paramFormRef.value?.clearValidate()
    resultFormRef.value?.clearValidate()
    workspaceFormRef.value?.clearValidate()
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const taskResourceName = () => normalizeTektonName(formData.code || 'tekton-step')

  const buildTaskObject = () =>
    buildTaskResource(taskForm, rawTaskResource.value, taskResourceName())

  const syncGeneratedYaml = (syncAdvanced = true) => {
    taskForm.name = taskResourceName()
    taskForm.displayName = String(formData.name || '').trim()
    taskForm.description = String(formData.description || '').trim()
    let resource: Record<string, any>
    try {
      resource = buildTaskObject()
    } catch (error: any) {
      yamlParseError.value = error?.message || 'Task YAML 生成失败'
      ElMessage.warning(yamlParseError.value)
      return false
    }
    rawTaskResource.value = cloneValue(resource)
    const content = dumpTaskYaml(resource)
    yamlParseError.value = ''
    formData.stageContent = content
    if (syncAdvanced) {
      applyingFormToYaml = true
      advancedYaml.value = content
      nextTick(() => {
        applyingFormToYaml = false
      })
    }
    return true
  }

  const parseTaskYaml = (content: string, updateError = false) => {
    const result = parseTektonTaskYaml(content)
    if (!result.ok) {
      if (updateError) yamlParseError.value = result.error
      return null
    }
    if (updateError) yamlParseError.value = ''
    return result.data
  }

  const applyParsedTask = (parsed: TaskParseResult) => {
    rawTaskResource.value = cloneValue(parsed.resource)
    Object.assign(taskForm, parsed.form)
    taskForm.name = taskResourceName()
    lastAutoTaskName.value = taskForm.name || lastAutoTaskName.value
  }

  const parseAdvancedYamlToForm = (showMessage = false) => {
    const parsedResult = normalizeTaskYamlForSave(advancedYaml.value, taskResourceName())
    if (!parsedResult.ok) {
      yamlParseError.value = parsedResult.error
      return false
    }
    yamlParseError.value = ''
    const parsed = parsedResult.data
    if (!parsed) return false
    applyParsedTask(parsed)
    advancedYaml.value = parsed.normalizedYaml
    formData.stageContent = parsed.normalizedYaml
    if (showMessage) ElMessage.success('YAML 已校验并按编码修正')
    return true
  }

  const resourceName = (content?: string) => {
    const parsed = parseTaskYaml(content || '')
    return parsed?.form.name || ''
  }

  const resourceImages = (content?: string) => collectTaskImagesFromYaml(content || '')

  const normalizeTektonTaskParamType = (type?: string) => {
    const value = String(type || '').trim()
    return tektonTaskParamTypeSet.has(value) ? value : 'string'
  }

  const paramTypeFromTektonTaskType = (type?: string) => {
    const value = String(type || '').trim()
    if (value === 'array') return 'array'
    if (value === 'object') return 'object'
    return 'string'
  }

  const tektonTypeFromParamType = (type?: string): TektonTaskParam['type'] => {
    const value = String(type || '').trim()
    if (value === 'array') return 'array'
    if (value === 'object') return 'object'
    return 'string'
  }

  const taskPropertiesFromCompoundFields = (
    fields?: NonNullable<DevopsStepParam['config']>['compoundFields']
  ) =>
    (fields || []).reduce<NonNullable<TektonTaskParam['properties']>>((acc, field) => {
      const name = String(field.code || field.name || '').trim()
      if (!name) return acc
      acc[name] = { type: tektonTypeFromParamType(field.paramType) }
      return acc
    }, {})

  const compoundFieldsFromTaskProperties = (
    properties?: TektonTaskParam['properties']
  ): NonNullable<DevopsStepParam['config']>['compoundFields'] =>
    Object.entries(properties || {}).map(([name, property]) => ({
      name,
      code: name,
      defaultValue: '',
      paramType: paramTypeFromTektonTaskType(property.type),
      mode: 'params',
      required: false,
      readonly: false,
      description: '',
      selectList: [],
      allowCustom: false,
      runtimeMode: 'params',
      runtimeConfig: false,
      config: { ...defaultParam().config }
    }))

  const enumTextFromParam = (param?: Partial<DevopsStepParam>, taskParam?: TektonTaskParam) => {
    const values = (param?.selectList?.length ? param.selectList : param?.config?.options || [])
      .map((item) => String(item.value || '').trim())
      .filter(Boolean)
    if (values.length > 0) return Array.from(new Set(values)).join('\n')
    return (taskParam?.enum || []).join('\n')
  }

  const propertiesRawFromParam = (
    param?: Partial<DevopsStepParam>,
    taskParam?: TektonTaskParam
  ) => {
    const fromForm = taskPropertiesFromCompoundFields(param?.config?.compoundFields)
    const properties = Object.keys(fromForm).length > 0 ? fromForm : taskParam?.properties || {}
    return Object.keys(properties).length > 0 ? stringify(properties).trim() : ''
  }

  const propertiesRawFromResult = (result?: Partial<TektonTaskResult>) =>
    Object.keys(result?.properties || {}).length > 0 ? stringify(result?.properties).trim() : ''

  const parseParamEnumText = () =>
    Array.from(
      new Set(
        String(paramEnumText.value || '')
          .split('\n')
          .map((item) => item.trim())
          .filter(Boolean)
      )
    )

  const parseParamPropertiesRaw = () => {
    const raw = String(paramPropertiesRaw.value || '').trim()
    if (!raw) return {}
    let parsed: any
    try {
      parsed = parse(raw)
    } catch {
      throw new Error('对象属性必须是合法 YAML 对象')
    }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('对象属性必须是合法 YAML 对象')
    }
    return Object.entries(parsed).reduce<NonNullable<TektonTaskParam['properties']>>(
      (acc, [name, value]) => {
        const key = String(name || '').trim()
        if (!key) throw new Error('对象属性名称不能为空')
        if (!value || typeof value !== 'object' || Array.isArray(value)) {
          throw new Error(`对象属性 ${key} 必须配置 type`)
        }
        const type = String((value as Record<string, unknown>).type || 'string').trim()
        if (!['string', 'array', 'object'].includes(type)) {
          throw new Error(`对象属性 ${key} 类型只支持 string、array、object`)
        }
        acc[key] = { type: type as TektonTaskParam['type'] }
        return acc
      },
      {}
    )
  }

  const parseResultPropertiesRaw = () => {
    const raw = String(resultPropertiesRaw.value || '').trim()
    if (!raw) return {}
    let parsed: any
    try {
      parsed = parse(raw)
    } catch {
      throw new Error('Result 对象属性必须是合法 YAML 对象')
    }
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      throw new Error('Result 对象属性必须是合法 YAML 对象')
    }
    return Object.entries(parsed).reduce<NonNullable<TektonTaskResult['properties']>>(
      (acc, [name, value]) => {
        const key = String(name || '').trim()
        if (!key) throw new Error('Result 对象属性名称不能为空')
        if (!value || typeof value !== 'object' || Array.isArray(value)) {
          throw new Error(`Result 对象属性 ${key} 必须配置 type`)
        }
        const type = String((value as Record<string, unknown>).type || 'string').trim()
        if (!['string', 'array', 'object'].includes(type)) {
          throw new Error(`Result 对象属性 ${key} 类型只支持 string、array、object`)
        }
        acc[key] = { type: type as TektonTaskResult['type'] }
        return acc
      },
      {}
    )
  }

  const normalizeParam = (param: Partial<DevopsStepParam>): DevopsStepParam => ({
    ...defaultParam(),
    ...param,
    paramType: normalizeTektonTaskParamType(param.paramType),
    mode: 'params',
    config: {
      ...defaultParam().config,
      ...(param.config || {}),
      mappingField: '',
      provider: '',
      fullRow: false
    },
    runtimeMode: 'params',
    runtimeConfig: false,
    readonly: false
  })

  function isLegacyWorkspaceBindingParam(param: Partial<DevopsStepParam>) {
    const type = String(param.paramType || '').trim()
    return (
      param.mode === 'workspace' ||
      type === 'workspaceSecret' ||
      type === 'workspacePvc' ||
      (Boolean(param.config?.mappingField) &&
        (type === 'kubernetesSecretName' || type === 'kubernetesPvcName'))
    )
  }

  const paramTypeName = (type?: string) =>
    paramTypeOptions.find((item) => item.value === type)?.label || 'string'

  const paramConstraintSummary = (param: DevopsStepParam) => {
    const parts: string[] = []
    const enumCount = (param.selectList?.length || param.config?.options?.length || 0) as number
    const propertyCount = param.config?.compoundFields?.length || 0
    if (enumCount > 0) parts.push(`enum ${enumCount}`)
    if (propertyCount > 0) parts.push(`properties ${propertyCount}`)
    return parts.join(' / ') || '-'
  }

  const taskResultConstraintSummary = (result: TektonTaskResult) => {
    const propertyCount = Object.keys(result.properties || {}).length
    return propertyCount > 0 ? `properties ${propertyCount}` : '-'
  }

  const cloneParam = (param: Partial<DevopsStepParam>): DevopsStepParam => normalizeParam(param)

  const normalizeResult = (result: Partial<TektonTaskResult>): TektonTaskResult => ({
    ...defaultResult(),
    ...result,
    name: String(result.name || '').trim(),
    type: ['string', 'array', 'object'].includes(String(result.type || '').trim())
      ? (String(result.type || '').trim() as TektonTaskResult['type'])
      : 'string',
    description: String(result.description || '').trim(),
    value: String(result.value || '').trim(),
    properties: result.type === 'object' ? result.properties || {} : {}
  })

  const assignParamForm = (param: Partial<DevopsStepParam>) => {
    const next = cloneParam(param)
    Object.assign(paramForm, next)
    const taskParam = taskForm.params.find((item) => item.name === next.code)
    paramEnumText.value = enumTextFromParam(next, taskParam)
    paramPropertiesRaw.value = propertiesRawFromParam(next, taskParam)
  }

  const assignResultForm = (result: Partial<TektonTaskResult>) => {
    Object.assign(resultForm, normalizeResult(result))
    resultPropertiesRaw.value = propertiesRawFromResult(result)
  }

  const normalizeWorkspace = (workspace: Partial<TektonWorkspace>): TektonWorkspace => ({
    ...defaultWorkspace(),
    ...workspace,
    name: String(workspace.name || '').trim(),
    description: String(workspace.description || '').trim(),
    optional: workspace.optional === true,
    readOnly: workspace.readOnly === true,
    mountPath: String(workspace.mountPath || '').trim()
  })

  const assignWorkspaceForm = (workspace: Partial<TektonWorkspace>) => {
    Object.assign(workspaceForm, normalizeWorkspace(workspace))
  }

  const visibleParams = computed(() =>
    formData.params.map((item, index) => ({ ...item, __index: index }))
  )
  const visibleResults = computed(() =>
    taskForm.results.map((item, index) => ({ ...item, __index: index }))
  )

  const visibleStepParamCount = (item: DevopsStepTemplate) => {
    if (item.taskParams?.length) return item.taskParams.length
    const parsed = parseTaskYaml(item.stageContent || '')
    if (parsed) return parsed.form.params.length
    return (item.params || []).filter((param) => !isLegacyWorkspaceBindingParam(param)).length
  }

  const normalizeParamSortOrders = (renumber = true) => {
    formData.params = formData.params
      .map((item, index) => ({
        item,
        index,
        order: Number(item.sortOrder || 0) > 0 ? Number(item.sortOrder) : index + 1
      }))
      .sort((left, right) => left.order - right.order || left.index - right.index)
      .map(({ item }, index) =>
        normalizeParam({
          ...item,
          sortOrder: renumber ? index + 1 : item.sortOrder
        })
      )
  }

  const handlePlatformParamTypeChange = (param: DevopsStepParam) => {
    param.paramType = normalizeTektonTaskParamType(param.paramType)
    if (!param.config) param.config = { ...defaultParam().config }
    param.mode = 'params'
    param.runtimeMode = 'params'
    param.readonly = false
    param.runtimeConfig = false
    param.config.mappingField = ''
    param.config.provider = ''
    param.config.fullRow = false
  }

  const handleParamRequiredChange = (value: string | number | boolean) => {
    paramForm.required = value === true
  }

  const stepParamFromTaskParam = (item: TektonTaskParam, index: number, old?: DevopsStepParam) => {
    const code = String(item.name || '').trim()
    const enumOptions = (item.enum || []).map((value) => ({ label: value, value }))
    return normalizeParam({
      ...(old || {}),
      name: old?.name || item.description || code,
      code,
      defaultValue: old?.defaultValue ?? item.default ?? '',
      paramType: paramTypeFromTektonTaskType(item.type),
      mode: 'params',
      runtimeMode: 'params',
      required: old?.required ?? item.required ?? !String(item.default || '').trim(),
      description: old?.description || item.description || '',
      selectList: old?.selectList?.length ? old.selectList : enumOptions,
      config: {
        ...(old?.config || {}),
        options: old?.config?.options?.length ? old.config.options : enumOptions,
        compoundFields: old?.config?.compoundFields?.length
          ? old.config.compoundFields
          : compoundFieldsFromTaskProperties(item.properties)
      },
      sortOrder: index + 1
    })
  }

  const stepParamsFromTaskParams = (items: TektonTaskParam[], legacy: DevopsStepParam[] = []) => {
    const oldMap = new Map(legacy.map((item) => [item.code, item]))
    return items
      .filter((item) => String(item.name || '').trim())
      .map((item, index) => stepParamFromTaskParam(item, index, oldMap.get(item.name)))
  }

  const syncParamsFromTask = (showMessage = false) => {
    const parsed = parseTaskYaml(advancedYaml.value || formData.stageContent || '', true)
    if (!parsed) return
    applyParsedTask(parsed)
    formData.params = stepParamsFromTaskParams(parsed.form.params, formData.params)
    normalizeParamSortOrders()
    if (showMessage) ElMessage.success('参数已同步')
  }

  const syncResultsFromTask = (showMessage = false) => {
    const parsed = parseTaskYaml(advancedYaml.value || formData.stageContent || '', true)
    if (!parsed) return
    applyParsedTask(parsed)
    if (showMessage) ElMessage.success('Result 已同步')
  }

  const syncWorkspacesFromTask = (showMessage = false) => {
    const parsed = parseTaskYaml(advancedYaml.value || formData.stageContent || '', true)
    if (!parsed) return
    applyParsedTask(parsed)
    if (showMessage) ElMessage.success('Workspace 已同步')
  }

  const addWorkspace = () => {
    workspaceDialogIndex.value = -1
    assignWorkspaceForm(defaultWorkspace())
    workspaceDialogVisible.value = true
    requestAnimationFrame(() => workspaceFormRef.value?.clearValidate())
  }

  const addResult = () => {
    resultDialogIndex.value = -1
    assignResultForm(defaultResult())
    resultDialogVisible.value = true
    requestAnimationFrame(() => resultFormRef.value?.clearValidate())
  }

  const editWorkspace = (index: number) => {
    const workspace = taskForm.workspaces[index]
    if (!workspace) return
    workspaceDialogIndex.value = index
    assignWorkspaceForm(workspace)
    workspaceDialogVisible.value = true
    requestAnimationFrame(() => workspaceFormRef.value?.clearValidate())
  }

  const saveWorkspaceDialog = async () => {
    await workspaceFormRef.value?.validate()
    const next = normalizeWorkspace(workspaceForm)
    const duplicateIndex = taskForm.workspaces.findIndex(
      (item, index) => item.name === next.name && index !== workspaceDialogIndex.value
    )
    if (duplicateIndex >= 0) {
      ElMessage.warning(`Workspace 名称重复：${next.name}`)
      return
    }
    if (workspaceDialogIndex.value === -1) {
      taskForm.workspaces.push(next)
    } else {
      taskForm.workspaces.splice(workspaceDialogIndex.value, 1, next)
    }
    syncGeneratedYaml(true)
    workspaceDialogVisible.value = false
  }

  const editResult = (index: number) => {
    const result = taskForm.results[index]
    if (!result) return
    resultDialogIndex.value = index
    assignResultForm(result)
    resultDialogVisible.value = true
    requestAnimationFrame(() => resultFormRef.value?.clearValidate())
  }

  const saveResultDialog = async () => {
    await resultFormRef.value?.validate()
    const properties = resultForm.type === 'object' ? parseResultPropertiesRaw() : {}
    const next = normalizeResult({
      ...resultForm,
      properties
    })
    const duplicateIndex = taskForm.results.findIndex(
      (item, index) => item.name === next.name && index !== resultDialogIndex.value
    )
    if (duplicateIndex >= 0) {
      ElMessage.warning(`Result 名称重复：${next.name}`)
      return
    }
    if (resultDialogIndex.value === -1) {
      taskForm.results.push(next)
    } else {
      taskForm.results.splice(resultDialogIndex.value, 1, next)
    }
    syncGeneratedYaml(true)
    resultDialogVisible.value = false
  }

  const removeWorkspace = (index: number) => {
    const workspace = taskForm.workspaces[index]
    if (!workspace) return
    taskForm.workspaces.splice(index, 1)
    syncGeneratedYaml(true)
  }

  const removeResult = (index: number) => {
    taskForm.results.splice(index, 1)
    syncGeneratedYaml(true)
  }

  const validateWorkspaceList = () => {
    const names = new Set<string>()
    for (const item of taskForm.workspaces) {
      const name = String(item.name || '').trim()
      if (!name) {
        ElMessage.warning('Workspace 名称不能为空')
        return false
      }
      if (names.has(name)) {
        ElMessage.warning(`Workspace 名称重复：${name}`)
        return false
      }
      names.add(name)
    }
    return true
  }

  const syncTaskFormToYaml = (showMessage = false, showSuccess = false) => {
    const error = validateTaskForSave(taskForm)
    if (error) {
      yamlParseError.value = error
      if (showMessage) ElMessage.warning(error)
      return false
    }
    const synced = syncGeneratedYaml(true)
    if (synced && showMessage && showSuccess) ElMessage.success('Task 契约已同步')
    return synced
  }

  const addParam = () => {
    paramDialogIndex.value = -1
    assignParamForm({
      ...defaultParam(),
      sortOrder: formData.params.length + 1
    })
    paramDialogVisible.value = true
    requestAnimationFrame(() => paramFormRef.value?.clearValidate())
  }

  const applyTaskParamContractToStepParam = (param: DevopsStepParam) => {
    const enumValues = parseParamEnumText()
    const properties = param.paramType === 'object' ? parseParamPropertiesRaw() : {}
    const options = enumValues.map((value) => ({ label: value, value }))
    param.selectList = options
    if (!param.config) param.config = { ...defaultParam().config }
    param.config.options = options
    param.config.compoundFields =
      param.paramType === 'object' ? compoundFieldsFromTaskProperties(properties) : []
    return { enumValues, properties }
  }

  const upsertTaskParamFromStepParam = (
    param: DevopsStepParam,
    enumValues: string[],
    properties: NonNullable<TektonTaskParam['properties']>
  ) => {
    const code = String(param.code || '').trim()
    if (!code) return
    const index = taskForm.params.findIndex((item) => item.name === code)
    const previous = index >= 0 ? taskForm.params[index] : undefined
    const next: TektonTaskParam = {
      ...(previous || {}),
      name: code,
      type: tektonTypeFromParamType(param.paramType),
      default: String(param.defaultValue ?? ''),
      required: param.required === true,
      description: String(param.description || param.name || '').trim(),
      enum: enumValues,
      properties:
        param.paramType === 'object'
          ? properties
          : previous?.type === 'object'
            ? {}
            : previous?.properties
    }
    if (next.type !== 'object') {
      next.properties = {}
    }
    if (index >= 0) {
      taskForm.params.splice(index, 1, next)
    } else {
      taskForm.params.push(next)
    }
  }

  const editParam = (index: number) => {
    const param = formData.params[index]
    if (!param) return
    paramDialogIndex.value = index
    assignParamForm(param)
    paramDialogVisible.value = true
    requestAnimationFrame(() => paramFormRef.value?.clearValidate())
  }

  const saveParamDialog = async () => {
    await paramFormRef.value?.validate()
    const next = normalizeParam(paramForm)
    let taskParamContract: {
      enumValues: string[]
      properties: NonNullable<TektonTaskParam['properties']>
    }
    try {
      taskParamContract = applyTaskParamContractToStepParam(next)
    } catch (error: any) {
      ElMessage.warning(error?.message || '参数限制配置不合法')
      return
    }
    const duplicateIndex = formData.params.findIndex(
      (item, index) => item.code === next.code && index !== paramDialogIndex.value
    )
    if (duplicateIndex >= 0) {
      ElMessage.warning(`Task 参数名重复：${next.code}`)
      return
    }
    if (paramDialogIndex.value === -1) {
      formData.params.push(next)
    } else {
      formData.params.splice(paramDialogIndex.value, 1, next)
    }
    normalizeParamSortOrders()
    upsertTaskParamFromStepParam(next, taskParamContract.enumValues, taskParamContract.properties)
    if (!syncGeneratedYaml(true)) return
    paramDialogVisible.value = false
  }

  const removeParam = (index: number) => {
    const [removed] = formData.params.splice(index, 1)
    normalizeParamSortOrders()
    const code = String(removed?.code || '').trim()
    if (code) {
      taskForm.params = taskForm.params.filter((item) => item.name !== code)
      syncGeneratedYaml(true)
    }
  }

  const validateAdvancedYaml = (showMessage = true, showSuccess = false) => {
    const parsedResult = normalizeTaskYamlForSave(advancedYaml.value, taskResourceName())
    if (!parsedResult.ok) {
      yamlParseError.value = parsedResult.error
      if (showMessage) ElMessage.warning(parsedResult.error)
      return false
    }
    yamlParseError.value = ''
    const parsed = parsedResult.data
    const error = validateTaskForSave(parsed.form)
    if (error) {
      yamlParseError.value = error
      if (showMessage) ElMessage.warning(error)
      return false
    }
    applyParsedTask(parsed)
    advancedYaml.value = parsed.normalizedYaml
    formData.stageContent = parsed.normalizedYaml
    if (showMessage && showSuccess) ElMessage.success('YAML 校验通过')
    return true
  }

  const handleValidateYamlClick = () => {
    validateAdvancedYaml(true, true)
  }

  const formatAdvancedYaml = (showMessage = false) => {
    const parsedResult = normalizeTaskYamlForSave(advancedYaml.value, taskResourceName())
    if (!parsedResult.ok) {
      yamlParseError.value = parsedResult.error
      return false
    }
    yamlParseError.value = ''
    const parsed = parsedResult.data
    applyingFormToYaml = true
    advancedYaml.value = parsed.normalizedYaml
    formData.stageContent = parsed.normalizedYaml
    applyParsedTask(parsed)
    nextTick(() => {
      applyingFormToYaml = false
    })
    if (showMessage) ElMessage.success('YAML 已整理')
    return true
  }

  const validateBeforeLeaveStep = async (targetIndex: number) => {
    if (targetIndex === dialogStep.value) return true
    if (dialogStep.value === 0) {
      await formRef.value?.validateField(['name', 'code', 'categoryId'])
      if (!advancedYaml.value.trim()) {
        syncGeneratedYaml(true)
      } else {
        parseAdvancedYamlToForm(false)
      }
    }
    if (dialogStep.value === 1) {
      if (!validateAdvancedYaml(true)) return false
      syncParamsFromTask(false)
    }
    if (dialogStep.value === 2) {
      if (!syncTaskFormToYaml(true)) return false
    }
    if (dialogStep.value === 3) {
      if (!syncTaskFormToYaml(true)) return false
    }
    if (dialogStep.value === 4) {
      if (!validateWorkspaceList()) return false
      if (!syncTaskFormToYaml(true)) return false
    }
    return true
  }

  const handleNextStep = async (targetIndex?: number) => {
    const nextIndex =
      typeof targetIndex === 'number'
        ? targetIndex
        : Math.min(dialogStep.value + 1, wizardSteps.length - 1)
    if (!(await validateBeforeLeaveStep(nextIndex))) return false
    dialogStep.value = nextIndex
    return true
  }

  const handlePrevStep = async () => {
    const nextIndex = Math.max(dialogStep.value - 1, 0)
    if (!(await validateBeforeLeaveStep(nextIndex))) return
    dialogStep.value = nextIndex
  }

  const handleStepClick = async (index: number) => {
    if (index === dialogStep.value) return
    if (index < dialogStep.value) {
      if (!(await validateBeforeLeaveStep(index))) return
      dialogStep.value = index
      return
    }
    while (dialogStep.value < index) {
      const passed = await handleNextStep()
      if (!passed) return
    }
  }

  const taskParamsPayload = (
    items: TektonTaskParam[] = taskForm.params
  ): NonNullable<DevopsStepTemplateForm['taskParams']> =>
    items.map((item) => {
      const name = String(item.name || '').trim()
      const stepParam = formData.params.find((param) => param.code === name)
      return {
        name,
        type: item.type || 'string',
        defaultValue: String(item.default ?? ''),
        description: String(item.description || '').trim(),
        enum: item.enum || [],
        properties: item.type === 'object' ? item.properties || {} : {},
        required: stepParam?.required ?? item.required ?? !String(item.default || '').trim()
      }
    })

  const taskResultsPayload = (
    items: TektonTaskResult[] = taskForm.results
  ): NonNullable<DevopsStepTemplateForm['taskResults']> =>
    items.map((item) => ({
      name: String(item.name || '').trim(),
      type: item.type || 'string',
      description: String(item.description || '').trim(),
      value: String(item.value || '').trim(),
      properties: item.type === 'object' ? item.properties || {} : {}
    }))

  const taskWorkspacesPayload = (
    items: TektonWorkspace[] = taskForm.workspaces
  ): NonNullable<DevopsStepTemplateForm['taskWorkspaces']> =>
    items.map((item) => ({
      name: String(item.name || '').trim(),
      description: String(item.description || '').trim(),
      optional: item.optional === true,
      readOnly: item.readOnly === true,
      mountPath: String(item.mountPath || '').trim()
    }))

  const stepParamsPayload = (items: DevopsStepParam[] = formData.params) =>
    items
      .filter((item) => !isLegacyWorkspaceBindingParam(item))
      .map((item, index) =>
        normalizeParam({
          ...item,
          runtimeMode: 'params',
          runtimeConfig: false,
          readonly: false,
          sortOrder: index + 1
        })
      )

  const normalizedPayload = (): DevopsStepTemplateForm | null => {
    if (!validateWorkspaceList()) return null
    if (!syncTaskFormToYaml(false)) return null
    if (!syncGeneratedYaml(true)) return null
    if (!validateAdvancedYaml()) return null
    syncParamsFromTask(false)
    return {
      ...formData,
      type: 'task',
      params: stepParamsPayload(),
      taskParams: taskParamsPayload(),
      taskResults: taskResultsPayload(),
      taskWorkspaces: taskWorkspacesPayload()
    }
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    const payload = normalizedPayload()
    if (!payload) return
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsTektonStepApi(payload)
        ElMessage.success('新增成功，已同步 Tekton 实例')
      } else {
        await updateDevopsTektonStepApi(currentId.value, payload)
        ElMessage.success('保存成功，已同步 Tekton 实例')
      }
      dialogVisible.value = false
      await fetchList()
    } finally {
      submitLoading.value = false
    }
  }

  const handleStatusChange = async (row: DevopsStepTemplate, value: boolean) => {
    const nextStatus = value ? 1 : 0
    await updateDevopsTektonStepStatusApi(row.id, nextStatus)
    ElMessage.success(value ? '步骤已启用并同步' : '步骤已停用并删除远端资源')
    await fetchList()
  }

  const handleDelete = async (row: DevopsStepTemplate) => {
    await ElMessageBox.confirm(`确认删除 Tekton 步骤 ${row.name}？`, '删除确认', {
      type: 'warning'
    })
    await deleteDevopsTektonStepApi(row.id)
    ElMessage.success('删除成功，已删除远端资源')
    await fetchList()
  }

  const fetchImageRows = async () => {
    imageListLoading.value = true
    try {
      const res = await listDevopsTektonStepImageApi({
        page: imagePagination.value.current,
        pageSize: imagePagination.value.size,
        code: imageDialogMode.value === 'row' ? currentImageStep.value?.code : undefined
      })
      imageRows.value = (res.items || []).map((item, index) => ({
        ...item,
        rowKey: `${item.id}-${item.imageType}-${item.paramName || item.image}-${index}`,
        newImage: item.image || ''
      }))
      imagePagination.value.total = res.total || 0
    } finally {
      imageListLoading.value = false
    }
  }

  const handleImageSizeChange = (size: number) => {
    imagePagination.value.size = size
    imagePagination.value.current = 1
    fetchImageRows()
  }

  const handleImageCurrentChange = (page: number) => {
    imagePagination.value.current = page
    fetchImageRows()
  }

  const openImageReplaceDialog = async (mode: ImageDialogMode, row?: DevopsStepTemplate) => {
    imageDialogMode.value = mode
    currentImageStep.value = row || null
    imageRows.value = []
    imagePagination.value.current = 1
    imageDialogVisible.value = true
    await fetchImageRows()
  }

  const handleImageReplace = async () => {
    const changes = imageChangedRows.value
    if (changes.length === 0) {
      ElMessage.warning('请先修改镜像地址')
      return
    }
    const items: UpdateTektonStepImageItem[] = changes.map((item) => ({
      id: item.id,
      image: item.image,
      newImage: item.newImage.trim(),
      imageType: item.imageType,
      paramName: item.paramName
    }))
    imageReplaceLoading.value = true
    try {
      const res = await batchUpdateDevopsTektonStepImageApi(items)
      imageDialogVisible.value = false
      ElMessage.success(
        res.updated > 0
          ? `已提交 ${res.total} 条变更，更新 ${res.updated} 个步骤`
          : '没有镜像发生变化'
      )
      await fetchList()
    } finally {
      imageReplaceLoading.value = false
    }
  }

  watch(advancedYaml, (value) => {
    if (applyingFormToYaml) return
    const parsed = parseTaskYaml(value, true)
    if (!parsed) return
    applyParsedTask(parsed)
    formData.stageContent = value
  })

  watch(
    () => formData.code,
    (value) => {
      if (dialogType.value !== 'add') return
      if (taskForm.name && taskForm.name !== lastAutoTaskName.value) return
      const next = normalizeTektonName(String(value || 'tekton-step'))
      taskForm.name = next
      lastAutoTaskName.value = next
    }
  )

  let activatedOnce = false
  onMounted(refreshAll)
  onActivated(() => {
    if (!activatedOnce) {
      activatedOnce = true
      return
    }
    refreshAll()
  })
</script>

<style scoped lang="scss">
  .devops-tekton-step-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .step-card {
    display: flex;
    flex-direction: column;
    min-height: 214px;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
    box-shadow: 0 8px 22px rgb(15 23 42 / 4%);
  }

  .step-card-head {
    display: grid;
    grid-template-columns: 42px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    padding: 12px 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .step-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 42px;
    height: 42px;
    font-size: 24px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .step-title {
    min-width: 0;
  }

  .name,
  .code,
  .desc,
  .resource-line strong {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .name {
    display: block;
    font-size: 15px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .code {
    display: block;
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .step-card-body {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 10px;
    padding: 12px 14px;
  }

  .tag-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .desc {
    display: block;
    min-height: 20px;
    margin: 0;
    color: var(--el-text-color-secondary);
  }

  .resource-line {
    display: grid;
    grid-template-columns: 76px minmax(0, 1fr);
    gap: 8px;
    align-items: center;
    padding: 8px 10px;
    font-size: 12px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .resource-line span {
    color: var(--el-text-color-secondary);
  }

  .card-image-line {
    display: grid;
    grid-template-columns: 44px minmax(0, 1fr);
    gap: 6px;
    align-items: center;
    padding: 8px 10px;
    min-width: 0;
    font-size: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .card-field-label {
    flex: 0 0 auto;
    color: var(--el-text-color-secondary);
  }

  .image-tag-list {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    min-width: 0;
    overflow: hidden;
  }

  .step-card-footer {
    display: grid;
    grid-template-columns: max-content auto;
    column-gap: 8px;
    align-items: center;
    justify-content: space-between;
    min-height: 44px;
    padding: 8px 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .created-time {
    line-height: 18px;
    white-space: nowrap;
  }

  .footer-actions {
    display: inline-flex;
    flex-shrink: 0;
    gap: 10px;
    align-items: center;
  }

  .footer-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  .tekton-step-overview {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    justify-content: space-between;
    min-height: 48px;
    padding: 8px 12px;
    margin-bottom: 14px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .overview-metrics {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    min-width: 0;
  }

  .overview-item {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    min-height: 32px;
    padding: 0 10px;
    font-size: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .overview-item span {
    color: var(--el-text-color-secondary);
  }

  .overview-item strong {
    color: var(--el-text-color-primary);
  }

  :deep(.devops-polish-dialog) {
    display: flex;
    flex-direction: column;
    height: min(88vh, calc(100vh - 36px));
    max-height: calc(100vh - 36px);
    overflow: hidden;
    border-radius: 14px;
  }

  :deep(.devops-polish-dialog .el-dialog__header) {
    flex: 0 0 auto;
    padding: 18px 20px 12px;
    margin-right: 0;
  }

  :deep(.devops-polish-dialog .el-dialog__body) {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
    padding: 0 20px 10px;
    overflow: hidden;
  }

  :deep(.devops-polish-dialog .el-dialog__footer) {
    flex: 0 0 auto;
    padding: 12px 20px 18px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  :deep(.devops-polish-dialog .el-form) {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
  }

  .wizard-steps {
    position: relative;
    display: flex;
    flex: 0 0 auto;
    gap: 0;
    padding: 0 4px 2px;
    margin-bottom: 6px;

    &::before {
      position: absolute;
      right: 64px;
      bottom: 19px;
      left: 64px;
      height: 1px;
      content: '';
      background: var(--el-border-color-lighter);
    }
  }

  .wizard-step {
    position: relative;
    z-index: 1;
    display: flex;
    flex: 1;
    gap: 10px;
    align-items: center;
    justify-content: center;
    min-width: 0;
    min-height: 34px;
    padding: 0 10px;
    text-align: left;
    cursor: pointer;
    background: transparent;
    border: 0;

    &.active {
      .wizard-index {
        color: #fff;
        background: var(--el-color-primary);
        border-color: var(--el-color-primary);
        box-shadow: 0 0 0 3px var(--el-color-primary-light-9);
      }

      .wizard-text strong {
        color: var(--el-color-primary);
      }
    }

    &.done {
      .wizard-index {
        color: var(--el-color-primary);
        background: var(--el-color-primary-light-9);
        border-color: var(--el-color-primary-light-6);
      }
    }
  }

  .wizard-index {
    display: inline-flex;
    flex: 0 0 auto;
    align-items: center;
    justify-content: center;
    width: 24px;
    height: 24px;
    font-weight: 700;
    color: var(--el-color-primary);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 50%;
  }

  .wizard-text {
    min-width: 0;

    strong,
    em {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    strong {
      font-size: 13px;
      color: var(--el-text-color-primary);
    }

    em {
      margin-top: 1px;
      font-size: 11px;
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }

  .dialog-step-panel {
    display: flex;
    flex: 1 1 auto;
    height: auto;
    min-height: 0;
    max-height: none;
    padding: 2px 4px 0 0;
    overflow: hidden;
  }

  .config-panel :deep(.el-input__wrapper),
  .config-panel :deep(.el-select__wrapper),
  .param-edit-form :deep(.el-input__wrapper),
  .param-edit-form :deep(.el-select__wrapper),
  .workspace-edit-form :deep(.el-input__wrapper),
  .workspace-edit-form :deep(.el-select__wrapper) {
    min-height: 42px;
    padding: 0 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  .config-panel :deep(.el-select__wrapper),
  .param-edit-form :deep(.el-select__wrapper),
  .workspace-edit-form :deep(.el-select__wrapper) {
    padding: 10px 12px;
  }

  .config-panel :deep(.el-select__selection),
  .param-edit-form :deep(.el-select__selection),
  .workspace-edit-form :deep(.el-select__selection) {
    min-width: 0;
  }

  .config-panel :deep(.el-textarea__inner),
  .param-edit-form :deep(.el-textarea__inner),
  .workspace-edit-form :deep(.el-textarea__inner) {
    padding: 10px 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  .config-panel :deep(.el-input__wrapper:hover),
  .config-panel :deep(.el-select__wrapper:hover),
  .config-panel :deep(.el-textarea__inner:hover),
  .param-edit-form :deep(.el-input__wrapper:hover),
  .param-edit-form :deep(.el-select__wrapper:hover),
  .param-edit-form :deep(.el-textarea__inner:hover),
  .workspace-edit-form :deep(.el-input__wrapper:hover),
  .workspace-edit-form :deep(.el-select__wrapper:hover),
  .workspace-edit-form :deep(.el-textarea__inner:hover) {
    box-shadow: 0 0 0 1px #94a3b8 inset;
  }

  .config-panel :deep(.el-input__wrapper.is-focus),
  .config-panel :deep(.el-select__wrapper.is-focused),
  .config-panel :deep(.el-textarea__inner:focus),
  .param-edit-form :deep(.el-input__wrapper.is-focus),
  .param-edit-form :deep(.el-select__wrapper.is-focused),
  .param-edit-form :deep(.el-textarea__inner:focus),
  .workspace-edit-form :deep(.el-input__wrapper.is-focus),
  .workspace-edit-form :deep(.el-select__wrapper.is-focused),
  .workspace-edit-form :deep(.el-textarea__inner:focus) {
    box-shadow:
      0 0 0 1px #2563eb inset,
      0 0 0 4px rgb(37 99 235 / 14%);
  }

  .config-panel :deep(.el-select__caret),
  .param-edit-form :deep(.el-select__caret),
  .workspace-edit-form :deep(.el-select__caret) {
    width: 16px;
    height: 16px;
    color: #334155;
  }

  .config-panel :deep(.el-select__caret svg),
  .param-edit-form :deep(.el-select__caret svg),
  .workspace-edit-form :deep(.el-select__caret svg) {
    width: 14px;
    height: 14px;
  }

  .icon-select-panel {
    display: flex;
    flex-direction: column;
    gap: 12px;
    width: 100%;
    padding: 12px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .icon-group-tabs {
    align-self: flex-start;
  }

  .icon-picker {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(124px, 1fr));
    gap: 10px;
    width: 100%;
  }

  .icon-picker button {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    justify-content: center;
    min-width: 0;
    min-height: 36px;
    padding: 6px 10px;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .icon-picker button.active {
    background: var(--el-color-primary-light-9);
    border-color: var(--el-color-primary);
  }

  .icon-picker span {
    overflow: hidden;
    color: var(--el-text-color-primary);
    line-height: 18px;
    text-overflow: clip;
    white-space: normal;
  }

  .config-panel {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 8px 12px;
    margin-bottom: 0;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .scroll-panel {
    overflow: auto;
    scrollbar-gutter: stable;
  }

  .panel-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .yaml-panel-head {
    min-height: 24px;
    margin-bottom: 4px;
    line-height: 24px;
  }

  .yaml-panel-head strong,
  .yaml-panel-head span {
    display: inline;
  }

  .yaml-panel-head strong {
    margin-right: 10px;
  }

  .yaml-panel-head span {
    margin-top: 0;
  }

  .panel-head strong,
  .panel-head span {
    display: block;
  }

  .panel-head strong {
    color: var(--el-text-color-primary);
  }

  .panel-head span {
    margin-top: 3px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .panel-actions {
    display: inline-flex;
    gap: 2px;
    align-items: center;
  }

  .param-table-wrap,
  .result-table-wrap,
  .workspace-table-wrap {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
  }

  .task-advanced-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 14px;
    min-width: 0;
  }

  .step-advanced-section {
    min-width: 0;
    padding-top: 14px;
    margin-top: 14px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .section-title {
    display: flex;
    gap: 10px;
    align-items: baseline;
    min-width: 0;
    margin-bottom: 12px;
  }

  .section-title strong {
    flex: 0 0 auto;
    color: var(--el-text-color-primary);
  }

  .section-title span {
    min-width: 0;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .step-advanced-list {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px;
    min-width: 0;
  }

  .advanced-yaml-field {
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 12px;
    margin-bottom: 0;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .advanced-yaml-field :deep(.el-form-item__content) {
    display: flex;
    flex-direction: column;
    align-items: stretch;
    min-width: 0;
  }

  .advanced-yaml-field :deep(.el-textarea__inner) {
    font-family: var(--ax-font-mono);
    line-height: 20px;
  }

  .step-advanced-field {
    min-height: 230px;
  }

  .advanced-field-tip {
    margin-top: 8px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .param-table-wrap :deep(.el-empty),
  .result-table-wrap :deep(.el-empty),
  .workspace-table-wrap :deep(.el-empty) {
    height: 100%;
  }

  .param-table,
  .result-table,
  .workspace-table {
    height: 100%;
  }

  .param-table :deep(.el-table__cell),
  .result-table :deep(.el-table__cell),
  .workspace-table :deep(.el-table__cell) {
    padding: 8px 0;
  }

  .param-table :deep(.cell),
  .result-table :deep(.cell),
  .workspace-table :deep(.cell) {
    line-height: 20px;
  }

  .param-table :deep(.el-button.is-text),
  .result-table :deep(.el-button.is-text),
  .workspace-table :deep(.el-button.is-text) {
    padding: 0 2px;
    margin-left: 0;
  }

  .table-row-actions {
    display: inline-flex;
    gap: 10px;
    align-items: center;
    justify-content: flex-end;
    width: 100%;
    min-width: 0;
    white-space: nowrap;
  }

  :deep(.param-edit-dialog),
  :deep(.workspace-edit-dialog) {
    height: auto;
    max-height: calc(100vh - 36px);
  }

  :deep(.param-edit-dialog .el-dialog__body),
  :deep(.workspace-edit-dialog .el-dialog__body) {
    max-height: min(74vh, 660px);
    padding: 18px 28px 8px;
    overflow: auto;
  }

  :deep(.param-edit-dialog .el-dialog__footer),
  :deep(.workspace-edit-dialog .el-dialog__footer) {
    padding: 12px 22px 16px;
  }

  .param-edit-form,
  .workspace-edit-form {
    padding: 0;
  }

  .param-edit-form :deep(.el-form-item),
  .workspace-edit-form :deep(.el-form-item) {
    margin-bottom: 14px;
  }

  .workspace-edit-form :deep(.el-form-item__label) {
    white-space: nowrap;
  }

  .param-edit-form :deep(.el-form-item__label) {
    white-space: nowrap;
  }

  .form-label-with-help {
    display: inline-flex;
    gap: 4px;
    align-items: center;
    max-width: 100%;
    white-space: nowrap;
  }

  .field-help {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 16px;
    height: 16px;
    font-size: 12px;
    line-height: 1;
    color: var(--el-text-color-secondary);
    cursor: help;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 50%;
  }

  .param-edit-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 20px;
    row-gap: 0;
  }

  .param-edit-full,
  .param-description,
  .param-contract-tip {
    grid-column: 1 / -1;
  }

  .param-contract-tip {
    margin-top: -4px;
  }

  .param-contract-tip :deep(.el-form-item__content) {
    min-height: 28px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .param-description {
    margin-bottom: 10px;
  }

  .workspace-form-tip {
    width: 100%;
    margin-top: 8px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .dialog-footer-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .yaml-panel {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    margin-bottom: 0;
    overflow: hidden;
  }

  .yaml-panel :deep(.yaml-editor-pro) {
    flex: 1;
    min-height: 0;
  }

  .yaml-field-hints {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    margin-top: 6px;
  }

  .image-strip {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    align-items: center;
    padding: 8px 10px;
    margin-bottom: 10px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .image-strip.compact {
    min-height: 34px;
    padding: 5px 8px;
    margin-bottom: 6px;
  }

  .image-strip span {
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .image-tag {
    max-width: min(100%, 260px);
    min-width: 0;
    cursor: pointer;
  }

  .image-tag :deep(.el-tag__content),
  .image-tag-text {
    display: inline-block;
    max-width: 100%;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  :deep(.image-replace-dialog) {
    display: flex;
    flex-direction: column;
    height: min(76vh, calc(100vh - 48px));
    max-height: calc(100vh - 48px);
    overflow: hidden;
    border-radius: 12px;
  }

  :deep(.image-replace-dialog .el-dialog__header) {
    flex: 0 0 auto;
    padding: 18px 20px 12px;
    margin-right: 0;
  }

  :deep(.image-replace-dialog .el-dialog__body) {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
    padding: 0 20px 12px;
    overflow: hidden;
  }

  :deep(.image-replace-dialog .el-dialog__footer) {
    flex: 0 0 auto;
    padding: 12px 20px 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .image-dialog-layout {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
  }

  .image-dialog-head {
    display: flex;
    flex: 0 0 auto;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    padding: 10px 12px;
    margin-bottom: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .image-dialog-stats {
    display: flex;
    flex-wrap: wrap;
    gap: 10px;
    min-width: 0;
  }

  .image-stat {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    min-height: 30px;
    padding: 0 10px;
    font-size: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .image-stat span {
    color: var(--el-text-color-secondary);
  }

  .image-stat strong {
    color: var(--el-text-color-primary);
  }

  .image-manage-table-wrap {
    flex: 1 1 auto;
    min-height: 0;
    overflow: hidden;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .image-manage-table {
    height: 100%;
  }

  .image-manage-table :deep(.el-table__cell) {
    padding: 9px 0;
  }

  .image-manage-table :deep(.el-table__inner-wrapper::before) {
    display: none;
  }

  .image-manage-table :deep(.el-input__wrapper) {
    min-height: 32px;
  }

  .image-row-id {
    font-family: var(--ax-font-mono);
    font-size: 12px;
    color: var(--el-text-color-regular);
  }

  .image-name-cell {
    display: flex;
    flex-direction: column;
    min-width: 0;
  }

  .image-row-name,
  .image-row-code {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .image-row-name {
    color: var(--el-text-color-primary);
  }

  .image-row-code {
    margin-top: 3px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .image-editor-cell {
    display: grid;
    grid-template-columns: minmax(0, 1fr) max-content;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .image-input {
    min-width: 0;
  }

  .image-dialog-pagination {
    display: flex;
    justify-content: center;
    padding-top: 12px;
  }

  .image-dialog-footer {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    width: 100%;
  }

  .image-dialog-footer span {
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  @media (max-width: 720px) {
    .tekton-step-overview,
    .image-dialog-head,
    .image-dialog-footer {
      align-items: stretch;
    }

    .tekton-step-overview,
    .image-dialog-head,
    .image-dialog-footer {
      flex-direction: column;
    }

    .overview-metrics,
    .image-dialog-stats {
      width: 100%;
    }

    .image-dialog-footer .dialog-footer-actions {
      width: 100%;
    }

    .overview-item,
    .image-stat {
      flex: 1;
      justify-content: space-between;
    }

    .image-editor-cell {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  :global(.devops-step-template-dropdown) {
    max-width: min(640px, calc(100vw - 32px));
    padding: 6px 0;
    overflow: hidden !important;
    background: #ffffff !important;
    border: 1px solid #e2e8f0 !important;
    border-radius: 14px !important;
    box-shadow: 0 16px 34px rgb(15 23 42 / 12%) !important;
  }

  :global(.devops-step-template-dropdown .el-select-dropdown__wrap) {
    max-height: 260px !important;
    background: #ffffff !important;
  }

  :global(.devops-step-template-dropdown .el-select-dropdown__list) {
    padding: 4px 0 8px !important;
    background: #ffffff !important;
  }

  :global(.devops-step-template-dropdown .el-select-dropdown__item),
  :global(.devops-step-template-dropdown .el-cascader-node),
  :global(.devops-step-template-dropdown .el-tree-node__content) {
    height: auto !important;
    min-height: 44px !important;
    padding: 10px 12px !important;
    overflow: visible !important;
    line-height: 1.35 !important;
    color: #334155 !important;
    white-space: normal !important;
    background: #ffffff !important;
    border-radius: 0 !important;
    transition:
      background-color 0.16s ease,
      color 0.16s ease;
  }

  :global(.devops-step-template-dropdown .el-select-dropdown__item:hover),
  :global(.devops-step-template-dropdown .el-select-dropdown__item.is-hovering),
  :global(.devops-step-template-dropdown .el-cascader-node:hover),
  :global(.devops-step-template-dropdown .el-cascader-node.in-active-path),
  :global(.devops-step-template-dropdown .el-tree-node__content:hover) {
    background: #f8fafc !important;
  }

  :global(.devops-step-template-dropdown .el-select-dropdown__item.is-selected),
  :global(.devops-step-template-dropdown .el-cascader-node.is-selectable.in-checked-path),
  :global(.devops-step-template-dropdown .el-tree-node.is-current > .el-tree-node__content) {
    font-weight: 600 !important;
    color: #0f172a !important;
    background: #ffffff !important;
  }

  :global(.devops-step-template-dropdown .el-cascader-node__label),
  :global(.devops-step-template-dropdown .el-tree-node__label),
  :global(.devops-step-template-dropdown .el-select-dropdown__item span) {
    overflow: visible !important;
    text-overflow: clip !important;
    white-space: normal !important;
    word-break: break-word;
  }
</style>
