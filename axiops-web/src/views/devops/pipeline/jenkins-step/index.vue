<template>
  <div class="devops-jenkins-step-page">
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
      empty-text="暂无 Jenkins 步骤"
      @add="openDialog('add')"
      @refresh="refreshAll"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #empty-actions>
        <ElButton type="primary" :icon="Plus" @click="openDialog('add')">新增</ElButton>
      </template>

      <template #default="{ item }">
        <div class="step-card">
          <div class="step-card-head">
            <div class="step-icon" :style="{ color: item.iconColor || '#64748b' }">
              <ArtSvgIcon :icon="item.icon || 'simple-icons:jenkins'" />
            </div>
            <div class="step-title">
              <span class="name" :title="item.name">{{ item.name }}</span>
              <span class="code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElSwitch
              :model-value="item.status === 1"
              @change="(val) => handleStatusChange(item, val)"
            />
          </div>
          <div class="step-card-body">
            <div class="tag-row">
              <ElTag size="small" type="info">{{ item.categoryName || '未分类' }}</ElTag>
              <ElTag size="small" :type="item.type === 'manual_approval' ? 'warning' : 'success'">
                {{ stepTypeName(item.type) }}
              </ElTag>
              <ElTag size="small" type="success">Jenkins</ElTag>
              <ElTag size="small">{{ item.params?.length || 0 }} 个参数</ElTag>
            </div>
            <p class="desc" :title="item.description || '暂无描述'">{{
              item.description || '暂无描述'
            }}</p>
          </div>
          <div class="step-card-footer">
            <span class="created-time" :title="`创建时间：${smartFormatTime(item.createdAt)}`">
              创建：{{ smartFormatTime(item.createdAt) }}
            </span>
            <span class="footer-actions">
              <ElButton text type="primary" @click="openDialog('edit', item)">配置</ElButton>
              <ElButton text type="danger" @click="handleDelete(item)">删除</ElButton>
            </span>
          </div>
        </div>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增步骤' : '配置步骤'"
      width="1180px"
      align-center
      class="devops-polish-dialog step-dialog"
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
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="名称" prop="name">
                <ElInput v-model="formData.name" placeholder="请输入名称" />
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="编码" prop="code">
                <ElInput
                  v-model="formData.code"
                  :disabled="dialogType === 'edit'"
                  placeholder="例如 dockerBuild"
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
                  @visible-change="handleCategoryVisible"
                >
                  <ElOption
                    v-for="item in categoryOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="引擎类型">
                <ElInput model-value="Jenkins" disabled />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElFormItem label="步骤类型" prop="type">
            <ElSegmented v-model="formData.type" :options="stepTypeOptions" />
          </ElFormItem>
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
          <ElFormItem label="步骤描述" prop="description">
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
        </div>

        <div v-if="dialogStep === 1" class="dialog-step-panel">
          <ElFormItem class="full-form-item" label="步骤代码" prop="stageContent">
            <JenkinsPipelineEditor
              ref="pipelineEditorRef"
              v-model="formData.stageContent"
              height="calc(100vh - 330px)"
              :content-type="isPostCategory ? 'post' : 'steps'"
              @extract-params="syncParamsFromStage"
            />
          </ElFormItem>
        </div>

        <div v-show="dialogStep === 2" class="dialog-step-panel param-step-panel">
          <div class="artifact-section">
            <div class="artifact-head">
              <div>
                <strong>产物归档</strong>
                <span>步骤执行后需要归档的报告或制品文件</span>
              </div>
              <div class="artifact-switch">
                <span>启用归档</span>
                <ElSwitch v-model="formData.artifactConfig!.enabled" />
              </div>
            </div>
            <ElRow v-if="formData.artifactConfig?.enabled" :gutter="14" class="artifact-grid">
              <ElCol :span="5">
                <ElFormItem label="产物类型">
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-model="formData.artifactConfig!.type"
                    style="width: 100%"
                  >
                    <ElOption label="报告" value="report" />
                    <ElOption label="制品" value="artifact" />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
              <ElCol :span="6">
                <ElFormItem label="产物名称">
                  <ElInput v-model="formData.artifactConfig!.name" placeholder="例如 扫描报告" />
                </ElFormItem>
              </ElCol>
              <ElCol :span="9">
                <ElFormItem label="文件路径">
                  <ElInput
                    v-model="formData.artifactConfig!.path"
                    placeholder="例如 reports/scan.html"
                  />
                </ElFormItem>
              </ElCol>
              <ElCol :span="4">
                <ElFormItem label="必须存在" class="param-switch-field">
                  <ElSwitch v-model="formData.artifactConfig!.required" />
                </ElFormItem>
              </ElCol>
            </ElRow>
          </div>

          <div class="param-section">
            <div class="section-head">
              <div>
                <strong>参数管理</strong>
                <span>从 parameters、environment、params/env 读取和变量引用中提取参数</span>
              </div>
              <div class="section-actions">
                <span class="param-stat">
                  <strong>{{ formData.params.length }}</strong>
                  <em>参数</em>
                </span>
                <span class="param-stat warning">
                  <strong>{{ missingExtractedCount }}</strong>
                  <em>待同步</em>
                </span>
                <ElButton type="primary" plain @click="syncParamsFromStage">从代码同步</ElButton>
                <ElButton type="primary" :icon="Plus" @click="addParam">新增参数</ElButton>
              </div>
            </div>

            <div class="param-table-wrap">
              <ElEmpty v-if="formData.params.length === 0" description="暂无参数">
                <ElButton type="primary" plain @click="syncParamsFromStage">
                  从代码同步参数
                </ElButton>
              </ElEmpty>
              <ElTable v-else class="param-table" :data="formData.params" border>
                <ElTableColumn type="index" label="#" width="56" align="center" />
                <ElTableColumn label="参数名称" min-width="160" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.name || '未配置名称' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="参数编码" min-width="170" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.code || '未配置编码' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="类型" width="120">
                  <template #default="{ row }">
                    <ElTag size="small" effect="plain">{{ paramTypeName(row.paramType) }}</ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="字段数" width="88" align="center">
                  <template #default="{ row }">
                    {{
                      row.paramType === 'objectList' ? row.config?.compoundFields?.length || 0 : '-'
                    }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="模式" width="110">
                  <template #default="{ row }">
                    <ElTag size="small" :type="modeTagType(row.mode)">
                      {{ modeName(row.mode) }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="必填" width="80" align="center">
                  <template #default="{ row }">
                    <ElTag size="small" :type="row.required ? 'danger' : 'info'" effect="plain">
                      {{ row.required ? '是' : '否' }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="只读" width="80" align="center">
                  <template #default="{ row }">
                    <ElTag size="small" :type="row.readonly ? 'warning' : 'info'" effect="plain">
                      {{ row.readonly ? '是' : '否' }}
                    </ElTag>
                  </template>
                </ElTableColumn>
                <ElTableColumn label="排序" prop="sortOrder" width="80" align="center" />
                <ElTableColumn label="描述" min-width="180" show-overflow-tooltip>
                  <template #default="{ row }">
                    {{ row.description || '-' }}
                  </template>
                </ElTableColumn>
                <ElTableColumn label="操作" width="120" fixed="right" align="center">
                  <template #default="{ $index }">
                    <ElButton text type="primary" @click="editParam($index)">编辑</ElButton>
                    <ElButton text type="danger" @click="removeParam($index)">删除</ElButton>
                  </template>
                </ElTableColumn>
              </ElTable>
            </div>
          </div>
        </div>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton :disabled="dialogLoading" @click="dialogVisible = false">取消</ElButton>
          <ElButton v-if="dialogStep > 0" @click="dialogStep -= 1">上一步</ElButton>
          <ElButton
            v-if="dialogStep < 2"
            type="primary"
            :disabled="dialogLoading"
            @click="handleNextStep"
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
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="paramDialogVisible"
      :title="paramDialogIndex === -1 ? '新增参数' : '编辑参数'"
      width="920px"
      append-to-body
      class="devops-polish-dialog param-edit-dialog"
      :close-on-click-modal="false"
    >
      <ElForm
        ref="paramFormRef"
        :model="paramForm"
        :rules="paramRules"
        label-width="116px"
        class="param-edit-form"
      >
        <div class="param-edit-grid">
          <ElFormItem prop="name" label="参数名称">
            <ElInput
              v-model="paramForm.name"
              :disabled="isEditingParamReadonly"
              placeholder="例如 Git 工作目录"
            />
          </ElFormItem>
          <ElFormItem prop="code" label="参数编码">
            <ElInput
              v-model="paramForm.code"
              :disabled="isEditingParamReadonly"
              placeholder="例如 GIT_DIR"
            />
          </ElFormItem>
          <ElFormItem label="模式" class="param-edit-full">
            <ElSegmented
              v-model="paramForm.mode"
              :disabled="isEditingParamReadonly"
              :options="modeOptions"
              @change="() => handleParamModeChange(paramForm)"
            />
          </ElFormItem>
          <ElFormItem label="参数类型">
            <ElSelect
              popper-class="devops-step-template-dropdown"
              v-model="paramForm.paramType"
              :disabled="isEditingParamReadonly"
              style="width: 100%"
              @change="handleParamTypeChange(paramForm)"
            >
              <ElOption
                v-for="item in availableParamTypeOptions(paramForm.mode)"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="默认值">
            <ElSwitch
              v-if="paramForm.paramType === 'booleanParam'"
              v-model="paramForm.defaultValue"
              :disabled="isEditingParamReadonly"
              active-value="true"
              inactive-value="false"
            />
            <ElInputNumber
              v-else-if="paramForm.paramType === 'number'"
              :model-value="numberParamValue(paramForm.defaultValue)"
              :disabled="isEditingParamReadonly"
              controls-position="right"
              style="width: 100%"
              placeholder="请输入数字"
              @update:model-value="
                (value: number | undefined) => setParamNumberDefault(paramForm, value)
              "
            />
            <ElInput
              v-else-if="paramForm.paramType === 'text'"
              v-model="paramForm.defaultValue"
              :disabled="isEditingParamReadonly"
              type="textarea"
              :rows="5"
              resize="vertical"
              placeholder="请输入多行文本"
            />
            <ElInput
              v-else
              v-model="paramForm.defaultValue"
              :disabled="isEditingParamReadonly"
              placeholder="请输入默认值"
            />
          </ElFormItem>
          <ElFormItem label="排序">
            <ElInputNumber
              v-model="paramForm.sortOrder"
              :disabled="isEditingParamReadonly"
              :min="0"
              style="width: 100%"
            />
          </ElFormItem>
          <ElFormItem :label="paramForm.mode === 'paas' ? '输出方式' : '运行方式'">
            <ElSelect
              popper-class="devops-step-template-dropdown"
              v-model="paramForm.runtimeMode"
              :disabled="isEditingParamReadonly || paramForm.mode !== 'paas'"
              style="width: 100%"
              @change="() => handleRuntimeModeChange(paramForm)"
            >
              <ElOption
                v-for="item in runtimeModeOptions(paramForm.mode)"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="param-switch-grid">
          <ElFormItem label="强制配置" class="param-switch-field">
            <ElSwitch v-model="paramForm.required" :disabled="isEditingParamReadonly" />
          </ElFormItem>
          <ElFormItem label="不允许修改" class="param-switch-field">
            <ElSwitch v-model="paramForm.readonly" :disabled="isEditingParamReadonly" />
          </ElFormItem>
          <ElFormItem
            v-if="canConfigureRuntimeParam(paramForm)"
            label="运行时配置"
            class="param-switch-field"
          >
            <ElSwitch v-model="paramForm.runtimeConfig" :disabled="isEditingParamReadonly" />
          </ElFormItem>
          <ElFormItem label="占据整行" class="param-switch-field">
            <ElSwitch v-model="paramForm.config!.fullRow" :disabled="isEditingParamReadonly" />
          </ElFormItem>
        </div>

        <ElRow
          v-if="paramForm.paramType === 'hostGroupTargets'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="Jenkins 类型">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.renderMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
              >
                <ElOption label="JSON" value="json" />
                <ElOption label="JSON Base64" value="jsonBase64" />
                <ElOption label="Ansible" value="ansible" />
                <ElOption label="Ansible Base64" value="ansibleBase64" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <div v-if="paramForm.paramType === 'objectList'" class="compound-editor">
          <ElFormItem label="输出方式">
            <ElSegmented
              v-model="paramForm.config!.valueMode"
              :disabled="isEditingParamReadonly"
              :options="objectListValueModeOptions"
            />
          </ElFormItem>
          <div class="option-head">
            <span>对象字段</span>
            <ElButton
              text
              type="primary"
              :disabled="isEditingParamReadonly"
              @click="addCompoundField"
            >
              新增字段
            </ElButton>
          </div>
          <ElEmpty
            v-if="!(paramForm.config?.compoundFields || []).length"
            description="请至少配置一个字段"
          />
          <ElTable
            v-else
            class="compound-field-table"
            :data="paramForm.config?.compoundFields || []"
            border
            size="small"
          >
            <ElTableColumn type="index" label="#" width="56" align="center" />
            <ElTableColumn label="字段名称" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.name || '-' }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="字段编码" min-width="150" show-overflow-tooltip>
              <template #default="{ row }">
                {{ row.code || '-' }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="字段模式" width="110" align="center">
              <template #default="{ row }">
                <ElTag size="small" :type="modeTagType(row.mode)" effect="plain">
                  {{ modeName(row.mode) }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="字段类型" width="120" align="center">
              <template #default="{ row }">
                <ElTag size="small" effect="plain">{{ paramTypeName(row.paramType) }}</ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="必填" width="80" align="center">
              <template #default="{ row }">
                <ElTag size="small" :type="row.required ? 'danger' : 'info'" effect="plain">
                  {{ row.required ? '是' : '否' }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="120" fixed="right" align="center">
              <template #default="{ $index }">
                <ElButton
                  text
                  type="primary"
                  :disabled="isEditingParamReadonly"
                  @click="editCompoundField($index)"
                >
                  编辑
                </ElButton>
                <ElButton
                  text
                  type="danger"
                  :disabled="isEditingParamReadonly"
                  @click="removeCompoundField($index)"
                >
                  删除
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
          <template v-if="false">
            <div
              v-for="(field, fieldIndex) in paramForm.config?.compoundFields || []"
              :key="`${field.code || 'field'}-${fieldIndex}`"
              class="compound-field-card"
            >
              <div class="compound-field-head">
                <strong>字段 {{ fieldIndex + 1 }}</strong>
                <ElButton
                  text
                  type="danger"
                  :disabled="isEditingParamReadonly"
                  @click="removeCompoundField(fieldIndex)"
                >
                  删除
                </ElButton>
              </div>
              <div class="compound-field-grid">
                <ElFormItem label="字段名称">
                  <ElInput
                    v-model="field.name"
                    :disabled="isEditingParamReadonly"
                    placeholder="例如 仓库名称"
                  />
                </ElFormItem>
                <ElFormItem label="字段编码">
                  <ElInput
                    v-model="field.code"
                    :disabled="isEditingParamReadonly"
                    placeholder="例如 repoName"
                  />
                </ElFormItem>
                <ElFormItem label="字段模式">
                  <ElSegmented
                    v-model="field.mode"
                    :disabled="isEditingParamReadonly"
                    :options="compoundModeOptions"
                    @change="() => handleCompoundModeChange(field)"
                  />
                </ElFormItem>
                <ElFormItem label="字段类型">
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-model="field.paramType"
                    :disabled="isEditingParamReadonly"
                    style="width: 100%"
                    @change="handleCompoundTypeChange(field)"
                  >
                    <ElOption
                      v-for="item in availableCompoundParamTypeOptions(field.mode)"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </ElSelect>
                </ElFormItem>
                <ElFormItem label="默认值">
                  <ElSwitch
                    v-if="field.paramType === 'booleanParam'"
                    v-model="field.defaultValue"
                    :disabled="isEditingParamReadonly"
                    active-value="true"
                    inactive-value="false"
                  />
                  <ElInputNumber
                    v-else-if="field.paramType === 'number'"
                    :model-value="numberParamValue(field.defaultValue)"
                    :disabled="isEditingParamReadonly"
                    controls-position="right"
                    style="width: 100%"
                    placeholder="请输入数字"
                    @update:model-value="
                      (value: number | undefined) => setParamNumberDefault(field, value)
                    "
                  />
                  <ElInput
                    v-else-if="field.paramType === 'text'"
                    v-model="field.defaultValue"
                    :disabled="isEditingParamReadonly"
                    type="textarea"
                    :rows="4"
                    resize="vertical"
                    placeholder="请输入多行文本"
                  />
                  <ElInput
                    v-else
                    v-model="field.defaultValue"
                    :disabled="isEditingParamReadonly"
                    placeholder="请输入默认值"
                  />
                </ElFormItem>
              </div>
              <div class="param-switch-grid compact">
                <ElFormItem label="强制配置" class="param-switch-field">
                  <ElSwitch v-model="field.required" :disabled="isEditingParamReadonly" />
                </ElFormItem>
                <ElFormItem label="不允许修改" class="param-switch-field">
                  <ElSwitch v-model="field.readonly" :disabled="isEditingParamReadonly" />
                </ElFormItem>
              </div>
              <div v-if="needSelectList(field.paramType)" class="option-editor compact">
                <div class="option-head">
                  <span>选项列表</span>
                  <ElButton
                    text
                    type="primary"
                    :disabled="isEditingParamReadonly"
                    @click="addCompoundOption(field)"
                  >
                    新增选项
                  </ElButton>
                </div>
                <div
                  v-for="(option, optionIndex) in field.selectList || []"
                  :key="optionIndex"
                  class="option-row"
                >
                  <ElInput
                    v-model="option.label"
                    :disabled="isEditingParamReadonly"
                    placeholder="显示名称"
                  />
                  <ElInput
                    v-model="option.value"
                    :disabled="isEditingParamReadonly"
                    placeholder="实际值"
                  />
                  <ElButton
                    text
                    type="danger"
                    :disabled="isEditingParamReadonly"
                    @click="removeCompoundOption(field, optionIndex)"
                  >
                    删除
                  </ElButton>
                </div>
              </div>
              <ElRow
                v-if="field.paramType === 'channelGroupCode'"
                :gutter="14"
                class="param-special-grid"
              >
                <ElCol :span="12">
                  <ElFormItem label="渠道分组" required>
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.channelGroupCode"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择渠道分组"
                      @change="handleCompoundChannelGroupChange(field)"
                      @visible-change="handleOptionVisible"
                    >
                      <ElOption
                        v-for="item in channelGroupOptions"
                        :key="item.code"
                        :label="`${item.name}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="渠道类型" required>
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.channelTypeFilter"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择渠道类型"
                      @change="handleCompoundChannelTypeChange(field)"
                    >
                      <ElOption
                        v-for="item in channelTypeOptionsByGroup(field.config?.channelGroupCode)"
                        :key="item.value"
                        :label="item.label"
                        :value="item.value"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="映射字段" required>
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.mappingField"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      @change="handleChannelMappingFieldChange(field)"
                    >
                      <ElOption
                        v-for="item in channelVariableMappingOptions(field)"
                        :key="item.field"
                        :label="channelMappingFieldLabel(item)"
                        :value="item.field"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="usesSpecialChannelDependencyEditor(field)" :span="24">
                  <ElFormItem label="依赖来源" required class="channel-dependency-form-item">
                    <div class="channel-dependency-editor">
                      <ElSegmented
                        :model-value="specialChannelDependencyMode(field)"
                        :disabled="isEditingParamReadonly"
                        :options="specialChannelDependencyModeOptions(field)"
                        @change="
                          (value: string | number) =>
                            setSpecialChannelDependencyMode(field, String(value))
                        "
                      />
                      <div
                        v-if="specialChannelDependencyMode(field) === 'endpoint'"
                        class="channel-dependency-grid"
                      >
                        <ElSelect
                          popper-class="devops-step-template-dropdown"
                          v-model="field.config!.channelParamCode"
                          :disabled="isEditingParamReadonly"
                          filterable
                          clearable
                          style="width: 100%"
                          placeholder="请选择渠道实例来源"
                        >
                          <ElOption
                            v-for="item in compoundChannelVariableOptions(field)"
                            :key="item.code"
                            :label="`${item.name || item.code}（${item.code}）`"
                            :value="item.code"
                          />
                        </ElSelect>
                        <ElSelect
                          popper-class="devops-step-template-dropdown"
                          :model-value="specialProjectParamCode(field)"
                          :disabled="isEditingParamReadonly"
                          filterable
                          clearable
                          style="width: 100%"
                          :placeholder="specialProjectPlaceholder(field, '字段')"
                          @change="
                            (value) => setSpecialProjectParamCode(field, String(value || ''))
                          "
                        >
                          <ElOption
                            v-for="item in specialProjectOptions(field, compoundFields())"
                            :key="item.code"
                            :label="`${item.name || item.code}（${item.code}）`"
                            :value="item.code"
                          />
                        </ElSelect>
                        <ElSelect
                          popper-class="devops-step-template-dropdown"
                          v-if="needsSpecialImageDependency(field)"
                          :model-value="specialImageParamCode(field)"
                          :disabled="isEditingParamReadonly"
                          filterable
                          clearable
                          style="width: 100%"
                          placeholder="请选择镜像字段"
                          @change="(value) => setSpecialImageParamCode(field, String(value || ''))"
                        >
                          <ElOption
                            v-for="item in specialImageOptions(field, compoundFields())"
                            :key="item.code"
                            :label="`${item.name || item.code}（${item.code}）`"
                            :value="item.code"
                          />
                        </ElSelect>
                      </div>
                      <div v-else class="channel-dependency-grid single">
                        <ElSelect
                          popper-class="devops-step-template-dropdown"
                          :model-value="specialAddressParamCode(field)"
                          :disabled="isEditingParamReadonly"
                          filterable
                          clearable
                          style="width: 100%"
                          :placeholder="specialAddressPlaceholder(field, '字段')"
                          @change="
                            (value) => setSpecialAddressParamCode(field, String(value || ''))
                          "
                        >
                          <ElOption
                            v-for="item in specialAddressOptions(field, compoundFields())"
                            :key="item.code"
                            :label="`${item.name || item.code}（${item.code}）`"
                            :value="item.code"
                          />
                        </ElSelect>
                        <ElSelect
                          popper-class="devops-step-template-dropdown"
                          v-if="needsSpecialAddressImageDependency(field)"
                          :model-value="specialImageParamCode(field)"
                          :disabled="isEditingParamReadonly"
                          filterable
                          clearable
                          style="width: 100%"
                          placeholder="项目地址模式下请选择镜像字段"
                          @change="(value) => setSpecialImageParamCode(field, String(value || ''))"
                        >
                          <ElOption
                            v-for="item in specialImageOptions(field, compoundFields())"
                            :key="item.code"
                            :label="`${item.name || item.code}（${item.code}）`"
                            :value="item.code"
                          />
                        </ElSelect>
                      </div>
                    </div>
                  </ElFormItem>
                </ElCol>
                <ElCol
                  v-for="dep in channelParamDependencies(field)"
                  :key="`${field.code}-${dep.field}`"
                  :span="channelDependencyColumnSpan(field, dep.field)"
                  :class="{
                    'wide-label-field': channelDependencyColumnSpan(field, dep.field) === 24
                  }"
                >
                  <ElFormItem :label="dep.name || dep.field">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      :model-value="dependencyParamCode(field, dep.field)"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择依赖字段"
                      @change="(value) => setDependencyParamCode(field, dep.field, String(value))"
                    >
                      <ElOption
                        v-for="item in channelDependencyOptions(field, dep.field, compoundFields())"
                        :key="item.code"
                        :label="`${item.name || item.code}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="needChannelBindingParam(field)" :span="12">
                  <ElFormItem
                    :label="channelBindingLabel(field)"
                    :required="currentMappingFieldOption(field)?.allowManualInput !== true"
                  >
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.channelParamCode"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      :placeholder="channelBindingPlaceholder(field)"
                    >
                      <ElOption
                        v-for="item in compoundChannelVariableOptions(field)"
                        :key="item.code"
                        :label="`${item.name || item.code}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="needChannelProjectParam(field)" :span="12">
                  <ElFormItem :label="channelProjectLabel(field)" required>
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.projectParamCode"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择项目字段"
                    >
                      <ElOption
                        v-for="item in compoundChannelProjectOptions(field)"
                        :key="item.code"
                        :label="`${item.name || item.code}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
              </ElRow>
              <ElRow
                v-if="field.paramType === 'voucherModel'"
                :gutter="14"
                class="param-special-grid"
              >
                <ElCol :span="12">
                  <ElFormItem label="取值模式">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.credentialMode"
                      :disabled="isEditingParamReadonly"
                      style="width: 100%"
                      @change="handleCompoundCredentialModeChange(field)"
                    >
                      <ElOption label="凭证字段映射" value="field" />
                      <ElOption label="Jenkins credentialId" value="jenkinsCredentialId" />
                      <ElOption label="JSON 字符串" value="jsonData" />
                      <ElOption label="JSON Base64" value="jsonBase64" />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="凭证类型">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.voucherModel"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择凭证类型"
                      @change="handleCompoundVoucherModelChange(field)"
                      @visible-change="handleOptionVisible"
                    >
                      <ElOption
                        v-for="item in voucherCredentialTypeOptions(field.config?.credentialMode)"
                        :key="item.credentialType"
                        :label="`${item.name}（${item.credentialType}）`"
                        :value="item.credentialType"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="isCredentialFieldMode(field.config)" :span="12">
                  <ElFormItem label="映射字段">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.mappingField"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                    >
                      <ElOption
                        v-for="mappingField in voucherMappingFields(field.config?.voucherModel)"
                        :key="mappingField"
                        :label="`${credentialMappingFieldLabel(mappingField)}（${mappingField}）`"
                        :value="mappingField"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="24">
                  <div class="param-field-tip">
                    {{ credentialModeTip(field.config) }}
                  </div>
                </ElCol>
              </ElRow>
              <ElRow
                v-if="isDynamicCompoundPaasField(field) || isConfigCenterPaasParam(field)"
                :gutter="14"
                class="param-special-grid"
              >
                <ElCol v-if="field.paramType === 'configCenter'" :span="12">
                  <ElFormItem label="配置类型">
                    <ElTreeSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.configTypeId"
                      :disabled="isEditingParamReadonly"
                      :data="configTypeTree"
                      :props="{ label: 'name', value: 'id', children: 'children' }"
                      check-strictly
                      filterable
                      clearable
                      :render-after-expand="false"
                      style="width: 100%"
                      placeholder="请选择配置类型"
                      @change="handleConfigTypeChange(field)"
                      @visible-change="handleConfigTypeVisible"
                    >
                      <template #default="{ data }">
                        <span>{{ data.name }}（{{ data.code }}）</span>
                      </template>
                    </ElTreeSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="showDynamicChannelDependency(field)" :span="12">
                  <ElFormItem label="渠道字段">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.channelParamCode"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      placeholder="请选择渠道字段"
                      clearable
                    >
                      <ElOption
                        v-for="item in compoundChannelFieldOptions(field)"
                        :key="item.code"
                        :label="`${item.name || item.code}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="needProjectParam(field.paramType)" :span="12">
                  <ElFormItem :label="dependencyProjectLabel(field.paramType)">
                    <ElSelect
                      popper-class="devops-step-template-dropdown"
                      v-model="field.config!.projectParamCode"
                      :disabled="isEditingParamReadonly"
                      filterable
                      style="width: 100%"
                      :placeholder="dependencyProjectPlaceholder(field.paramType)"
                      @change="handleDynamicProjectDependencyChange(field)"
                    >
                      <ElOption
                        v-for="item in compoundProjectFieldOptions(field)"
                        :key="item.code"
                        :label="`${item.name || item.code}（${item.code}）`"
                        :value="item.code"
                      />
                    </ElSelect>
                  </ElFormItem>
                </ElCol>
                <ElCol v-if="isGitRefParamType(field.paramType)" :span="24">
                  <div class="param-field-tip compact">
                    项目来源可选择 Git 项目字段，也可选择 Git
                    仓库地址字段；选择仓库地址时会按地址解析项目路径。
                  </div>
                </ElCol>
              </ElRow>
              <ElFormItem label="描述" class="param-description">
                <ElInput
                  v-model="field.description"
                  :disabled="isEditingParamReadonly"
                  type="textarea"
                  :rows="2"
                  placeholder="请输入字段描述"
                />
              </ElFormItem>
            </div>
          </template>
        </div>

        <div v-if="needSelectList(paramForm.paramType)" class="option-editor">
          <div class="option-head">
            <span>选项列表</span>
            <ElButton
              text
              type="primary"
              :disabled="isEditingParamReadonly"
              @click="addOption(paramForm)"
            >
              新增选项
            </ElButton>
          </div>
          <div
            v-for="(option, optionIndex) in paramForm.selectList"
            :key="optionIndex"
            class="option-row"
          >
            <ElInput
              v-model="option.label"
              :disabled="isEditingParamReadonly"
              placeholder="显示名称"
            />
            <ElInput
              v-model="option.value"
              :disabled="isEditingParamReadonly"
              placeholder="实际值"
            />
            <ElButton
              text
              type="danger"
              :disabled="isEditingParamReadonly"
              @click="removeOption(paramForm, optionIndex)"
            >
              删除
            </ElButton>
          </div>
        </div>

        <ElRow
          v-if="paramForm.paramType === 'channelGroupCode'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="渠道分组" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.channelGroupCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道分组"
                @change="handleChannelGroupChange(paramForm)"
                @visible-change="handleOptionVisible"
              >
                <ElOption
                  v-for="item in channelGroupOptions"
                  :key="item.code"
                  :label="`${item.name}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="渠道类型" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.channelTypeFilter"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道类型"
                @change="handleChannelTypeChange(paramForm)"
              >
                <ElOption
                  v-for="item in channelTypeOptionsByGroup(paramForm.config?.channelGroupCode)"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="映射字段" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.mappingField"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                @change="handleChannelMappingFieldChange(paramForm)"
              >
                <ElOption
                  v-for="item in channelVariableMappingOptions(paramForm)"
                  :key="item.field"
                  :label="channelMappingFieldLabel(item)"
                  :value="item.field"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="usesSpecialChannelDependencyEditor(paramForm)" :span="24">
            <ElFormItem label="依赖来源" required class="channel-dependency-form-item">
              <div class="channel-dependency-editor">
                <ElSegmented
                  :model-value="specialChannelDependencyMode(paramForm)"
                  :disabled="isEditingParamReadonly"
                  :options="specialChannelDependencyModeOptions(paramForm)"
                  @change="
                    (value: string | number) =>
                      setSpecialChannelDependencyMode(paramForm, String(value))
                  "
                />
                <div
                  v-if="specialChannelDependencyMode(paramForm) === 'endpoint'"
                  class="channel-dependency-grid"
                >
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-model="paramForm.config!.channelParamCode"
                    :disabled="isEditingParamReadonly"
                    filterable
                    clearable
                    style="width: 100%"
                    placeholder="请选择渠道实例来源"
                  >
                    <ElOption
                      v-for="item in channelVariableOptions(paramForm)"
                      :key="item.code"
                      :label="`${item.name || item.code}（${item.code}）`"
                      :value="item.code"
                    />
                  </ElSelect>
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    :model-value="specialProjectParamCode(paramForm)"
                    :disabled="isEditingParamReadonly"
                    filterable
                    clearable
                    style="width: 100%"
                    :placeholder="specialProjectPlaceholder(paramForm, '参数')"
                    @change="(value) => setSpecialProjectParamCode(paramForm, String(value || ''))"
                  >
                    <ElOption
                      v-for="item in specialProjectOptions(paramForm, formData.params)"
                      :key="item.code"
                      :label="`${item.name || item.code}（${item.code}）`"
                      :value="item.code"
                    />
                  </ElSelect>
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-if="needsSpecialImageDependency(paramForm)"
                    :model-value="specialImageParamCode(paramForm)"
                    :disabled="isEditingParamReadonly"
                    filterable
                    clearable
                    style="width: 100%"
                    placeholder="请选择镜像参数"
                    @change="(value) => setSpecialImageParamCode(paramForm, String(value || ''))"
                  >
                    <ElOption
                      v-for="item in specialImageOptions(paramForm, formData.params)"
                      :key="item.code"
                      :label="`${item.name || item.code}（${item.code}）`"
                      :value="item.code"
                    />
                  </ElSelect>
                </div>
                <div v-else class="channel-dependency-grid single">
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    :model-value="specialAddressParamCode(paramForm)"
                    :disabled="isEditingParamReadonly"
                    filterable
                    clearable
                    style="width: 100%"
                    :placeholder="specialAddressPlaceholder(paramForm, '参数')"
                    @change="(value) => setSpecialAddressParamCode(paramForm, String(value || ''))"
                  >
                    <ElOption
                      v-for="item in specialAddressOptions(paramForm, formData.params)"
                      :key="item.code"
                      :label="`${item.name || item.code}（${item.code}）`"
                      :value="item.code"
                    />
                  </ElSelect>
                  <ElSelect
                    popper-class="devops-step-template-dropdown"
                    v-if="needsSpecialAddressImageDependency(paramForm)"
                    :model-value="specialImageParamCode(paramForm)"
                    :disabled="isEditingParamReadonly"
                    filterable
                    clearable
                    style="width: 100%"
                    placeholder="项目地址模式下请选择镜像参数"
                    @change="(value) => setSpecialImageParamCode(paramForm, String(value || ''))"
                  >
                    <ElOption
                      v-for="item in specialImageOptions(paramForm, formData.params)"
                      :key="item.code"
                      :label="`${item.name || item.code}（${item.code}）`"
                      :value="item.code"
                    />
                  </ElSelect>
                </div>
              </div>
            </ElFormItem>
          </ElCol>
          <ElCol
            v-for="dep in channelParamDependencies(paramForm)"
            :key="`${paramForm.code}-${dep.field}`"
            :span="channelDependencyColumnSpan(paramForm, dep.field)"
            :class="{
              'wide-label-field': channelDependencyColumnSpan(paramForm, dep.field) === 24
            }"
          >
            <ElFormItem :label="dep.name || dep.field">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                :model-value="dependencyParamCode(paramForm, dep.field)"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择依赖参数"
                @change="(value) => setDependencyParamCode(paramForm, dep.field, String(value))"
              >
                <ElOption
                  v-for="item in channelDependencyOptions(paramForm, dep.field, formData.params)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="needChannelBindingParam(paramForm)" :span="12">
            <ElFormItem
              :label="channelBindingLabel(paramForm)"
              :required="currentMappingFieldOption(paramForm)?.allowManualInput !== true"
            >
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.channelParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                :placeholder="channelBindingPlaceholder(paramForm)"
              >
                <ElOption
                  v-for="item in channelVariableOptions(paramForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="needChannelProjectParam(paramForm)" :span="12">
            <ElFormItem :label="channelProjectLabel(paramForm)" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.projectParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择项目参数"
              >
                <ElOption
                  v-for="item in channelProjectOptions(paramForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="isDeployConfigParamType(paramForm.paramType)"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="输出格式">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.valueMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
              >
                <ElOption
                  v-for="item in deployConfigValueModeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="paramForm.paramType === 'voucherModel'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="取值模式">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.credentialMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
                @change="handleCredentialModeChange(paramForm)"
              >
                <ElOption label="凭证字段映射" value="field" />
                <ElOption label="Jenkins credentialId" value="jenkinsCredentialId" />
                <ElOption label="JSON 字符串" value="jsonData" />
                <ElOption label="JSON Base64" value="jsonBase64" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="凭证类型">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.voucherModel"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择凭证类型"
                @change="handleVoucherModelChange(paramForm)"
                @visible-change="handleOptionVisible"
              >
                <ElOption
                  v-for="item in voucherCredentialTypeOptions(paramForm.config?.credentialMode)"
                  :key="item.credentialType"
                  :label="`${item.name}（${item.credentialType}）`"
                  :value="item.credentialType"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="isCredentialFieldMode(paramForm.config)" :span="12">
            <ElFormItem label="映射字段">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.mappingField"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
              >
                <ElOption
                  v-for="field in voucherMappingFields(paramForm.config?.voucherModel)"
                  :key="field"
                  :label="`${credentialMappingFieldLabel(field)}（${field}）`"
                  :value="field"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="24">
            <div class="param-field-tip">
              {{ credentialModeTip(paramForm.config) }}
            </div>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="paramForm.paramType === 'channelCredential'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="来源参数">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.credentialSourceParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道来源参数"
              >
                <ElOption
                  v-for="item in channelCredentialSourceOptions(paramForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="取值模式">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.credentialMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
                @change="handleChannelCredentialModeChange(paramForm)"
              >
                <ElOption label="凭证字段映射" value="field" />
                <ElOption label="JSON" value="jsonData" />
                <ElOption label="JSON Base64" value="jsonBase64" />
                <ElOption label="Jenkins credentialId" value="jenkinsCredentialId" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="isChannelCredentialFieldMode(paramForm.config)" :span="12">
            <ElFormItem label="映射字段">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.mappingField"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择凭证字段"
              >
                <ElOption
                  v-for="field in channelCredentialMappingFields"
                  :key="field"
                  :label="`${credentialMappingFieldLabel(field)}（${field}）`"
                  :value="field"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="24">
            <div class="param-field-tip">
              {{ channelCredentialModeTip(paramForm.config) }}
            </div>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="isDynamicPaasParam(paramForm) || isConfigCenterPaasParam(paramForm)"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol v-if="paramForm.paramType === 'configCenter'" :span="12">
            <ElFormItem label="配置类型">
              <ElTreeSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.configTypeId"
                :disabled="isEditingParamReadonly"
                :data="configTypeTree"
                :props="{ label: 'name', value: 'id', children: 'children' }"
                check-strictly
                filterable
                clearable
                :render-after-expand="false"
                style="width: 100%"
                placeholder="请选择配置类型"
                @change="handleConfigTypeChange(paramForm)"
                @visible-change="handleConfigTypeVisible"
              >
                <template #default="{ data }">
                  <span>{{ data.name }}（{{ data.code }}）</span>
                </template>
              </ElTreeSelect>
            </ElFormItem>
          </ElCol>
          <ElCol
            v-if="paramForm.paramType === 'configCenter' && paramForm.runtimeMode === 'params'"
            :span="12"
          >
            <ElFormItem label="Jenkins 类型">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.valueMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
                placeholder="请选择 Jenkins 类型"
              >
                <ElOption
                  v-for="item in configCenterValueModeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="showDynamicChannelDependency(paramForm)" :span="12">
            <ElFormItem label="渠道参数">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.channelParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道分组参数"
                clearable
              >
                <ElOption
                  v-for="item in channelParamOptions(paramForm)"
                  :key="item.code"
                  :label="`${item.name}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="needProjectParam(paramForm.paramType)" :span="12">
            <ElFormItem :label="dependencyProjectLabel(paramForm.paramType)">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="paramForm.config!.projectParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                :placeholder="dependencyProjectPlaceholder(paramForm.paramType)"
                @change="handleDynamicProjectDependencyChange(paramForm)"
              >
                <ElOption
                  v-for="item in projectDependencyParamOptions(paramForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="isGitRefParamType(paramForm.paramType)" :span="24">
            <div class="param-field-tip compact">
              项目来源可选择 Git 项目参数，也可选择 Git
              仓库地址参数；选择仓库地址时会按地址解析项目路径。
            </div>
          </ElCol>
        </ElRow>

        <ElFormItem label="描述" class="param-description">
          <ElInput
            v-model="paramForm.description"
            :disabled="isEditingParamReadonly"
            type="textarea"
            :rows="3"
            placeholder="请输入参数描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="paramDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :disabled="isEditingParamReadonly" @click="saveParamDialog">
            确定
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="compoundFieldDialogVisible"
      :title="compoundFieldDialogIndex === -1 ? '新增字段' : '编辑字段'"
      width="760px"
      append-to-body
      class="devops-polish-dialog compound-field-dialog"
      :close-on-click-modal="false"
    >
      <ElForm :model="compoundFieldForm" label-width="104px" class="compound-field-form">
        <div class="compound-field-grid">
          <ElFormItem label="字段名称" required>
            <ElInput
              v-model="compoundFieldForm.name"
              :disabled="isEditingParamReadonly"
              placeholder="例如 仓库名称"
            />
          </ElFormItem>
          <ElFormItem label="字段编码" required>
            <ElInput
              v-model="compoundFieldForm.code"
              :disabled="isEditingParamReadonly"
              placeholder="例如 repoName"
            />
          </ElFormItem>
          <ElFormItem label="字段模式">
            <ElSegmented
              v-model="compoundFieldForm.mode"
              :disabled="isEditingParamReadonly"
              :options="compoundModeOptions"
              @change="() => handleCompoundModeChange(compoundFieldForm)"
            />
          </ElFormItem>
          <ElFormItem label="字段类型">
            <ElSelect
              popper-class="devops-step-template-dropdown"
              v-model="compoundFieldForm.paramType"
              :disabled="isEditingParamReadonly"
              style="width: 100%"
              @change="handleCompoundTypeChange(compoundFieldForm)"
            >
              <ElOption
                v-for="item in availableCompoundParamTypeOptions(compoundFieldForm.mode)"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="默认值">
            <ElSwitch
              v-if="compoundFieldForm.paramType === 'booleanParam'"
              v-model="compoundFieldForm.defaultValue"
              :disabled="isEditingParamReadonly"
              active-value="true"
              inactive-value="false"
            />
            <ElInputNumber
              v-else-if="compoundFieldForm.paramType === 'number'"
              :model-value="numberParamValue(compoundFieldForm.defaultValue)"
              :disabled="isEditingParamReadonly"
              controls-position="right"
              style="width: 100%"
              placeholder="请输入数字"
              @update:model-value="
                (value: number | undefined) => setParamNumberDefault(compoundFieldForm, value)
              "
            />
            <ElInput
              v-else-if="compoundFieldForm.paramType === 'text'"
              v-model="compoundFieldForm.defaultValue"
              :disabled="isEditingParamReadonly"
              type="textarea"
              :rows="4"
              resize="vertical"
              placeholder="请输入多行文本"
            />
            <ElInput
              v-else
              v-model="compoundFieldForm.defaultValue"
              :disabled="isEditingParamReadonly"
              placeholder="请输入默认值"
            />
          </ElFormItem>
        </div>

        <div class="param-switch-grid compact">
          <ElFormItem label="强制配置" class="param-switch-field">
            <ElSwitch v-model="compoundFieldForm.required" :disabled="isEditingParamReadonly" />
          </ElFormItem>
          <ElFormItem label="不允许修改" class="param-switch-field">
            <ElSwitch v-model="compoundFieldForm.readonly" :disabled="isEditingParamReadonly" />
          </ElFormItem>
        </div>

        <div v-if="needSelectList(compoundFieldForm.paramType)" class="option-editor compact">
          <div class="option-head">
            <span>选项列表</span>
            <ElButton
              text
              type="primary"
              :disabled="isEditingParamReadonly"
              @click="addCompoundOption(compoundFieldForm)"
            >
              新增选项
            </ElButton>
          </div>
          <div
            v-for="(option, optionIndex) in compoundFieldForm.selectList || []"
            :key="optionIndex"
            class="option-row"
          >
            <ElInput
              v-model="option.label"
              :disabled="isEditingParamReadonly"
              placeholder="显示名称"
            />
            <ElInput
              v-model="option.value"
              :disabled="isEditingParamReadonly"
              placeholder="实际值"
            />
            <ElButton
              text
              type="danger"
              :disabled="isEditingParamReadonly"
              @click="removeCompoundOption(compoundFieldForm, optionIndex)"
            >
              删除
            </ElButton>
          </div>
        </div>

        <ElRow
          v-if="compoundFieldForm.paramType === 'channelGroupCode'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="渠道分组" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.channelGroupCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道分组"
                @change="handleCompoundChannelGroupChange(compoundFieldForm)"
                @visible-change="handleOptionVisible"
              >
                <ElOption
                  v-for="item in channelGroupOptions"
                  :key="item.code"
                  :label="`${item.name}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="渠道类型" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.channelTypeFilter"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道类型"
                @change="handleCompoundChannelTypeChange(compoundFieldForm)"
              >
                <ElOption
                  v-for="item in channelTypeOptionsByGroup(
                    compoundFieldForm.config?.channelGroupCode
                  )"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="映射字段" required>
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.mappingField"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                @change="handleChannelMappingFieldChange(compoundFieldForm)"
              >
                <ElOption
                  v-for="item in channelVariableMappingOptions(compoundFieldForm)"
                  :key="item.field"
                  :label="`${item.name}（${item.field}）`"
                  :value="item.field"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="compoundFieldForm.paramType === 'voucherModel'"
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol :span="12">
            <ElFormItem label="取值模式">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.credentialMode"
                :disabled="isEditingParamReadonly"
                style="width: 100%"
                @change="handleCompoundCredentialModeChange(compoundFieldForm)"
              >
                <ElOption label="凭证字段映射" value="field" />
                <ElOption label="Jenkins credentialId" value="jenkinsCredentialId" />
                <ElOption label="JSON 字符串" value="jsonData" />
                <ElOption label="JSON Base64" value="jsonBase64" />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="凭证类型">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.voucherModel"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择凭证类型"
                @change="handleCompoundVoucherModelChange(compoundFieldForm)"
                @visible-change="handleOptionVisible"
              >
                <ElOption
                  v-for="item in voucherCredentialTypeOptions(
                    compoundFieldForm.config?.credentialMode
                  )"
                  :key="item.credentialType"
                  :label="`${item.name}（${item.credentialType}）`"
                  :value="item.credentialType"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="isCredentialFieldMode(compoundFieldForm.config)" :span="12">
            <ElFormItem label="映射字段">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.mappingField"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
              >
                <ElOption
                  v-for="mappingField in voucherMappingFields(
                    compoundFieldForm.config?.voucherModel
                  )"
                  :key="mappingField"
                  :label="`${credentialMappingFieldLabel(mappingField)}（${mappingField}）`"
                  :value="mappingField"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="24">
            <div class="param-field-tip">
              {{ credentialModeTip(compoundFieldForm.config) }}
            </div>
          </ElCol>
        </ElRow>

        <ElRow
          v-if="
            isDynamicCompoundPaasField(compoundFieldForm) ||
            isConfigCenterPaasParam(compoundFieldForm)
          "
          :gutter="14"
          class="param-special-grid"
        >
          <ElCol v-if="compoundFieldForm.paramType === 'configCenter'" :span="12">
            <ElFormItem label="配置类型">
              <ElTreeSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.configTypeId"
                :disabled="isEditingParamReadonly"
                :data="configTypeTree"
                :props="{ label: 'name', value: 'id', children: 'children' }"
                check-strictly
                filterable
                clearable
                :render-after-expand="false"
                style="width: 100%"
                placeholder="请选择配置类型"
                @change="handleConfigTypeChange(compoundFieldForm)"
                @visible-change="handleConfigTypeVisible"
              >
                <template #default="{ data }">
                  <span>{{ data.name }}（{{ data.code }}）</span>
                </template>
              </ElTreeSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="showDynamicChannelDependency(compoundFieldForm)" :span="12">
            <ElFormItem label="渠道字段">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.channelParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                placeholder="请选择渠道字段"
                clearable
              >
                <ElOption
                  v-for="item in compoundChannelFieldOptions(compoundFieldForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol v-if="needProjectParam(compoundFieldForm.paramType)" :span="12">
            <ElFormItem :label="dependencyProjectLabel(compoundFieldForm.paramType)">
              <ElSelect
                popper-class="devops-step-template-dropdown"
                v-model="compoundFieldForm.config!.projectParamCode"
                :disabled="isEditingParamReadonly"
                filterable
                style="width: 100%"
                :placeholder="dependencyProjectPlaceholder(compoundFieldForm.paramType)"
                @change="handleDynamicProjectDependencyChange(compoundFieldForm)"
              >
                <ElOption
                  v-for="item in compoundProjectFieldOptions(compoundFieldForm)"
                  :key="item.code"
                  :label="`${item.name || item.code}（${item.code}）`"
                  :value="item.code"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="描述" class="param-description">
          <ElInput
            v-model="compoundFieldForm.description"
            :disabled="isEditingParamReadonly"
            type="textarea"
            :rows="2"
            placeholder="请输入字段描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="compoundFieldDialogVisible = false">取消</ElButton>
          <ElButton
            type="primary"
            :disabled="isEditingParamReadonly"
            @click="saveCompoundFieldDialog"
          >
            确定
          </ElButton>
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
  import type { FormInstance, FormRules } from 'element-plus'
  import { Plus } from '@element-plus/icons-vue'
  import { smartFormatTime } from '@/utils/format'
  import { treeDevopsConfigTypeApi, type DevopsConfigType } from '@/api/devops/project'
  import { listDevopsChannelTypeApi, type DevopsChannelType } from '@/api/devops/channel'
  import {
    createDevopsJenkinsStepApi,
    deleteDevopsJenkinsStepApi,
    listDevopsChannelGroupOptionsApi,
    listDevopsChannelMappingFieldOptionsApi,
    listDevopsCredentialTypeOptionsApi,
    listDevopsJenkinsStepApi,
    listDevopsStepCategoryApi,
    updateDevopsJenkinsStepApi,
    updateDevopsJenkinsStepStatusApi,
    type ChannelGroupOption,
    type ChannelMappingFieldDependency,
    type ChannelMappingFieldOption,
    type CredentialTypeOption,
    type DevopsStepCategory,
    type DevopsStepParam,
    type DevopsStepTemplate,
    type DevopsStepTemplateForm,
    type CompoundParamField
  } from '@/api/devops/pipeline'

  defineOptions({ name: 'DevopsJenkinsStep' })

  const JenkinsPipelineEditor = defineAsyncComponent(
    () => import('@/components/jenkins-pipeline-editor/index.vue')
  )

  const loading = ref(false)
  const submitLoading = ref(false)
  const dialogLoading = ref(false)
  const showSearchBar = ref(true)
  const searchRef = ref()
  const formRef = ref<FormInstance>()
  const paramFormRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const dialogStep = ref(0)
  const paramDialogVisible = ref(false)
  const paramDialogIndex = ref(-1)
  const paramDialogReadonly = ref(false)
  const compoundFieldDialogVisible = ref(false)
  const compoundFieldDialogIndex = ref(-1)
  const activeIconGroup = ref('工具平台')
  const currentId = ref('')
  const tableData = ref<DevopsStepTemplate[]>([])
  const categories = ref<DevopsStepCategory[]>([])
  const credentialTypeOptions = ref<CredentialTypeOption[]>([])
  const channelGroupOptions = ref<ChannelGroupOption[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const configTypeTree = ref<DevopsConfigType[]>([])
  const mappingFieldMap = ref<Record<string, ChannelMappingFieldOption[]>>({})
  const searchParams = ref<Record<string, any>>({})

  const searchForm = ref({
    name: undefined,
    code: undefined,
    categoryId: undefined,
    status: undefined
  })
  const pagination = ref({ current: 1, size: 15, total: 0 })

  const defaultStepsContent = `steps {
  script {
    echo 'hello'
  }
}`

  const defaultPostContent = `always {
  echo 'post 步骤'
}`

  const defaultParam = (): DevopsStepParam => ({
    name: '',
    code: '',
    defaultValue: '',
    paramType: 'string',
    mode: 'params',
    required: false,
    readonly: false,
    description: '',
    selectList: [],
    allowCustom: true,
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

  const defaultCompoundField = (): CompoundParamField => ({
    name: '',
    code: '',
    defaultValue: '',
    paramType: 'string',
    mode: 'env',
    required: false,
    readonly: false,
    description: '',
    selectList: [],
    allowCustom: true,
    runtimeMode: 'env',
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

  const defaultForm = (): DevopsStepTemplateForm => ({
    name: '',
    code: '',
    icon: 'simple-icons:jenkins',
    iconColor: '#d24939',
    description: '',
    type: 'task',
    categoryId: '',
    stageContent: defaultStepsContent,
    params: [],
    artifactConfig: {
      enabled: false,
      type: 'report',
      name: '',
      path: '',
      required: false,
      contentType: ''
    },
    status: 1
  })

  const formData = reactive<DevopsStepTemplateForm>(defaultForm())
  const paramForm = reactive<DevopsStepParam>(defaultParam())
  const compoundFieldForm = reactive<CompoundParamField>(defaultCompoundField())
  const pipelineEditorRef = ref<{ validateCode?: () => boolean }>()
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }],
    categoryId: [{ required: true, message: '请选择步骤分类', trigger: 'change' }],
    type: [{ required: true, message: '请选择步骤类型', trigger: 'change' }],
    stageContent: [{ required: true, message: '请输入步骤代码', trigger: 'blur' }]
  }
  const paramRules: FormRules = {
    name: [{ required: true, message: '请输入参数名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入参数编码', trigger: 'blur' }]
  }

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
        { label: 'JavaScript', icon: 'skill-icons:javascript', color: '#f7df1e' },
        { label: 'TypeScript', icon: 'skill-icons:typescript', color: '#3178c6' },
        { label: 'PHP', icon: 'skill-icons:php-light', color: '#777bb4' },
        { label: 'Rust', icon: 'skill-icons:rust', color: '#b7410e' },
        { label: 'Ruby', icon: 'skill-icons:ruby', color: '#cc342d' },
        { label: 'Shell', icon: 'skill-icons:bash-light', color: '#4eaa25' }
      ]
    },
    {
      name: '工具平台',
      items: [
        { label: 'Jenkins', icon: 'simple-icons:jenkins', color: '#d24939' },
        { label: 'Jenkins X', icon: 'simple-icons:jenkinsx', color: '#2563eb' },
        { label: 'Git', icon: 'simple-icons:git', color: '#f05032' },
        { label: 'GitLab', icon: 'simple-icons:gitlab', color: '#fc6d26' },
        { label: 'GitHub', icon: 'simple-icons:github', color: '#24292f' },
        { label: 'Gitee', icon: 'simple-icons:gitee', color: '#c71d23' },
        { label: 'Nexus', icon: 'simple-icons:sonatype', color: '#1b75bb' },
        { label: 'JFrog', icon: 'simple-icons:jfrog', color: '#41bf47' },
        { label: 'SonarQube', icon: 'simple-icons:sonarqube', color: '#4b9fd5' },
        { label: 'Argo CD', icon: 'simple-icons:argo', color: '#ef7b4d' },
        { label: 'Kubernetes', icon: 'simple-icons:kubernetes', color: '#326ce5' },
        { label: 'Docker', icon: 'simple-icons:docker', color: '#0ea5e9' },
        { label: 'Harbor', icon: 'simple-icons:harbor', color: '#60b932' },
        { label: 'Linux 主机', icon: 'skill-icons:linux-light', color: '#111827' }
      ]
    },
    {
      name: '构建工具',
      items: [
        { label: 'Maven', icon: 'simple-icons:apachemaven', color: '#c71a36' },
        { label: 'Ant', icon: 'simple-icons:apacheant', color: '#a81c7d' },
        { label: 'Gradle', icon: 'simple-icons:gradle', color: '#02303a' },
        { label: 'npm', icon: 'simple-icons:npm', color: '#cb3837' },
        { label: 'pnpm', icon: 'simple-icons:pnpm', color: '#f69220' },
        { label: 'Yarn', icon: 'simple-icons:yarn', color: '#2c8ebb' },
        { label: 'Vite', icon: 'simple-icons:vite', color: '#646cff' },
        { label: 'Webpack', icon: 'simple-icons:webpack', color: '#8dd6f9' },
        { label: 'Make', icon: 'simple-icons:gnu', color: '#a42e2b' },
        { label: 'CMake', icon: 'simple-icons:cmake', color: '#064f8c' },
        { label: 'Helm', icon: 'simple-icons:helm', color: '#0f1689' },
        { label: '通用工具', icon: 'ri:tools-line', color: '#64748b' }
      ]
    }
  ]
  const wizardSteps = [
    { title: '基本配置', desc: '名称、编码、分类、图标' },
    { title: '代码片段', desc: '维护 Jenkins steps' },
    { title: '参数管理', desc: '同步和维护变量' }
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
    { label: '字符串', value: 'string' },
    { label: '多行文本', value: 'text' },
    { label: '布尔', value: 'booleanParam' },
    { label: '数字', value: 'number' },
    { label: '密码', value: 'password' },
    { label: '文件上传', value: 'file' },
    { label: '单选', value: 'choice' },
    { label: '对象列表', value: 'objectList' },
    { label: '列表', value: 'list' },
    { label: '渠道变量', value: 'channelGroupCode' },
    { label: '平台凭证', value: 'voucherModel' },
    { label: '渠道凭证', value: 'channelCredential' },
    { label: '配置中心', value: 'configCenter' },
    { label: '主机组目标', value: 'hostGroupTargets' }
  ]
  const kubeNovaDeployConfigParamType = 'kubeNovaDeployConfig'
  const kubernetesDeployConfigParamType = 'kubernetesDeployConfig'
  const envParamTypes = ['string', 'text', 'booleanParam', 'number']
  const paramsParamTypes = [
    'string',
    'text',
    'booleanParam',
    'number',
    'password',
    'choice',
    'file'
  ]
  const channelPaasParamTypes = [
    'repositoryChannel',
    'nexusChannel',
    'harborChannel',
    'jfrogChannel'
  ]
  const dynamicPaasParamTypes = [
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
    'sonarProjectName',
    'sonarProjectKey'
  ]
  const compositeAddressParamTypes = [
    'gitRepositoryUrl',
    'nexusRepositoryUrl',
    'nexusArtifactUrl',
    'nexusArtifactVersionUrl',
    'harborProjectUrl',
    'harborImageUrl',
    'harborImageTagUrl',
    'registryRepositoryUrl',
    'registryImageUrl',
    'registryImageTagUrl',
    'jfrogRepositoryUrl',
    'jfrogArtifactUrl',
    'jfrogArtifactVersionUrl',
    'sonarAddress',
    'sonarProjectUrl',
    'sonarProjectKeyUrl',
    'hostConfig',
    'hostGroupConfig'
  ]
  const projectDynamicParamTypes = ['configCenter', 'mavenConfig', 'hostGroupTargets']
  const paasParamTypes = [
    'list',
    'objectList',
    'channelGroupCode',
    'voucherModel',
    'channelCredential',
    ...projectDynamicParamTypes,
    ...channelPaasParamTypes,
    ...dynamicPaasParamTypes,
    ...compositeAddressParamTypes,
    kubeNovaDeployConfigParamType,
    kubernetesDeployConfigParamType
  ]
  const visiblePaasParamTypes = [
    'list',
    'objectList',
    'channelGroupCode',
    'voucherModel',
    'channelCredential',
    ...projectDynamicParamTypes
  ]
  const channelCredentialSourceParamTypes = [
    ...channelPaasParamTypes,
    ...compositeAddressParamTypes
  ]
  const channelCredentialMappingFields = [
    'username',
    'password',
    'token',
    'secretText',
    'privateKey',
    'passphrase',
    'kubeconfig',
    'certificate'
  ]
  const compoundEnvParamTypes = ['string', 'text', 'booleanParam', 'number']
  const stepParamMap = ref<Record<string, { groupCode: string; channelTypeFilter: string }>>({})
  const channelGroupCode = (paramType: string): string => {
    const spec = stepParamMap.value[paramType]
    if (spec?.groupCode) return spec.groupCode
    // 回退：硬编码映射
    const fallback: Record<string, string> = {
      repositoryChannel: 'code_repo',
      nexusChannel: 'artifact_repo',
      harborChannel: 'image_registry',
      jfrogChannel: 'artifact_repo',
      kubeNovaDeployConfig: 'deploy_target',
      kubernetesDeployConfig: 'deploy_target'
    }
    return fallback[paramType] || ''
  }
  const channelTypeFilter = (paramType: string): string => {
    const spec = stepParamMap.value[paramType]
    if (spec?.channelTypeFilter) return spec.channelTypeFilter
    const fallback: Record<string, string> = {
      nexusChannel: 'nexus',
      harborChannel: 'harbor',
      jfrogChannel: 'jfrog',
      kubeNovaDeployConfig: 'kube-nova',
      kubernetesDeployConfig: 'kubernetes'
    }
    return fallback[paramType] || ''
  }
  const deployConfigParamTypes = [kubeNovaDeployConfigParamType, kubernetesDeployConfigParamType]
  const isDeployConfigParamType = (paramType?: string) =>
    deployConfigParamTypes.includes(paramType || '')
  const deployConfigChannelSpec = (paramType?: string) => ({
    groupCode: channelGroupCode(paramType || ''),
    mappingField:
      paramType === kubernetesDeployConfigParamType
        ? 'address.deploymentConfig'
        : 'address.deployConfig'
  })
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
      projectLabel: '项目来源'
    },
    gitTag: {
      channelParamType: 'repositoryChannel',
      projectParamType: 'gitProject',
      projectParamTypes: ['gitProject', 'gitRepositoryUrl'],
      projectLabel: '项目来源'
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
    sonarProjectName: { channelParamType: 'sonarAddress' },
    sonarProjectKey: { channelParamType: 'sonarAddress' },
    configCenter: { channelParamType: '' },
    mavenConfig: { channelParamType: '' },
    hostGroupTargets: { channelParamType: '' }
  }
  const configCenterValueModeOptions = [
    { label: 'str', value: 'string' },
    { label: 'text', value: 'text' },
    { label: 'Credentials secret file', value: 'credentialsSecretFile' },
    { label: 'base64', value: 'base64' }
  ]
  const objectListValueModeOptions = [
    { label: 'JSON 字符串', value: 'string' },
    { label: 'Base64(JSON)', value: 'base64' }
  ]
  const deployConfigValueModeOptions = [
    { label: '字符串', value: 'string' },
    { label: 'JSON', value: 'json' },
    { label: 'JSON Base64', value: 'jsonBase64' }
  ]
  const modeOptions = [
    { label: '执行变量', value: 'params' },
    { label: '环境变量', value: 'env' },
    { label: '平台变量', value: 'paas' }
  ]
  const compoundModeOptions = [
    { label: '环境变量', value: 'env' },
    { label: '平台变量', value: 'paas' }
  ]
  const runtimeModeOptions = (mode?: string) =>
    mode === 'paas'
      ? [
          { label: '执行变量', value: 'params' },
          { label: '静态变量', value: 'env' }
        ]
      : [
          { label: '执行变量', value: 'params' },
          { label: '环境变量', value: 'env' }
        ]
  const stepTypeOptions = [
    { label: '普通步骤', value: 'task' },
    { label: '人工审核', value: 'manual_approval' }
  ]
  const stepTypeName = (type?: string) => (type === 'manual_approval' ? '人工审核' : '普通步骤')
  const paramTypeName = (type?: string) =>
    type === 'mavenConfig'
      ? '配置中心'
      : paramTypeOptions.find((item) => item.value === type)?.label || '字符串'
  const modeName = (mode?: string) =>
    mode === 'env' ? '环境变量' : mode === 'paas' ? '平台变量' : '执行变量'
  const modeTagType = (mode?: string) =>
    mode === 'env' ? 'warning' : mode === 'paas' ? 'info' : 'success'

  const categoryOptions = computed(() =>
    categories.value.map((item) => ({ label: `${item.name}（${item.code}）`, value: item.id }))
  )
  const selectedCategory = computed(() =>
    categories.value.find((item) => item.id === formData.categoryId)
  )
  const isPostCategory = computed(() => selectedCategory.value?.code === 'post')
  const extractedParams = computed(() => extractParamsFromStage(formData.stageContent))
  const missingExtractedCount = computed(() => {
    const exists = new Set(formData.params.map((item) => item.code).filter(Boolean))
    return extractedParams.value.filter((item) => !exists.has(item.code)).length
  })
  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '步骤分类',
      key: 'categoryId',
      type: 'select',
      props: { clearable: true, filterable: true, options: categoryOptions.value }
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

  const fetchOptions = async () => {
    const [credentialRes, groupRes, typeRes] = await Promise.all([
      listDevopsCredentialTypeOptionsApi(),
      listDevopsChannelGroupOptionsApi(),
      treeDevopsConfigTypeApi({ status: 1 })
    ])
    credentialTypeOptions.value = credentialRes.items || []
    channelGroupOptions.value = groupRes.items || []
    configTypeTree.value = typeRes.items || []
    const channelTypeRes = await listDevopsChannelTypeApi({ page: 1, pageSize: 200, status: 1 })
    channelTypes.value = channelTypeRes.items || []
  }

  const mavenSettingsType = computed(() =>
    flattenConfigTypes(configTypeTree.value).find((item) => item.code === 'MavenSettings')
  )

  const fetchList = async () => {
    loading.value = true
    try {
      const res = await listDevopsJenkinsStepApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        engineType: 'jenkins',
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const refreshAll = async () => {
    await Promise.all([fetchCategories(), fetchList()])
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

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const isEditingParamReadonly = computed(() => paramDialogReadonly.value)

  const cloneParam = (param: DevopsStepParam): DevopsStepParam => {
    const isLegacyMavenConfig = param.paramType === 'mavenConfig'
    const config = {
      ...defaultParam().config,
      ...(param.config || {}),
      ...(isLegacyMavenConfig
        ? {
            configTypeId: mavenSettingsType.value?.id || param.config?.configTypeId || '',
            configTypeCode: 'MavenSettings',
            valueMode: normalizeConfigCenterValueMode(param.config?.valueMode)
          }
        : {}),
      compoundFields: (param.config?.compoundFields || []).map((field) => ({
        ...defaultCompoundField(),
        ...field,
        mode: ['env', 'paas'].includes(field.mode) ? field.mode : 'env',
        paramType: ['env', 'paas'].includes(field.mode) ? field.paramType : 'string',
        runtimeMode: 'env',
        selectList: (field.selectList || field.config?.options || []).map((option) => ({
          ...option
        })),
        config: { ...defaultParam().config, ...(field.config || {}), compoundFields: [] }
      }))
    }
    return {
      ...defaultParam(),
      ...param,
      paramType: isLegacyMavenConfig ? 'configCenter' : param.paramType,
      selectList: (param.selectList || param.config?.options || []).map((item) => ({ ...item })),
      config,
      runtimeMode: param.runtimeMode || param.mode,
      runtimeConfig:
        param.mode === 'params' || (param.mode === 'paas' && param.runtimeMode === 'params')
          ? param.runtimeConfig === true
          : false,
      sortOrder: Number(param.sortOrder || 0)
    }
  }

  const assignParamForm = (param: DevopsStepParam) => {
    Object.assign(paramForm, cloneParam(param))
  }

  const cloneCompoundField = (field?: CompoundParamField): CompoundParamField => ({
    ...defaultCompoundField(),
    ...(field || {}),
    mode: ['env', 'paas'].includes(field?.mode || '') ? field?.mode || 'env' : 'env',
    paramType: ['env', 'paas'].includes(field?.mode || '')
      ? field?.paramType || 'string'
      : 'string',
    runtimeMode: 'env',
    selectList: (field?.selectList || field?.config?.options || []).map((option) => ({
      ...option
    })),
    config: { ...defaultParam().config, ...(field?.config || {}), compoundFields: [] }
  })

  const assignCompoundFieldForm = (field?: CompoundParamField) => {
    Object.assign(compoundFieldForm, cloneCompoundField(field))
  }

  const numberParamValue = (value?: string | number) => {
    if (value === undefined || value === null || value === '') return undefined
    const parsed = Number(value)
    return Number.isNaN(parsed) ? undefined : parsed
  }

  const setParamNumberDefault = (
    target: Pick<DevopsStepParam | CompoundParamField, 'defaultValue'>,
    value?: number
  ) => {
    target.defaultValue = value === undefined || value === null ? '' : String(value)
  }

  const validateStepFields = async (fields: string[]) => {
    if (!formRef.value) return
    for (const field of fields) {
      await formRef.value.validateField(field)
    }
  }

  const validateStageContentSyntax = () => {
    return pipelineEditorRef.value?.validateCode?.() !== false
  }

  const handleNextStep = async () => {
    if (dialogStep.value === 0) {
      await validateStepFields(['name', 'code', 'categoryId', 'type'])
    }
    if (dialogStep.value === 1) {
      await validateStepFields(['stageContent'])
      if (!validateStageContentSyntax()) return false
    }
    dialogStep.value += 1
    return true
  }

  const handleStepClick = async (index: number) => {
    if (index <= dialogStep.value) {
      dialogStep.value = index
      return
    }
    while (dialogStep.value < index) {
      const passed = await handleNextStep()
      if (!passed) return
    }
  }

  const normalizeParamName = (code: string) =>
    code
      .toLowerCase()
      .split(/[_-]+/)
      .filter(Boolean)
      .map((word) => word.charAt(0).toUpperCase() + word.slice(1))
      .join(' ') || code

  const inferParamType = (code: string) => {
    if (/^(ENABLE|IS|HAS|NEED|PUSH|CLEAN)_/.test(code)) return 'booleanParam'
    if (/(PASSWORD|SECRET|TOKEN)$/.test(code)) return 'password'
    return 'string'
  }

  type ExtractedStageParam = Partial<DevopsStepParam> & { code: string; mode: string }

  const jenkinsSystemVariables = new Set([
    'BUILD_DISPLAY_NAME',
    'BUILD_ID',
    'BUILD_NUMBER',
    'BUILD_TAG',
    'BUILD_URL',
    'CHANGE_AUTHOR',
    'CHANGE_AUTHOR_DISPLAY_NAME',
    'CHANGE_AUTHOR_EMAIL',
    'CHANGE_BRANCH',
    'CHANGE_ID',
    'CHANGE_TARGET',
    'CHANGE_TITLE',
    'CHANGE_URL',
    'EXECUTOR_NUMBER',
    'GIT_AUTHOR_EMAIL',
    'GIT_AUTHOR_NAME',
    'GIT_BRANCH',
    'GIT_COMMIT',
    'GIT_COMMITTER_EMAIL',
    'GIT_COMMITTER_NAME',
    'GIT_PREVIOUS_COMMIT',
    'GIT_PREVIOUS_SUCCESSFUL_COMMIT',
    'GIT_URL',
    'HOME',
    'JOB_BASE_NAME',
    'JOB_DISPLAY_URL',
    'JOB_NAME',
    'JOB_URL',
    'NODE_LABELS',
    'NODE_NAME',
    'PATH',
    'PWD',
    'SHELL',
    'USER',
    'WORKSPACE',
    'WORKSPACE_TMP'
  ])

  const jenkinsParamTypeMap: Record<string, string> = {
    string: 'string',
    text: 'text',
    booleanParam: 'booleanParam',
    password: 'password',
    choice: 'choice',
    file: 'file',
    run: 'run'
  }

  const isJenkinsSystemVariable = (code: string) => {
    const upperCode = code.toUpperCase()
    return (
      jenkinsSystemVariables.has(upperCode) ||
      upperCode.startsWith('JENKINS_') ||
      upperCode.startsWith('HUDSON_')
    )
  }

  const isValidVariableCode = (code: string) => /^[A-Z_][A-Z0-9_]*$/.test(code)

  const isExtractableEnvCode = (code: string, localVars: Set<string>, assignedEnv: Set<string>) =>
    isValidVariableCode(code) &&
    !localVars.has(code) &&
    !assignedEnv.has(code) &&
    !isJenkinsSystemVariable(code) &&
    !code.endsWith('_PSW')

  const stripGroovyComments = (code: string) =>
    code.replace(/\/\*[\s\S]*?\*\//g, '').replace(/(^|[^:])\/\/.*$/gm, '$1')

  const findMatchingPair = (code: string, start: number, open: string, close: string) => {
    let depth = 0
    let quote = ''
    let escaped = false

    for (let index = start; index < code.length; index += 1) {
      const char = code[index]
      if (quote) {
        if (escaped) {
          escaped = false
          continue
        }
        if (char === '\\') {
          escaped = true
          continue
        }
        if (char === quote) quote = ''
        continue
      }

      if (char === '"' || char === "'") {
        quote = char
        continue
      }
      if (char === open) depth += 1
      if (char === close) {
        depth -= 1
        if (depth === 0) return index
      }
    }

    return -1
  }

  const stripInputCalls = (code: string) => {
    const pattern = /\binput\s*\(/g
    let result = ''
    let cursor = 0
    let match: RegExpExecArray | null

    while ((match = pattern.exec(code)) !== null) {
      const openIndex = code.indexOf('(', match.index)
      const closeIndex = findMatchingPair(code, openIndex, '(', ')')
      if (closeIndex === -1) continue

      result += code.slice(cursor, match.index)
      result += 'input()'
      cursor = closeIndex + 1
      pattern.lastIndex = cursor
    }

    return result + code.slice(cursor)
  }

  const extractNamedBlocks = (code: string, keyword: string) => {
    const blocks: string[] = []
    const pattern = new RegExp(`\\b${keyword}\\s*\\{`, 'g')
    let match: RegExpExecArray | null

    while ((match = pattern.exec(code)) !== null) {
      const openIndex = code.indexOf('{', match.index)
      const closeIndex = findMatchingPair(code, openIndex, '{', '}')
      if (closeIndex === -1) continue

      blocks.push(code.slice(openIndex + 1, closeIndex))
      pattern.lastIndex = closeIndex + 1
    }

    return blocks
  }

  const extractFunctionCalls = (code: string, names: string[]) => {
    const calls: Array<{ type: string; body: string }> = []
    const pattern = new RegExp(`\\b(${names.join('|')})\\s*\\(`, 'g')
    let match: RegExpExecArray | null

    while ((match = pattern.exec(code)) !== null) {
      const openIndex = code.indexOf('(', match.index)
      const closeIndex = findMatchingPair(code, openIndex, '(', ')')
      if (closeIndex === -1) continue

      calls.push({
        type: match[1],
        body: code.slice(openIndex + 1, closeIndex)
      })
      pattern.lastIndex = closeIndex + 1
    }

    return calls
  }

  const readNamedStringArg = (body: string, key: string) => {
    const match = body.match(new RegExp(`\\b${key}\\s*:\\s*(['"])([\\s\\S]*?)\\1`))
    return match?.[2]
  }

  const readDefaultValueArg = (body: string) => {
    const stringValue = readNamedStringArg(body, 'defaultValue')
    if (stringValue !== undefined) return stringValue

    const booleanValue = body.match(/\bdefaultValue\s*:\s*(true|false)\b/)
    return booleanValue?.[1] || ''
  }

  const readChoiceOptions = (body: string) => {
    const match = body.match(/\bchoices\s*:\s*\[([\s\S]*?)\]/)
    if (!match) return []

    return Array.from(match[1].matchAll(/['"]([^'"]+)['"]/g), (item) => ({
      label: item[1],
      value: item[1]
    }))
  }

  const collectLocalVariables = (code: string) => {
    const result = new Set<string>()
    const pattern =
      /\b(?:def|var|String|Integer|Long|Boolean|BigDecimal|List|Map|Set|int|long|double|boolean)\s+([A-Za-z_][A-Za-z0-9_]*)\b/g

    for (const match of code.matchAll(pattern)) {
      result.add(match[1])
    }

    return result
  }

  const collectAssignedEnvVariables = (code: string) => {
    const result = new Set<string>()
    const dotPattern = /\benv\.([A-Za-z_][A-Za-z0-9_]*)\s*=/g
    const bracketPattern = /\benv\[['"]([^'"]+)['"]\]\s*=/g

    for (const match of code.matchAll(dotPattern)) {
      result.add(match[1])
    }
    for (const match of code.matchAll(bracketPattern)) {
      result.add(match[1])
    }

    return result
  }

  const mergeExtractedParam = (
    map: Map<string, ExtractedStageParam>,
    item: ExtractedStageParam
  ) => {
    const code = item.code.trim()
    if (!code) return

    const existing = map.get(code)
    if (!existing) {
      map.set(code, item)
      return
    }

    map.set(code, {
      ...existing,
      ...item,
      mode: existing.mode === 'params' ? existing.mode : item.mode
    })
  }

  const extractParamsFromStage = (code: string) => {
    const cleanCode = stripInputCalls(stripGroovyComments(code || ''))
    const localVars = collectLocalVariables(cleanCode)
    const assignedEnv = collectAssignedEnvVariables(cleanCode)
    const result = new Map<string, ExtractedStageParam>()
    const addParam = (item: ExtractedStageParam) => {
      const paramCode = item.code.trim()
      if (!paramCode) return
      if (item.mode === 'env' && !isExtractableEnvCode(paramCode, localVars, assignedEnv)) return
      if (item.mode === 'params' && !/^[A-Za-z_][A-Za-z0-9_]*$/.test(paramCode)) return

      mergeExtractedParam(result, {
        name: normalizeParamName(paramCode),
        paramType: inferParamType(paramCode),
        description: `从 ${item.mode === 'env' ? '环境变量' : '执行变量'} ${paramCode} 自动提取`,
        ...item,
        code: paramCode
      })
    }

    extractNamedBlocks(cleanCode, 'parameters').forEach((block) => {
      extractFunctionCalls(block, Object.keys(jenkinsParamTypeMap)).forEach((call) => {
        const paramCode = readNamedStringArg(call.body, 'name')
        if (!paramCode) return

        addParam({
          code: paramCode,
          mode: 'params',
          paramType: jenkinsParamTypeMap[call.type] || inferParamType(paramCode),
          defaultValue:
            call.type === 'run'
              ? readNamedStringArg(call.body, 'projectName') ||
                readNamedStringArg(call.body, 'project') ||
                ''
              : readDefaultValueArg(call.body),
          description: readNamedStringArg(call.body, 'description') || `从 parameters 自动提取`,
          selectList: call.type === 'choice' ? readChoiceOptions(call.body) : []
        })
      })
    })

    extractNamedBlocks(cleanCode, 'environment').forEach((block) => {
      const envPattern = /^\s*([A-Z_][A-Z0-9_]*)\s*=\s*(.+?)\s*$/gm
      for (const match of block.matchAll(envPattern)) {
        const paramCode = match[1]
        const valueMatch = match[2].match(/^(['"])(.*?)\1$/)
        addParam({
          code: paramCode,
          mode: 'env',
          paramType: 'string',
          defaultValue: valueMatch?.[2] || '',
          description: '从 environment 自动提取'
        })
      }
    })

    const addReferenceParam = (mode: string, value: string) => {
      const paramCode = value.trim()
      addParam({
        code: paramCode,
        mode,
        paramType: inferParamType(paramCode),
        description: `从 ${mode === 'env' ? '环境变量' : '执行变量'} ${paramCode} 自动提取`
      })
    }

    const dotPattern = /\b(params|env)\.([A-Za-z_][A-Za-z0-9_]*)\b/g
    const bracketPattern = /\b(params|env)\[['"]([^'"]+)['"]\]/g
    const bareEnvPattern = /\$\{\s*([A-Za-z_][A-Za-z0-9_]*)\s*\}/g
    for (const match of cleanCode.matchAll(dotPattern)) {
      addReferenceParam(match[1] === 'env' ? 'env' : 'params', match[2])
    }
    for (const match of cleanCode.matchAll(bracketPattern)) {
      addReferenceParam(match[1] === 'env' ? 'env' : 'params', match[2])
    }
    for (const match of cleanCode.matchAll(bareEnvPattern)) {
      addReferenceParam('env', match[1])
    }

    return Array.from(result.values())
  }

  const syncParamsFromStage = () => {
    if (isPostCategory.value) {
      ElMessage.warning('post 步骤不提取参数')
      return
    }
    if (!validateStageContentSyntax()) return
    const exists = new Set(formData.params.map((item) => item.code).filter(Boolean))
    const nextParams = extractedParams.value.filter((item) => !exists.has(item.code))
    if (nextParams.length === 0) {
      ElMessage.info('没有发现新的参数')
      return
    }
    nextParams.forEach((item) => {
      formData.params.push({
        ...defaultParam(),
        name: normalizeParamName(item.code),
        code: item.code,
        mode: item.mode,
        paramType: item.paramType || inferParamType(item.code),
        defaultValue: item.defaultValue || '',
        description:
          item.description ||
          `从 ${item.mode === 'env' ? '环境变量' : '执行变量'} ${item.code} 自动提取`,
        selectList: item.selectList || [],
        runtimeMode: item.mode,
        config: { ...defaultParam().config, ...(item.config || {}) }
      })
    })
    ElMessage.success(`已同步 ${nextParams.length} 个参数`)
    editParam(formData.params.length - nextParams.length)
  }

  const openDialog = async (type: 'add' | 'edit', row?: DevopsStepTemplate) => {
    dialogType.value = type
    dialogStep.value = 0
    currentId.value = row?.id || ''
    Object.assign(formData, defaultForm(), row || {})
    activeIconGroup.value = getIconGroupName(formData.icon)
    formData.artifactConfig = {
      ...defaultForm().artifactConfig,
      ...(row?.artifactConfig || {})
    }
    formData.params = (row?.params || []).map((item) => ({
      ...defaultParam(),
      ...item,
      selectList: item.selectList || item.config?.options || [],
      config: { ...defaultParam().config, ...(item.config || {}) },
      runtimeMode: item.runtimeMode || item.mode,
      runtimeConfig:
        item.mode === 'params' || (item.mode === 'paas' && item.runtimeMode === 'params')
          ? item.runtimeConfig === true
          : false,
      sortOrder: item.sortOrder || 0
    }))
    normalizeParamSortOrders(false)
    dialogVisible.value = true
    dialogLoading.value = true
    try {
      await Promise.all([fetchCategories(), fetchOptions()])
    } finally {
      dialogLoading.value = false
    }
  }

  const resetForm = () => {
    dialogLoading.value = false
    formRef.value?.clearValidate()
    paramFormRef.value?.clearValidate()
    dialogStep.value = 0
    paramDialogVisible.value = false
    paramDialogIndex.value = -1
    paramDialogReadonly.value = false
    compoundFieldDialogVisible.value = false
    compoundFieldDialogIndex.value = -1
    Object.assign(formData, defaultForm())
    assignParamForm(defaultParam())
    assignCompoundFieldForm(defaultCompoundField())
  }

  const addParam = () => {
    paramDialogIndex.value = -1
    paramDialogReadonly.value = false
    assignParamForm({
      ...defaultParam(),
      sortOrder: formData.params.length + 1
    })
    paramDialogVisible.value = true
    requestAnimationFrame(() => paramFormRef.value?.clearValidate())
  }

  const editParam = async (index: number) => {
    const param = formData.params[index]
    if (!param) return
    paramDialogIndex.value = index
    paramDialogReadonly.value = false
    assignParamForm(param)
    if (param.mode === 'paas' && param.paramType === 'channelGroupCode') {
      await ensureMappingFields(param.config?.channelGroupCode, param.config?.channelTypeFilter)
      await nextTick()
    }
    paramDialogVisible.value = true
    requestAnimationFrame(() => paramFormRef.value?.clearValidate())
  }

  const normalizeParamOrderValue = (value: unknown, fallback: number) => {
    const order = Number(value || 0)
    if (!Number.isFinite(order) || order <= 0) return fallback
    return Math.max(1, Math.floor(order))
  }

  const normalizeParamSortOrders = (renumber = true) => {
    formData.params = formData.params
      .map((item, index) => ({
        item,
        index,
        order: normalizeParamOrderValue(item.sortOrder, index + 1)
      }))
      .sort((left, right) => left.order - right.order || left.index - right.index)
      .map(({ item }, index) => ({
        ...item,
        sortOrder: renumber ? index + 1 : item.sortOrder
      }))
  }

  const ensureStepDynamicParamDependencies = (param: DevopsStepParam) => {
    if (!isDynamicPaasParam(param)) return true
    const channelParamCode = String(param.config?.channelParamCode || '').trim()
    const requiredChannel = requiredChannelParamType(param.paramType)
    if (channelParamCode) {
      const channelParam = formData.params.find((item) => item.code === channelParamCode)
      if (!channelParam || channelParam.paramType !== requiredChannel) {
        ElMessage.warning(`平台参数「${param.name || param.code}」的渠道参数配置不正确`)
        return false
      }
    }
    if (needProjectParam(param.paramType)) {
      const projectParamCode = String(param.config?.projectParamCode || '').trim()
      if (projectParamCode) {
        const targetTypes = dynamicProjectParamTypes(param.paramType)
        const projectParam = formData.params.find((item) => item.code === projectParamCode)
        if (!projectParam || !targetTypes.includes(projectParam.paramType)) {
          ElMessage.warning(`平台参数「${param.name || param.code}」的项目参数配置不正确`)
          return false
        }
      }
    }
    return true
  }

  const ensureStepChannelVariableDependencies = (param: DevopsStepParam) => {
    if (param.mode !== 'paas' || param.paramType !== 'channelGroupCode') return true
    const name = `渠道参数「${param.name || param.code}」`
    if (!String(param.config?.channelGroupCode || '').trim()) {
      ElMessage.warning(`${name}必须选择渠道分组`)
      return false
    }
    if (needsChannelTypeFilter(param) && !String(param.config?.channelTypeFilter || '').trim()) {
      ElMessage.warning(`${name}必须选择渠道类型`)
      return false
    }
    if (!String(param.config?.mappingField || '').trim()) {
      ElMessage.warning(`${name}必须选择映射字段`)
      return false
    }
    if (isCodeRepoGitRefChannelVariable(param)) {
      if (codeRepoGitRefDependencyMode(param) === 'address') {
        const addressParamCode = codeRepoGitRefAddressParamCode(param)
        if (
          addressParamCode &&
          !codeRepoGitRefAddressOptions(param, formData.params).some(
            (item) => item.code === addressParamCode
          )
        ) {
          ElMessage.warning(`${name}的仓库地址参数不匹配`)
          return false
        }
        return true
      }
      const channelParamCode = String(param.config?.channelParamCode || '').trim()
      const projectParamCode = codeRepoGitRefProjectParamCode(param)
      if (
        channelParamCode &&
        !channelVariableOptions(param).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的渠道实例不匹配`)
        return false
      }
      if (
        projectParamCode &&
        !codeRepoGitRefProjectOptions(param, formData.params).some(
          (item) => item.code === projectParamCode
        )
      ) {
        ElMessage.warning(`${name}的仓库项目参数不匹配`)
        return false
      }
      return true
    }
    if (isImageRegistryDynamicChannelVariable(param)) {
      if (specialChannelDependencyMode(param) === 'address') {
        const addressParamCode = specialAddressParamCode(param)
        if (
          addressParamCode &&
          !specialAddressOptions(param, formData.params).some(
            (item) => item.code === addressParamCode
          )
        ) {
          ElMessage.warning(`${name}的镜像地址参数不匹配`)
          return false
        }
        if (needsSpecialAddressImageDependency(param)) {
          const imageParamCode = specialImageParamCode(param)
          if (
            imageParamCode &&
            !specialImageOptions(param, formData.params).some(
              (item) => item.code === imageParamCode
            )
          ) {
            ElMessage.warning(`${name}的镜像参数不匹配`)
            return false
          }
        }
        return true
      }
      const channelParamCode = String(param.config?.channelParamCode || '').trim()
      const projectParamCode = specialProjectParamCode(param)
      if (
        channelParamCode &&
        !channelVariableOptions(param).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的渠道实例不匹配`)
        return false
      }
      if (
        projectParamCode &&
        !specialProjectOptions(param, formData.params).some(
          (item) => item.code === projectParamCode
        )
      ) {
        ElMessage.warning(`${name}的镜像项目参数不匹配`)
        return false
      }
      if (needsSpecialImageDependency(param)) {
        const imageParamCode = specialImageParamCode(param)
        if (
          imageParamCode &&
          !specialImageOptions(param, formData.params).some((item) => item.code === imageParamCode)
        ) {
          ElMessage.warning(`${name}的镜像参数不匹配`)
          return false
        }
      }
      return true
    }
    if (
      needChannelBindingParam(param) &&
      currentMappingFieldOption(param)?.allowManualInput !== true
    ) {
      const channelParamCode = String(param.config?.channelParamCode || '').trim()
      if (!channelParamCode && !hasCodeRepoAddressDependency(param)) {
        ElMessage.warning(`${name}必须选择${channelBindingLabel(param)}`)
        return false
      }
      if (
        channelParamCode &&
        !channelVariableOptions(param).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的${channelBindingLabel(param)}不匹配`)
        return false
      }
    }
    for (const dep of channelParamDependencies(param)) {
      const paramCode = dependencyParamCode(param, dep.field)
      if (
        paramCode &&
        !channelDependencyOptions(param, dep.field, formData.params).some(
          (option) => option.code === paramCode
        )
      ) {
        ElMessage.warning(`${name}的${dep.name || dep.field}不匹配`)
        return false
      }
    }
    if (needChannelProjectParam(param)) {
      const projectParamCode = String(param.config?.projectParamCode || '').trim()
      if (!projectParamCode) {
        ElMessage.warning(`${name}必须选择${channelProjectLabel(param)}`)
        return false
      }
      if (!channelProjectOptions(param).some((item) => item.code === projectParamCode)) {
        ElMessage.warning(`${name}的${channelProjectLabel(param)}不匹配`)
        return false
      }
    }
    return true
  }

  const ensureCompoundDynamicDependencies = (field: CompoundParamField) => {
    if (!isDynamicCompoundPaasField(field)) return true
    const channelParamCode = String(field.config?.channelParamCode || '').trim()
    const requiredChannel = requiredChannelParamType(field.paramType)
    if (channelParamCode) {
      const channelParam = compoundFields().find((item) => item.code === channelParamCode)
      if (!channelParam || channelParam.paramType !== requiredChannel) {
        ElMessage.warning(`对象字段「${field.name || field.code}」的渠道来源配置不正确`)
        return false
      }
    }
    if (needProjectParam(field.paramType)) {
      const projectParamCode = String(field.config?.projectParamCode || '').trim()
      if (projectParamCode) {
        const targetTypes = dynamicProjectParamTypes(field.paramType)
        const projectParam = compoundFields().find((item) => item.code === projectParamCode)
        if (!projectParam || !targetTypes.includes(projectParam.paramType)) {
          ElMessage.warning(`对象字段「${field.name || field.code}」的项目字段配置不正确`)
          return false
        }
      }
    }
    return true
  }

  const ensureCompoundChannelVariableDependencies = (field: CompoundParamField) => {
    if (field.mode !== 'paas' || field.paramType !== 'channelGroupCode') return true
    const name = `对象字段「${field.name || field.code}」`
    if (!String(field.config?.channelGroupCode || '').trim()) {
      ElMessage.warning(`${name}必须选择渠道分组`)
      return false
    }
    if (needsChannelTypeFilter(field) && !String(field.config?.channelTypeFilter || '').trim()) {
      ElMessage.warning(`${name}必须选择渠道类型`)
      return false
    }
    if (!String(field.config?.mappingField || '').trim()) {
      ElMessage.warning(`${name}必须选择映射字段`)
      return false
    }
    if (isCodeRepoGitRefChannelVariable(field)) {
      if (codeRepoGitRefDependencyMode(field) === 'address') {
        const addressParamCode = codeRepoGitRefAddressParamCode(field)
        if (
          addressParamCode &&
          !codeRepoGitRefAddressOptions(field, compoundFields()).some(
            (item) => item.code === addressParamCode
          )
        ) {
          ElMessage.warning(`${name}的仓库地址字段不匹配`)
          return false
        }
        return true
      }
      const channelParamCode = String(field.config?.channelParamCode || '').trim()
      const projectParamCode = codeRepoGitRefProjectParamCode(field)
      if (
        channelParamCode &&
        !compoundChannelVariableOptions(field).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的渠道实例来源不匹配`)
        return false
      }
      if (
        projectParamCode &&
        !codeRepoGitRefProjectOptions(field, compoundFields()).some(
          (item) => item.code === projectParamCode
        )
      ) {
        ElMessage.warning(`${name}的仓库项目字段不匹配`)
        return false
      }
      return true
    }
    if (isImageRegistryDynamicChannelVariable(field)) {
      if (specialChannelDependencyMode(field) === 'address') {
        const addressParamCode = specialAddressParamCode(field)
        if (
          addressParamCode &&
          !specialAddressOptions(field, compoundFields()).some(
            (item) => item.code === addressParamCode
          )
        ) {
          ElMessage.warning(`${name}的镜像地址字段不匹配`)
          return false
        }
        if (needsSpecialAddressImageDependency(field)) {
          const imageParamCode = specialImageParamCode(field)
          if (
            imageParamCode &&
            !specialImageOptions(field, compoundFields()).some(
              (item) => item.code === imageParamCode
            )
          ) {
            ElMessage.warning(`${name}的镜像字段不匹配`)
            return false
          }
        }
        return true
      }
      const channelParamCode = String(field.config?.channelParamCode || '').trim()
      const projectParamCode = specialProjectParamCode(field)
      if (
        channelParamCode &&
        !compoundChannelVariableOptions(field).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的渠道实例来源不匹配`)
        return false
      }
      if (
        projectParamCode &&
        !specialProjectOptions(field, compoundFields()).some(
          (item) => item.code === projectParamCode
        )
      ) {
        ElMessage.warning(`${name}的镜像项目字段不匹配`)
        return false
      }
      if (needsSpecialImageDependency(field)) {
        const imageParamCode = specialImageParamCode(field)
        if (
          imageParamCode &&
          !specialImageOptions(field, compoundFields()).some((item) => item.code === imageParamCode)
        ) {
          ElMessage.warning(`${name}的镜像字段不匹配`)
          return false
        }
      }
      return true
    }
    if (
      needChannelBindingParam(field) &&
      currentMappingFieldOption(field)?.allowManualInput !== true
    ) {
      const channelParamCode = String(field.config?.channelParamCode || '').trim()
      if (!channelParamCode && !hasCodeRepoAddressDependency(field)) {
        ElMessage.warning(`${name}必须选择${channelBindingLabel(field)}`)
        return false
      }
      if (
        channelParamCode &&
        !compoundChannelVariableOptions(field).some((item) => item.code === channelParamCode)
      ) {
        ElMessage.warning(`${name}的${channelBindingLabel(field)}不匹配`)
        return false
      }
    }
    for (const dep of channelParamDependencies(field)) {
      const paramCode = dependencyParamCode(field, dep.field)
      if (
        paramCode &&
        !channelDependencyOptions(field, dep.field, compoundFields()).some(
          (option) => option.code === paramCode
        )
      ) {
        ElMessage.warning(`${name}的${dep.name || dep.field}不匹配`)
        return false
      }
    }
    if (needChannelProjectParam(field)) {
      const projectParamCode = String(field.config?.projectParamCode || '').trim()
      if (!projectParamCode) {
        ElMessage.warning(`${name}必须选择${channelProjectLabel(field)}`)
        return false
      }
      if (!compoundChannelProjectOptions(field).some((item) => item.code === projectParamCode)) {
        ElMessage.warning(`${name}的${channelProjectLabel(field)}不匹配`)
        return false
      }
    }
    return true
  }

  const saveParamDialog = async () => {
    await paramFormRef.value?.validate()
    const next = normalizeParamPayload(cloneParam(paramForm))
    if (!ensureStepChannelVariableDependencies(next)) return
    if (!ensureStepDynamicParamDependencies(next)) return
    if (next.paramType === 'objectList') {
      for (const field of next.config?.compoundFields || []) {
        if (!ensureCompoundChannelVariableDependencies(field)) return
        if (!ensureCompoundDynamicDependencies(field)) return
      }
    }
    const duplicateIndex = formData.params.findIndex(
      (item, index) => item.code === next.code && index !== paramDialogIndex.value
    )
    if (duplicateIndex >= 0) {
      ElMessage.warning(`参数编码重复：${next.code}`)
      return
    }
    if (paramDialogIndex.value === -1) {
      formData.params.push(next)
    } else {
      formData.params.splice(paramDialogIndex.value, 1, next)
    }
    normalizeParamSortOrders(false)
    paramDialogVisible.value = false
  }

  const removeParam = (index: number) => {
    formData.params.splice(index, 1)
    normalizeParamSortOrders(false)
  }

  const handleParamModeChange = (param: DevopsStepParam) => {
    if (param.mode === 'paas') {
      if (!['params', 'env'].includes(String(param.runtimeMode || ''))) {
        param.runtimeMode = 'env'
      }
    } else {
      param.runtimeMode = param.mode
    }
    if (!canConfigureRuntimeParam(param)) {
      param.runtimeConfig = false
    }
    if (param.paramType === 'channelCredential') {
      param.runtimeConfig = false
    }
    if (!param.config) param.config = { ...defaultParam().config }
    if (param.mode === 'env') {
      if (!envParamTypes.includes(param.paramType)) param.paramType = 'string'
    } else if (param.mode === 'params') {
      if (!paramsParamTypes.includes(param.paramType)) param.paramType = 'string'
    } else if (param.mode === 'paas') {
      if (!paasParamTypes.includes(param.paramType)) param.paramType = 'list'
    }
    handleParamTypeChange(param)
  }

  const canConfigureRuntimeParam = (param: DevopsStepParam) =>
    param.paramType !== 'channelCredential' &&
    (param.mode === 'params' || (param.mode === 'paas' && param.runtimeMode === 'params'))

  const handleRuntimeModeChange = (param: DevopsStepParam) => {
    if (!canConfigureRuntimeParam(param)) {
      param.runtimeConfig = false
    }
    if (param.paramType === 'channelCredential') {
      param.readonly = true
    }
    if (param.paramType === 'configCenter') {
      if (!param.config) param.config = { ...defaultParam().config }
      param.config.valueMode = normalizeConfigCenterValueMode(param.config.valueMode)
    }
  }

  const handleCompoundModeChange = (field: CompoundParamField) => {
    if (!['env', 'paas'].includes(field.mode)) field.mode = 'env'
    field.runtimeMode = 'env'
    if (!field.config) field.config = { ...defaultParam().config }
    if (field.mode === 'env') {
      if (!compoundEnvParamTypes.includes(field.paramType)) field.paramType = 'string'
    } else if (field.mode === 'paas') {
      if (!paasParamTypes.includes(field.paramType)) field.paramType = 'list'
    }
    handleCompoundTypeChange(field)
  }

  const needSelectList = (type: string) => ['choice', 'list'].includes(type)

  const addOption = (param: DevopsStepParam) => {
    if (!param.selectList) param.selectList = []
    param.selectList.push({ label: '', value: '' })
  }

  const removeOption = (param: DevopsStepParam, index: number) => {
    param.selectList?.splice(index, 1)
  }

  const addCompoundField = () => {
    openCompoundFieldDialog()
  }

  const compoundFieldItems = () => {
    if (!paramForm.config) paramForm.config = { ...defaultParam().config }
    if (!paramForm.config.compoundFields) paramForm.config.compoundFields = []
    return paramForm.config.compoundFields
  }

  const openCompoundFieldDialog = (index = -1) => {
    const fields = compoundFieldItems()
    compoundFieldDialogIndex.value = index
    assignCompoundFieldForm(index >= 0 ? fields[index] : defaultCompoundField())
    compoundFieldDialogVisible.value = true
  }

  const editCompoundField = (index: number) => {
    openCompoundFieldDialog(index)
  }

  const removeCompoundField = (index: number) => {
    paramForm.config?.compoundFields?.splice(index, 1)
  }

  const saveCompoundFieldDialog = () => {
    const fields = compoundFieldItems()
    const next = cloneCompoundField(compoundFieldForm)
    next.name = String(next.name || '').trim()
    next.code = String(next.code || '').trim()
    if (!next.name) {
      ElMessage.warning('字段名称不能为空')
      return
    }
    if (!next.code) {
      ElMessage.warning('字段编码不能为空')
      return
    }
    const duplicateIndex = fields.findIndex(
      (item, index) => item.code === next.code && index !== compoundFieldDialogIndex.value
    )
    if (duplicateIndex >= 0) {
      ElMessage.warning(`字段编码重复：${next.code}`)
      return
    }
    next.config = { ...defaultParam().config, ...(next.config || {}), compoundFields: [] }
    next.config.options = next.selectList || next.config.options || []
    if (compoundFieldDialogIndex.value === -1) {
      fields.push(next)
    } else {
      fields.splice(compoundFieldDialogIndex.value, 1, next)
    }
    compoundFieldDialogVisible.value = false
  }

  const addCompoundOption = (field: CompoundParamField) => {
    if (!field.selectList) field.selectList = []
    field.selectList.push({ label: '', value: '' })
  }

  const removeCompoundOption = (field: CompoundParamField, index: number) => {
    field.selectList?.splice(index, 1)
  }

  const handleParamTypeChange = (param: DevopsStepParam) => {
    if (!needSelectList(param.paramType)) param.selectList = []
    if (!param.config) param.config = { ...defaultParam().config }
    if (param.paramType === 'objectList' && param.mode === 'env') {
      param.mode = 'paas'
      param.runtimeMode = normalizeRuntimeMode(param.runtimeMode, param.mode)
    }
    if (param.paramType === 'configCenter') {
      param.config.valueMode = normalizeConfigCenterValueMode(param.config.valueMode)
    } else if (param.paramType === 'objectList') {
      param.config.valueMode = normalizeObjectListValueMode(param.config.valueMode)
    } else {
      param.config.configTypeId = ''
      param.config.configTypeCode = ''
      param.config.valueMode = 'string'
    }
    if (param.paramType !== 'channelGroupCode') {
      param.config.channelGroupCode = ''
      param.config.channelTypeFilter = ''
      param.config.channelBindingId = ''
    }
    if (param.paramType !== 'voucherModel' && param.paramType !== 'channelCredential') {
      param.config.voucherModel = ''
      param.config.credentialId = ''
      param.config.credentialMode = 'field'
    } else if (param.paramType === 'voucherModel') {
      param.config.credentialMode = normalizeCredentialMode(param.config.credentialMode)
    } else {
      normalizeChannelCredentialConfig(param)
    }
    if (param.paramType !== 'channelCredential') {
      param.config.credentialSourceParamCode = ''
    }
    if (!isDynamicPaasParam(param)) {
      param.config.channelParamCode = ''
      param.config.provider = ''
    }
    if (param.paramType === 'channelGroupCode' && needChannelBindingParam(param)) {
      param.config.provider = ''
    }
    if (isGitRepositoryURLDependency(param)) {
      param.config.channelParamCode = ''
      param.config.provider = ''
    }
    if (!needProjectParam(param.paramType) && !needChannelProjectParam(param)) {
      param.config.projectParamCode = ''
    }
    param.config.componentParamCode = ''
    if (isDeployConfigParamType(param.paramType)) {
      const spec = deployConfigChannelSpec(param.paramType)
      param.runtimeMode = 'params'
      param.runtimeConfig = true
      param.required = true
      param.config.fullRow = true
      param.config.channelGroupCode = spec.groupCode
      param.config.mappingField = spec.mappingField
    } else if (isChannelPaasParamType(param.paramType)) {
      param.config.channelGroupCode = channelGroupCode(param.paramType)
      param.config.channelTypeFilter = channelTypeFilter(param.paramType)
      param.config.mappingField = 'endpoint'
    } else if (param.paramType !== 'channelCredential') {
      param.config.mappingField = ''
    }
    if (param.paramType === 'hostGroupTargets') {
      param.config.renderMode = param.config.renderMode || 'json'
    }
  }

  const handleCompoundTypeChange = (field: CompoundParamField) => {
    field.runtimeMode = 'env'
    if (!needSelectList(field.paramType)) field.selectList = []
    if (!field.config) field.config = { ...defaultParam().config }
    field.config.configTypeId = ''
    field.config.configTypeCode = ''
    field.config.valueMode = 'string'
    field.config.mappingField = ''
    field.config.channelGroupCode = ''
    field.config.channelTypeFilter = ''
    field.config.voucherModel = ''
    field.config.credentialMode = 'field'
    field.config.channelParamCode = ''
    field.config.projectParamCode = ''
    field.config.componentParamCode = ''
    if (isChannelPaasParamType(field.paramType)) {
      field.config.channelGroupCode = channelGroupCode(field.paramType)
      field.config.channelTypeFilter = channelTypeFilter(field.paramType)
      field.config.mappingField = 'endpoint'
    }
  }

  const handleDynamicProjectDependencyChange = (item: DevopsStepParam | CompoundParamField) => {
    if (!item.config) item.config = { ...defaultParam().config }
    if (isGitRepositoryURLDependency(item)) {
      item.config.channelParamCode = ''
      item.config.provider = ''
    }
  }

  const flattenConfigTypes = (items: DevopsConfigType[]) =>
    items.reduce<DevopsConfigType[]>((acc, item) => {
      acc.push(item)
      if (item.children?.length) acc.push(...flattenConfigTypes(item.children))
      return acc
    }, [])

  const handleConfigTypeVisible = async (visible: boolean) => {
    if (!visible) return
    const res = await treeDevopsConfigTypeApi({ status: 1 })
    configTypeTree.value = res.items || []
  }

  const handleConfigTypeChange = (param: DevopsStepParam | CompoundParamField) => {
    if (!param.config) param.config = { ...defaultParam().config }
    const selected = flattenConfigTypes(configTypeTree.value).find(
      (item) => item.id === param.config?.configTypeId
    )
    param.config.configTypeCode = selected?.code || ''
  }

  const handleCompoundChannelGroupChange = async (field: CompoundParamField) => {
    if (!field.config) field.config = { ...defaultParam().config }
    await ensureMappingFields(field.config.channelGroupCode)
    field.config.channelTypeFilter = ''
    field.config.channelParamCode = ''
    field.config.projectParamCode = ''
    field.config.mappingField = ''
  }

  const handleCompoundChannelTypeChange = async (field: CompoundParamField) => {
    if (!field.config) field.config = { ...defaultParam().config }
    await ensureMappingFields(field.config.channelGroupCode, field.config.channelTypeFilter)
    field.config.channelParamCode = ''
    field.config.projectParamCode = ''
    field.config.mappingField = firstMappingFieldForType(
      field.config.channelGroupCode,
      field.config.channelTypeFilter
    )
  }

  const handleChannelMappingFieldChange = (item: DevopsStepParam | CompoundParamField) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = ''
    item.config.projectParamCode = ''
    item.config.componentParamCode = ''
    item.config.dependencyParamCodes = []
  }

  const handleCompoundVoucherModelChange = (field: CompoundParamField) => {
    if (!field.config) field.config = { ...defaultParam().config }
    normalizeVoucherConfig(field.config)
    const option = credentialTypeOptions.value.find(
      (item) => item.credentialType === field.config?.voucherModel
    )
    field.config.mappingField = isCredentialFieldMode(field.config)
      ? option?.defaultMappingField || option?.mappingFields?.[0] || ''
      : ''
  }

  const handleVoucherModelChange = (param: DevopsStepParam) => {
    if (!param.config) return
    normalizeVoucherConfig(param.config)
    const option = credentialTypeOptions.value.find(
      (item) => item.credentialType === param.config?.voucherModel
    )
    param.config.mappingField = isCredentialFieldMode(param.config)
      ? option?.defaultMappingField || option?.mappingFields?.[0] || ''
      : ''
  }

  type CredentialMode = NonNullable<NonNullable<DevopsStepParam['config']>['credentialMode']>

  const credentialModeValues: CredentialMode[] = [
    'field',
    'jsonData',
    'jsonBase64',
    'jenkinsCredentialId'
  ]
  const channelCredentialModeValues: CredentialMode[] = [
    'field',
    'jsonData',
    'jsonBase64',
    'jenkinsCredentialId'
  ]

  const normalizeCredentialMode = (mode?: string, mappingField?: string): CredentialMode => {
    const value = String(mode || '').trim()
    if (credentialModeValues.includes(value as CredentialMode)) return value as CredentialMode
    const legacyField = String(mappingField || '').trim()
    if (legacyField === 'credentialId') return 'jenkinsCredentialId'
    if (legacyField === 'jsonData' || legacyField === 'jsonBase64') return legacyField
    return 'field'
  }

  const normalizeChannelCredentialMode = (mode?: string, mappingField?: string): CredentialMode => {
    const value = String(mode || '').trim()
    if (channelCredentialModeValues.includes(value as CredentialMode)) {
      return value as CredentialMode
    }
    const legacyField = String(mappingField || '').trim()
    if (legacyField === 'credentialId') return 'jenkinsCredentialId'
    if (legacyField === 'jsonData' || legacyField === 'jsonBase64') return legacyField
    return 'field'
  }

  const isJenkinsCredentialMode = (config?: DevopsStepParam['config']) =>
    normalizeCredentialMode(config?.credentialMode, config?.mappingField) === 'jenkinsCredentialId'

  const isCredentialFieldMode = (config?: DevopsStepParam['config']) =>
    normalizeCredentialMode(config?.credentialMode, config?.mappingField) === 'field'

  const isChannelCredentialJenkinsMode = (config?: DevopsStepParam['config']) =>
    normalizeChannelCredentialMode(config?.credentialMode, config?.mappingField) ===
    'jenkinsCredentialId'

  const isChannelCredentialFieldMode = (config?: DevopsStepParam['config']) =>
    normalizeChannelCredentialMode(config?.credentialMode, config?.mappingField) === 'field'

  const jenkinsCredentialTypes = [
    'username_password',
    'token',
    'secret_text',
    'ssh_key',
    'kubeconfig',
    'json'
  ]

  const voucherCredentialTypeOptions = (credentialMode?: string) =>
    normalizeCredentialMode(credentialMode) === 'jenkinsCredentialId'
      ? credentialTypeOptions.value.filter((item) =>
          jenkinsCredentialTypes.includes(item.credentialType)
        )
      : credentialTypeOptions.value

  const normalizeVoucherConfig = (config: NonNullable<DevopsStepParam['config']>) => {
    config.credentialMode = normalizeCredentialMode(config.credentialMode, config.mappingField)
    if (config.credentialMode === 'jenkinsCredentialId') {
      if (!jenkinsCredentialTypes.includes(config.voucherModel || '')) {
        config.voucherModel =
          voucherCredentialTypeOptions(config.credentialMode)[0]?.credentialType || ''
      }
      config.mappingField = ''
    } else if (config.credentialMode === 'jsonData' || config.credentialMode === 'jsonBase64') {
      config.mappingField = ''
    } else if (
      config.mappingField === 'credentialId' ||
      config.mappingField === 'jsonData' ||
      config.mappingField === 'jsonBase64'
    ) {
      config.mappingField = ''
    }
  }

  const handleCredentialModeChange = (param: DevopsStepParam) => {
    if (!param.config) return
    normalizeVoucherConfig(param.config)
    handleVoucherModelChange(param)
  }

  const handleCompoundCredentialModeChange = (field: CompoundParamField) => {
    if (!field.config) return
    normalizeVoucherConfig(field.config)
    handleCompoundVoucherModelChange(field)
  }

  const normalizeChannelCredentialConfig = (param: DevopsStepParam) => {
    if (!param.config) param.config = { ...defaultParam().config }
    param.mode = 'paas'
    param.readonly = true
    param.runtimeConfig = false
    param.config.voucherModel = ''
    param.config.credentialId = ''
    param.config.credentialMode = normalizeChannelCredentialMode(
      param.config.credentialMode,
      param.config.mappingField
    )
    if (param.config.credentialMode !== 'field') {
      param.config.mappingField = ''
      return
    }
    if (!channelCredentialMappingFields.includes(param.config.mappingField || '')) {
      param.config.mappingField = 'username'
    }
  }

  const handleChannelCredentialModeChange = (param: DevopsStepParam) => {
    normalizeChannelCredentialConfig(param)
  }

  const credentialModeTip = (config?: DevopsStepParam['config']) =>
    isJenkinsCredentialMode(config)
      ? '参数编码是步骤片段里的接收字段；平台会同步所选凭证，并把生成的 Jenkins credentialId 写入该参数。Token 会同步为 Jenkins Secret text，Git 插件 credentialsId 建议使用用户名密码或 SSH Key。'
      : isCredentialFieldMode(config)
        ? '参数编码是步骤片段里的接收字段；映射字段决定把平台凭证里的哪个字段写入该参数。'
        : '参数编码是步骤片段里的接收字段；平台会把所选凭证按 JSON 或 JSON Base64 写入该参数。'

  const channelCredentialModeTip = (config?: DevopsStepParam['config']) =>
    isChannelCredentialJenkinsMode(config)
      ? '参数编码就是步骤片段里的变量名；平台会从来源渠道绑定凭证派生 Jenkins credentialId。'
      : isChannelCredentialFieldMode(config)
        ? '参数编码就是步骤片段里的变量名；平台会从来源渠道绑定凭证派生所选字段。'
        : '参数编码就是步骤片段里的变量名；平台会从来源渠道绑定凭证自动生成 JSON 或 JSON Base64。'

  const voucherMappingFields = (credentialType?: string) =>
    credentialTypeOptions.value.find((item) => item.credentialType === credentialType)
      ?.mappingFields || []

  const credentialMappingFieldLabel = (field: string) => {
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
    return labelMap[field] || field
  }

  const mappingCacheKey = (groupCode?: string, channelType?: string, channelTypeId?: string) =>
    [
      String(groupCode || '').trim(),
      String(channelType || '').trim(),
      String(channelTypeId || '').trim()
    ].join(':')

  const channelTypeIdByCode = (groupCode?: string, channelType?: string) =>
    channelTypes.value.find(
      (item) =>
        item.code === String(channelType || '').trim() &&
        item.groupCode === String(groupCode || '').trim()
    )?.id || ''

  const ensureMappingFields = async (groupCode?: string, channelType?: string) => {
    const channelTypeId = channelTypeIdByCode(groupCode, channelType)
    const key = mappingCacheKey(groupCode, channelType, channelTypeId)
    if (!groupCode || mappingFieldMap.value[key]) return
    const res = await listDevopsChannelMappingFieldOptionsApi(groupCode, channelType, channelTypeId)
    mappingFieldMap.value[key] = res.items || []
  }

  const mappingOptions = (groupCode?: string, channelType?: string) =>
    groupCode
      ? mappingFieldMap.value[
          mappingCacheKey(groupCode, channelType, channelTypeIdByCode(groupCode, channelType))
        ] || []
      : []

  const dedupeMappingFieldOptions = (items: ChannelMappingFieldOption[]) => {
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
  const deployTargetMappingFields = new Set([
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

  const filterChannelMappingOptions = (
    groupCode: string | undefined,
    items: ChannelMappingFieldOption[]
  ) => {
    const group = String(groupCode || '').trim()
    if (group === 'code_repo') {
      return items.filter((item) => codeRepoMappingFields.has(String(item.field || '').trim()))
    }
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
      return items.filter((item) => deployTargetMappingFields.has(String(item.field || '').trim()))
    }
    return items
  }

  const channelVariableMappingOptions = (item: DevopsStepParam | CompoundParamField) => {
    const options = mappingOptions(item.config?.channelGroupCode, item.config?.channelTypeFilter)
    const type = channelTypes.value.find(
      (typeItem) =>
        typeItem.code === String(item.config?.channelTypeFilter || '').trim() &&
        typeItem.groupCode === String(item.config?.channelGroupCode || '').trim()
    )
    const typeOptions = parseMappingFields(type?.mappingFields)
    return filterChannelMappingOptions(
      item.config?.channelGroupCode,
      dedupeMappingFieldOptions([...options, ...typeOptions])
    )
  }

  const channelTypeOptionsByGroup = (groupCode?: string) =>
    channelTypes.value
      .filter((item) => item.status === 1 && item.groupCode === String(groupCode || '').trim())
      .map((item) => ({ label: `${item.name}（${item.code}）`, value: item.code }))

  const parseMappingFields = (raw?: string): ChannelMappingFieldOption[] => {
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

  const currentMappingFieldOption = (item: DevopsStepParam | CompoundParamField) =>
    channelVariableMappingOptions(item).find(
      (option) => option.field === String(item.config?.mappingField || '').trim()
    )

  const codeRepoGitRefDependencyModeOptions = [
    { label: '渠道实例 + 仓库项目', value: 'endpoint' },
    { label: '仓库地址', value: 'address' }
  ]
  const imageRegistryDependencyModeOptions = [
    { label: '渠道实例 + 镜像项目', value: 'endpoint' },
    { label: '镜像地址', value: 'address' }
  ]
  const artifactRepoDependencyModeOptions = [
    { label: '渠道实例 + 仓库', value: 'endpoint' },
    { label: '制品地址', value: 'address' }
  ]

  const channelMappingFieldLabel = (item: ChannelMappingFieldOption) =>
    `${item.field === 'endpoint' ? '渠道实例' : item.name}（${item.field}）`

  const isCodeRepoGitRefChannelVariable = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode' &&
    String(item.config?.channelGroupCode || '').trim() === 'code_repo' &&
    ['dynamic.branch', 'dynamic.tag'].includes(String(item.config?.mappingField || '').trim())

  const isImageRegistryDynamicChannelVariable = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode' &&
    String(item.config?.channelGroupCode || '').trim() === 'image_registry' &&
    ['dynamic.image', 'dynamic.tag', 'dynamic.imageTag'].includes(
      String(item.config?.mappingField || '').trim()
    )

  const isArtifactRepoDynamicChannelVariable = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode' &&
    String(item.config?.channelGroupCode || '').trim() === 'artifact_repo' &&
    [
      'dynamic.artifact',
      'dynamic.artifactName',
      'dynamic.artifactVersion',
      'dynamic.artifactTag'
    ].includes(String(item.config?.mappingField || '').trim())

  const usesSpecialChannelDependencyEditor = (item: DevopsStepParam | CompoundParamField) =>
    isCodeRepoGitRefChannelVariable(item) ||
    isImageRegistryDynamicChannelVariable(item) ||
    isArtifactRepoDynamicChannelVariable(item)

  const specialChannelDependencyModeOptions = (item: DevopsStepParam | CompoundParamField) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? artifactRepoDependencyModeOptions
      : isImageRegistryDynamicChannelVariable(item)
        ? imageRegistryDependencyModeOptions
        : codeRepoGitRefDependencyModeOptions

  const specialChannelDependencyMode = (item: DevopsStepParam | CompoundParamField) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? hasArtifactRepoAddressDependency(item)
        ? 'address'
        : 'endpoint'
      : isImageRegistryDynamicChannelVariable(item)
        ? hasImageRegistryAddressDependency(item)
          ? 'address'
          : 'endpoint'
        : codeRepoGitRefDependencyMode(item)

  const codeRepoGitRefDependencyMode = (item: DevopsStepParam | CompoundParamField) =>
    hasCodeRepoAddressDependency(item) ? 'address' : 'endpoint'

  const codeRepoGitRefProjectParamCode = (item: DevopsStepParam | CompoundParamField) => {
    const value = dependencyParamCode(item, 'dynamic.project')
    return codeRepoGitRefDependencyMode(item) === 'endpoint' ? value : ''
  }

  const codeRepoGitRefAddressParamCode = (item: DevopsStepParam | CompoundParamField) => {
    const value =
      dependencyParamCode(item, 'address.projectUrl') ||
      dependencyParamCode(item, 'dynamic.project')
    return codeRepoGitRefDependencyMode(item) === 'address' ? value : ''
  }

  const imageRegistryAddressFields = [
    'address.projectUrl',
    'address.imageUrl',
    'address.imageTagUrl'
  ]
  const artifactRepoAddressFields = [
    'address.repositoryUrl',
    'address.artifactUrl',
    'address.artifactVersionUrl',
    'address.artifactTagUrl'
  ]

  const imageRegistryEndpointProjectFields = ['dynamic.project', 'dynamic.repository']

  const imageRegistryTagProjectAddressFields = [
    'address.projectUrl',
    'address.imageUrl',
    'address.imageTagUrl'
  ]

  const imageRegistryDependencyFields = (item: DevopsStepParam | CompoundParamField) => {
    const mappingField = String(item.config?.mappingField || '').trim()
    if (mappingField === 'dynamic.tag') {
      return ['dynamic.project', 'dynamic.image', ...imageRegistryAddressFields]
    }
    if (['dynamic.image', 'dynamic.imageTag'].includes(mappingField)) {
      return ['dynamic.project', ...imageRegistryAddressFields]
    }
    return imageRegistryAddressFields
  }

  const hasImageRegistryAddressDependency = (item: DevopsStepParam | CompoundParamField) =>
    (item.config?.dependencyParamCodes || []).some((dep) =>
      imageRegistryAddressFields.includes(String(dep.field || '').trim())
    )

  const hasArtifactRepoAddressDependency = (item: DevopsStepParam | CompoundParamField) =>
    (item.config?.dependencyParamCodes || []).some((dep) =>
      artifactRepoAddressFields.includes(String(dep.field || '').trim())
    )

  const specialProjectParamCode = (item: DevopsStepParam | CompoundParamField) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? dependencyParamCode(item, 'dynamic.repository')
      : isImageRegistryDynamicChannelVariable(item)
        ? dependencyParamCode(item, 'dynamic.project')
        : codeRepoGitRefProjectParamCode(item)

  const specialImageParamCode = (item: DevopsStepParam | CompoundParamField) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? dependencyParamCode(item, 'dynamic.artifactName') ||
        dependencyParamCode(item, 'dynamic.artifact')
      : dependencyParamCode(item, 'dynamic.image')

  const specialAddressParamCode = (item: DevopsStepParam | CompoundParamField) => {
    if (!isImageRegistryDynamicChannelVariable(item)) {
      if (!isArtifactRepoDynamicChannelVariable(item)) return codeRepoGitRefAddressParamCode(item)
      for (const field of artifactRepoAddressFields) {
        const value = dependencyParamCode(item, field)
        if (value) return value
      }
      return ''
    }
    for (const field of imageRegistryAddressFields) {
      const value = dependencyParamCode(item, field)
      if (value) return value
    }
    return ''
  }

  const specialProjectPlaceholder = (item: DevopsStepParam | CompoundParamField, suffix: string) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? `请选择制品仓库${suffix}`
      : isImageRegistryDynamicChannelVariable(item)
        ? `请选择镜像项目${suffix}`
        : `请选择仓库项目${suffix}`

  const specialAddressPlaceholder = (item: DevopsStepParam | CompoundParamField, suffix: string) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? `请选择制品地址${suffix}`
      : isImageRegistryDynamicChannelVariable(item)
        ? `请选择镜像地址${suffix}`
        : `请选择仓库地址${suffix}`

  const needsSpecialImageDependency = (item: DevopsStepParam | CompoundParamField) =>
    (isImageRegistryDynamicChannelVariable(item) &&
      String(item.config?.mappingField || '').trim() === 'dynamic.tag') ||
    (isArtifactRepoDynamicChannelVariable(item) &&
      ['dynamic.artifactVersion', 'dynamic.artifactTag'].includes(
        String(item.config?.mappingField || '').trim()
      ))

  const needsSpecialAddressImageDependency = (item: DevopsStepParam | CompoundParamField) =>
    needsSpecialImageDependency(item) &&
    specialChannelDependencyMode(item) === 'address' &&
    (isArtifactRepoDynamicChannelVariable(item)
      ? artifactRepoAddressFieldByParamCode(item, specialAddressParamCode(item)) ===
        'address.repositoryUrl'
      : imageRegistryAddressFieldByParamCode(item, specialAddressParamCode(item)) ===
        'address.projectUrl')

  const codeRepoGitRefProjectOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) => codeRepoGitRefOptions(item, candidates, 'dynamic.project')

  const codeRepoGitRefAddressOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) => codeRepoGitRefOptions(item, candidates, 'address.projectUrl')

  const specialProjectOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? imageRegistryOptions(item, candidates, ['dynamic.repository'])
      : isImageRegistryDynamicChannelVariable(item)
        ? imageRegistryOptions(item, candidates, imageRegistryEndpointProjectFields)
        : codeRepoGitRefProjectOptions(item, candidates)

  const imageRegistryImageOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) => imageRegistryOptions(item, candidates, ['dynamic.image'])

  const artifactRepoArtifactOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) => imageRegistryOptions(item, candidates, ['dynamic.artifactName', 'dynamic.artifact'])

  const specialImageOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? artifactRepoArtifactOptions(item, candidates)
      : imageRegistryImageOptions(item, candidates)

  const specialAddressOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) =>
    isArtifactRepoDynamicChannelVariable(item)
      ? imageRegistryOptions(item, candidates, artifactRepoAddressFields)
      : isImageRegistryDynamicChannelVariable(item)
        ? imageRegistryOptions(
            item,
            candidates,
            String(item.config?.mappingField || '').trim() === 'dynamic.tag'
              ? imageRegistryTagProjectAddressFields
              : imageRegistryAddressFields
          )
        : codeRepoGitRefAddressOptions(item, candidates)

  const imageRegistryOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>,
    fields: string[]
  ) => {
    const groupCode = String(item.config?.channelGroupCode || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    return candidates.filter((candidate) => {
      if (candidate.code === item.code || candidate.paramType !== 'channelGroupCode') return false
      const config = candidate.config || {}
      return (
        config.channelGroupCode === groupCode &&
        config.channelTypeFilter === channelType &&
        fields.includes(String(config.mappingField || '').trim())
      )
    })
  }

  const imageRegistryAddressFieldByParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (!paramCode) return ''
    const candidates =
      'sortOrder' in item ? formData.params : paramForm.config?.compoundFields || []
    const target = candidates.find((candidate) => candidate.code === paramCode)
    return String(target?.config?.mappingField || '').trim()
  }

  const artifactRepoAddressFieldByParamCode = imageRegistryAddressFieldByParamCode

  const codeRepoGitRefOptions = (
    item: DevopsStepParam | CompoundParamField,
    candidates: Array<DevopsStepParam | CompoundParamField>,
    field: string
  ) => {
    const groupCode = String(item.config?.channelGroupCode || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    return candidates.filter((candidate) => {
      if (candidate.code === item.code || candidate.paramType !== 'channelGroupCode') return false
      const config = candidate.config || {}
      return (
        config.channelGroupCode === groupCode &&
        config.channelTypeFilter === channelType &&
        String(config.mappingField || '').trim() === field
      )
    })
  }

  const setCodeRepoGitRefDependencyMode = (
    item: DevopsStepParam | CompoundParamField,
    mode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = mode === 'address' ? '' : item.config.channelParamCode || ''
    item.config.projectParamCode = ''
    item.config.dependencyParamCodes = (item.config.dependencyParamCodes || []).filter(
      (dep) => !['dynamic.project', 'address.projectUrl'].includes(dep.field)
    )
    if (mode === 'address') {
      item.config.dependencyParamCodes.push({ field: 'address.projectUrl', paramCode: '' })
    }
  }

  const setCodeRepoGitRefProjectParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.projectParamCode = paramCode
    setDependencyParamCode(item, 'dynamic.project', paramCode)
  }

  const setCodeRepoGitRefAddressParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = ''
    item.config.projectParamCode = paramCode
    const next = (item.config.dependencyParamCodes || []).filter(
      (dep) => !['dynamic.project', 'address.projectUrl'].includes(dep.field)
    )
    next.push({ field: 'address.projectUrl', paramCode })
    item.config.dependencyParamCodes = next
  }

  const setSpecialChannelDependencyMode = (
    item: DevopsStepParam | CompoundParamField,
    mode: string
  ) => {
    if (isArtifactRepoDynamicChannelVariable(item)) {
      setArtifactRepoDependencyMode(item, mode)
      return
    }
    if (isImageRegistryDynamicChannelVariable(item)) {
      setImageRegistryDependencyMode(item, mode)
      return
    }
    setCodeRepoGitRefDependencyMode(item, mode)
  }

  const setImageRegistryDependencyMode = (
    item: DevopsStepParam | CompoundParamField,
    mode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = mode === 'address' ? '' : item.config.channelParamCode || ''
    item.config.projectParamCode = ''
    item.config.componentParamCode = ''
    item.config.dependencyParamCodes = (item.config.dependencyParamCodes || []).filter(
      (dep) => !imageRegistryDependencyFields(item).includes(dep.field)
    )
    if (mode === 'address') {
      item.config.dependencyParamCodes.push({ field: 'address.imageUrl', paramCode: '' })
    }
  }

  const setArtifactRepoDependencyMode = (
    item: DevopsStepParam | CompoundParamField,
    mode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = mode === 'address' ? '' : item.config.channelParamCode || ''
    item.config.projectParamCode = ''
    item.config.componentParamCode = ''
    item.config.dependencyParamCodes = (item.config.dependencyParamCodes || []).filter(
      (dep) =>
        ![
          'dynamic.repository',
          'dynamic.artifact',
          'dynamic.artifactName',
          ...artifactRepoAddressFields
        ].includes(dep.field)
    )
    if (mode === 'address') {
      item.config.dependencyParamCodes.push({ field: 'address.artifactUrl', paramCode: '' })
    }
  }

  const setSpecialProjectParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (isArtifactRepoDynamicChannelVariable(item)) {
      if (!item.config) item.config = { ...defaultParam().config }
      item.config.projectParamCode = paramCode
      setDependencyParamCode(item, 'dynamic.repository', paramCode)
      return
    }
    if (isImageRegistryDynamicChannelVariable(item)) {
      if (!item.config) item.config = { ...defaultParam().config }
      item.config.projectParamCode = paramCode
      setDependencyParamCode(item, 'dynamic.project', paramCode)
      return
    }
    setCodeRepoGitRefProjectParamCode(item, paramCode)
  }

  const setSpecialImageParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.componentParamCode = paramCode
    if (isArtifactRepoDynamicChannelVariable(item)) {
      setDependencyParamCode(item, 'dynamic.artifactName', paramCode)
      return
    }
    setDependencyParamCode(item, 'dynamic.image', paramCode)
  }

  const setSpecialAddressParamCode = (
    item: DevopsStepParam | CompoundParamField,
    paramCode: string
  ) => {
    if (isArtifactRepoDynamicChannelVariable(item)) {
      if (!item.config) item.config = { ...defaultParam().config }
      item.config.channelParamCode = ''
      item.config.projectParamCode = paramCode
      const targetField =
        artifactRepoAddressFieldByParamCode(item, paramCode) || 'address.artifactUrl'
      const next = (item.config.dependencyParamCodes || []).filter(
        (dep) => !artifactRepoAddressFields.includes(dep.field)
      )
      next.push({ field: targetField, paramCode })
      item.config.dependencyParamCodes = next
      if (targetField !== 'address.repositoryUrl') {
        setDependencyParamCode(item, 'dynamic.artifactName', '')
      }
      return
    }
    if (!isImageRegistryDynamicChannelVariable(item)) {
      setCodeRepoGitRefAddressParamCode(item, paramCode)
      return
    }
    if (!item.config) item.config = { ...defaultParam().config }
    item.config.channelParamCode = ''
    item.config.projectParamCode = paramCode
    const targetField =
      imageRegistryAddressFieldByParamCode(item, paramCode) ||
      (String(item.config.mappingField || '').trim() === 'dynamic.tag'
        ? 'address.imageUrl'
        : 'address.projectUrl')
    const next = (item.config.dependencyParamCodes || []).filter(
      (dep) => !imageRegistryAddressFields.includes(dep.field)
    )
    next.push({ field: targetField, paramCode })
    item.config.dependencyParamCodes = next
    if (targetField !== 'address.projectUrl') {
      setDependencyParamCode(item, 'dynamic.image', '')
    }
  }

  const channelParamDependencies = (
    item: DevopsStepParam | CompoundParamField
  ): ChannelMappingFieldDependency[] => {
    if (usesSpecialChannelDependencyEditor(item)) return []
    const option = currentMappingFieldOption(item)
    return (option?.dependencies || [])
      .filter((dep) => dep.source === 'param' && dep.field)
      .map((dep) => ({
        ...dep,
        name: codeRepoGitRefProjectDependency(item, dep.field) ? '仓库项目或仓库地址' : dep.name,
        required: option?.allowManualInput === true ? false : dep.required
      }))
  }

  const codeRepoGitRefProjectDependency = (
    item: DevopsStepParam | CompoundParamField,
    field: string
  ) =>
    String(item.config?.channelGroupCode || '').trim() === 'code_repo' &&
    ['dynamic.branch', 'dynamic.tag'].includes(String(item.config?.mappingField || '').trim()) &&
    field === 'dynamic.project'

  const channelDependencyColumnSpan = (
    item: DevopsStepParam | CompoundParamField,
    field: string
  ) => (codeRepoGitRefProjectDependency(item, field) ? 24 : 12)

  const hasCodeRepoAddressDependency = (item: DevopsStepParam | CompoundParamField) =>
    (item.config?.dependencyParamCodes || []).some((dep) => {
      if (dep.field !== 'address.projectUrl') return false
      const paramCode = String(dep.paramCode || '').trim()
      if (!paramCode) return true
      const candidates =
        'sortOrder' in item ? formData.params : paramForm.config?.compoundFields || []
      return candidates.some(
        (candidate) =>
          candidate.code === paramCode &&
          candidate.paramType === 'channelGroupCode' &&
          String(candidate.config?.channelGroupCode || '').trim() ===
            String(item.config?.channelGroupCode || '').trim() &&
          String(candidate.config?.channelTypeFilter || '').trim() ===
            String(item.config?.channelTypeFilter || '').trim() &&
          String(candidate.config?.mappingField || '').trim() === 'address.projectUrl'
      )
    })

  const useCodeRepoAddressDependency = (item: DevopsStepParam | CompoundParamField) => {
    const mappingField = String(item.config?.mappingField || '').trim()
    return (
      String(item.config?.channelGroupCode || '').trim() === 'code_repo' &&
      ['dynamic.branch', 'dynamic.tag'].includes(mappingField) &&
      hasCodeRepoAddressDependency(item)
    )
  }

  const useImageRegistryAddressDependency = (item: DevopsStepParam | CompoundParamField) =>
    (isImageRegistryDynamicChannelVariable(item) && hasImageRegistryAddressDependency(item)) ||
    (isArtifactRepoDynamicChannelVariable(item) && hasArtifactRepoAddressDependency(item))

  const useSpecialAddressDependency = (item: DevopsStepParam | CompoundParamField) =>
    useCodeRepoAddressDependency(item) || useImageRegistryAddressDependency(item)

  const dependencyParamCode = (item: DevopsStepParam | CompoundParamField, field: string) =>
    (item.config?.dependencyParamCodes || []).find((dep) => dep.field === field)?.paramCode || ''

  const setDependencyParamCode = (
    item: DevopsStepParam | CompoundParamField,
    field: string,
    paramCode: string
  ) => {
    if (!item.config) item.config = { ...defaultParam().config }
    const next = (item.config.dependencyParamCodes || []).filter((dep) => dep.field !== field)
    if (paramCode) next.push({ field, paramCode })
    item.config.dependencyParamCodes = next
    if (
      field === 'dynamic.project' ||
      field === 'dynamic.repository' ||
      field.startsWith('address.')
    ) {
      item.config.projectParamCode = paramCode
    }
    if (codeRepoGitRefProjectDependency(item, field) && hasCodeRepoAddressDependency(item)) {
      item.config.channelParamCode = ''
    }
    if (
      field === 'dynamic.image' ||
      field === 'dynamic.artifact' ||
      field === 'dynamic.artifactName'
    ) {
      item.config.componentParamCode = paramCode
    }
  }

  const channelDependencyOptions = (
    item: DevopsStepParam | CompoundParamField,
    field: string,
    candidates: Array<DevopsStepParam | CompoundParamField>
  ) => {
    const groupCode = String(item.config?.channelGroupCode || '').trim()
    const channelType = String(item.config?.channelTypeFilter || '').trim()
    const targetFields =
      isArtifactRepoDynamicChannelVariable(item) && field === 'dynamic.repository'
        ? ['dynamic.repository', ...artifactRepoAddressFields]
        : isImageRegistryDynamicChannelVariable(item) && field === 'dynamic.project'
          ? ['dynamic.project', ...imageRegistryAddressFields]
          : codeRepoGitRefProjectDependency(item, field)
            ? ['dynamic.project', 'address.projectUrl']
            : [field]
    return candidates.filter((candidate) => {
      if (candidate.code === item.code || candidate.paramType !== 'channelGroupCode') return false
      const config = candidate.config || {}
      return (
        config.channelGroupCode === groupCode &&
        config.channelTypeFilter === channelType &&
        targetFields.includes(String(config.mappingField || '').trim())
      )
    })
  }

  const firstMappingFieldForType = (groupCode?: string, channelType?: string) => {
    const options = channelVariableMappingOptions({
      ...defaultParam(),
      paramType: 'channelGroupCode',
      config: {
        ...defaultParam().config,
        channelGroupCode: groupCode,
        channelTypeFilter: channelType
      }
    })
    return options[0]?.field || ''
  }

  const handleChannelGroupChange = async (param: DevopsStepParam) => {
    await ensureMappingFields(param.config?.channelGroupCode)
    if (!param.config) param.config = { ...defaultParam().config }
    param.config.channelTypeFilter = ''
    param.config.channelParamCode = ''
    param.config.projectParamCode = ''
    param.config.mappingField = ''
  }

  const handleChannelTypeChange = async (param: DevopsStepParam) => {
    await ensureMappingFields(param.config?.channelGroupCode, param.config?.channelTypeFilter)
    if (!param.config) param.config = { ...defaultParam().config }
    param.config.channelParamCode = ''
    param.config.projectParamCode = ''
    param.config.mappingField = firstMappingFieldForType(
      param.config.channelGroupCode,
      param.config.channelTypeFilter
    )
  }

  const handleOptionVisible = async (visible: boolean) => {
    if (!visible) return
    await fetchOptions()
  }

  const availableParamTypeOptions = (mode?: string) => {
    if (mode === 'env') return paramTypeOptions.filter((item) => envParamTypes.includes(item.value))
    if (mode === 'paas') {
      return paramTypeOptions.filter((item) => visiblePaasParamTypes.includes(item.value))
    }
    return paramTypeOptions.filter((item) => paramsParamTypes.includes(item.value))
  }

  const availableCompoundParamTypeOptions = (mode?: string) => {
    if (mode !== 'paas') {
      return paramTypeOptions.filter((item) => compoundEnvParamTypes.includes(item.value))
    }
    if (mode === 'paas') {
      return paramTypeOptions.filter(
        (item) =>
          paasParamTypes.includes(item.value) &&
          visiblePaasParamTypes.includes(item.value) &&
          !compositeAddressParamTypes.includes(item.value) &&
          item.value !== 'channelCredential' &&
          !isDeployConfigParamType(item.value)
      )
    }
    return paramTypeOptions.filter((item) => item.value === 'string')
  }

  const isChannelPaasParamType = (paramType?: string) =>
    channelPaasParamTypes.includes(String(paramType || ''))

  const isDynamicPaasParam = (param: DevopsStepParam) =>
    param.mode === 'paas' && dynamicPaasParamTypes.includes(param.paramType)

  const isDynamicCompoundPaasField = (field: CompoundParamField) =>
    field.mode === 'paas' && dynamicPaasParamTypes.includes(field.paramType)

  const isConfigCenterPaasParam = (param: DevopsStepParam | CompoundParamField) =>
    param.mode === 'paas' && param.paramType === 'configCenter'

  const isGitRefParamType = (paramType?: string) =>
    paramType === 'gitBranch' || paramType === 'gitTag'

  const isGitRepositoryURLDependency = (item: DevopsStepParam | CompoundParamField | undefined) => {
    if (!item || !isGitRefParamType(item.paramType)) return false
    const projectParamCode = String(item.config?.projectParamCode || '').trim()
    if (!projectParamCode) return false
    const options =
      'sortOrder' in item ? projectDependencyParamOptions(item) : compoundProjectFieldOptions(item)
    return options.some(
      (option) => option.code === projectParamCode && option.paramType === 'gitRepositoryUrl'
    )
  }

  const dynamicMappingParamType = (item: DevopsStepParam | CompoundParamField) => {
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
    if (field === 'dynamic.branch') return 'gitBranch'
    if (field === 'dynamic.tag') {
      if (groupCode === 'image_registry') {
        return channelType === 'harbor' ? 'harborImageTag' : 'registryImageTag'
      }
      return 'gitTag'
    }
    if (field === 'address.projectUrl') {
      if (groupCode === 'code_repo') return 'gitRepositoryUrl'
      if (channelType === 'harbor') return 'harborProjectUrl'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepositoryUrl'
      if (channelType === 'sonarqube') return 'sonarProjectUrl'
    }
    if (field === 'dynamic.registry') {
      return channelType === 'harbor' ? 'harborProject' : 'registryRepository'
    }
    if (field === 'dynamic.repository') {
      if (channelType === 'harbor') return 'harborProject'
      if (channelType === 'registry' || channelType === 'aliyun_registry')
        return 'registryRepository'
      return channelType === 'jfrog' ? 'jfrogRepository' : 'nexusRepository'
    }
    if (field === 'dynamic.image') return channelType === 'harbor' ? 'harborImage' : 'registryImage'
    if (field === 'dynamic.imageTag')
      return channelType === 'harbor' ? 'harborImageTag' : 'registryImageTag'
    if (field === 'address.registryUrl')
      return channelType === 'harbor' ? 'harborProjectUrl' : 'registryRepositoryUrl'
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
    if (field === 'dynamic.projectKey') return 'sonarProjectKey'
    if (field === 'dynamic.projectName') return 'sonarProjectName'
    if (field === 'address.projectKeyUrl') return 'sonarProjectKeyUrl'
    if (field === 'dynamic.namespace') return 'kubernetesNamespace'
    if (
      [
        'dynamic.deployment',
        'dynamic.statefulSet',
        'dynamic.daemonSet',
        'dynamic.cronJob'
      ].includes(field)
    )
      return 'kubernetesResource'
    if (field === 'dynamic.cluster') return 'kubeNovaCluster'
    if (field === 'dynamic.workspace') return 'kubeNovaWorkspace'
    if (field === 'dynamic.application') return 'kubeNovaApplication'
    if (field === 'dynamic.version') return 'kubeNovaVersion'
    if (field === 'dynamic.workloadType') return 'kubernetesWorkloadType'
    if (field === 'dynamic.resourceName') return 'kubernetesResource'
    if (field === 'dynamic.container' || field === 'dynamic.containerName')
      return channelType === 'kubernetes' ? 'kubernetesContainer' : 'kubeNovaContainer'
    if (field === 'dynamic.imageName') return 'kubernetesImage'
    if (
      field === 'dynamic.deployConfig' ||
      field === 'address.deployConfig' ||
      field === 'address.deploymentConfig'
    )
      return channelType === 'kubernetes' ? 'kubernetesDeployConfig' : 'kubeNovaDeployConfig'
    if (field === 'address.hostConfig') return 'hostConfig'
    if (field === 'address.groupConfig') return 'hostGroupConfig'
    return ''
  }

  const isChannelDynamicMapping = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode' && !!dynamicMappingParamType(item)

  const isAddressDynamicParamType = (paramType?: string) =>
    compositeAddressParamTypes.includes(paramType || '')

  const channelDynamicType = (item: DevopsStepParam | CompoundParamField) =>
    dynamicMappingParamType(item)

  const needChannelBindingParam = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode'
      ? channelVariableNeedsEndpointParam(item) && !useSpecialAddressDependency(item)
      : isChannelDynamicMapping(item) && !isAddressDynamicParamType(channelDynamicType(item))

  const channelBindingLabel = (item: DevopsStepParam | CompoundParamField) =>
    String(item.config?.channelGroupCode || '').trim() === 'code_repo' ? '渠道实例' : '渠道实例'

  const channelBindingPlaceholder = (item: DevopsStepParam | CompoundParamField) =>
    `请选择${channelBindingLabel(item)}`

  const channelVariableNeedsEndpointParam = (item: DevopsStepParam | CompoundParamField) => {
    const option = currentMappingFieldOption(item)
    if (!option) return false
    if (
      option.kind === 'address' ||
      ['addressPicker', 'cascadePicker'].includes(option.uiControl || '')
    )
      return false
    if (option.uiControl === 'deployConfig') return false
    return (option.dependencies || []).some(
      (dep) => dep.source === 'channel' && dep.field === 'endpoint' && dep.required !== false
    )
  }

  const needsChannelTypeFilter = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType === 'channelGroupCode'

  const sameChannelVariableTarget = (
    source: DevopsStepParam | CompoundParamField,
    target: DevopsStepParam | CompoundParamField
  ) => {
    if (target.paramType !== 'channelGroupCode' || source.code === target.code) return false
    if (String(target.config?.mappingField || '').trim() !== 'endpoint') return false
    const sourceGroup = String(source.config?.channelGroupCode || '').trim()
    const targetGroup = String(target.config?.channelGroupCode || '').trim()
    const sourceType = String(source.config?.channelTypeFilter || '').trim()
    const targetType = String(target.config?.channelTypeFilter || '').trim()
    return (
      !!sourceGroup && sourceGroup === targetGroup && (!sourceType || sourceType === targetType)
    )
  }

  const needChannelProjectParam = (item: DevopsStepParam | CompoundParamField) =>
    item.paramType !== 'channelGroupCode' &&
    ['gitBranch', 'gitTag', 'harborImage'].includes(channelDynamicType(item))

  const channelProjectLabel = (item: DevopsStepParam | CompoundParamField) =>
    channelDynamicType(item) === 'harborImage' ? 'Harbor 项目字段' : 'Git 项目字段'

  const showDynamicChannelDependency = (item: DevopsStepParam | CompoundParamField) =>
    !isGitRepositoryURLDependency(item) && !!requiredChannelParamType(item.paramType)

  const needProjectParam = (paramType?: string) =>
    !!dynamicParamSpec[paramType || '']?.projectParamType

  const requiredChannelParamType = (paramType?: string) =>
    dynamicParamSpec[paramType || '']?.channelParamType || ''

  const channelParamOptions = (param: DevopsStepParam) => {
    const channelParamType = requiredChannelParamType(param.paramType)
    if (!channelParamType) return []
    return formData.params.filter(
      (item) => item.paramType === channelParamType && item.code !== param.code
    )
  }

  const channelProjectOptions = (param: DevopsStepParam) => {
    const targetTypes =
      channelDynamicType(param) === 'harborImage'
        ? ['harborProject']
        : ['gitProject', 'gitRepositoryUrl']
    return formData.params.filter(
      (item) =>
        item.code !== param.code &&
        (targetTypes.includes(item.paramType) ||
          (item.paramType === 'channelGroupCode' && targetTypes.includes(channelDynamicType(item))))
    )
  }

  const channelVariableOptions = (param: DevopsStepParam) =>
    formData.params.filter((item) => sameChannelVariableTarget(param, item))

  const channelCredentialSourceOptions = (param: DevopsStepParam) =>
    formData.params.filter(
      (item) =>
        item.code !== param.code &&
        item.mode === 'paas' &&
        (channelCredentialSourceParamTypes.includes(item.paramType) ||
          (item.paramType === 'channelGroupCode' && isChannelCredentialSourceChannelVariable(item)))
    )

  const isChannelCredentialSourceChannelVariable = (item: DevopsStepParam | CompoundParamField) => {
    if (item.paramType !== 'channelGroupCode') return false
    const field = String(item.config?.mappingField || '').trim()
    return field === 'endpoint' || field.startsWith('address.')
  }

  const compoundFields = () => paramForm.config?.compoundFields || []

  const compoundChannelFieldOptions = (field: CompoundParamField) => {
    const channelParamType = requiredChannelParamType(field.paramType)
    if (!channelParamType) return []
    return compoundFields().filter(
      (item) => item.paramType === channelParamType && item.code !== field.code
    )
  }

  const compoundProjectFieldOptions = (field: CompoundParamField) => {
    const targetTypes = dynamicProjectParamTypes(field.paramType)
    if (!targetTypes.length) return []
    return compoundFields().filter(
      (item) =>
        item.code !== field.code &&
        (targetTypes.includes(item.paramType) ||
          (item.paramType === 'channelGroupCode' && targetTypes.includes(channelDynamicType(item))))
    )
  }

  const compoundChannelProjectOptions = (field: CompoundParamField) => {
    const targetTypes =
      channelDynamicType(field) === 'harborImage'
        ? ['harborProject']
        : ['gitProject', 'gitRepositoryUrl']
    return compoundFields().filter(
      (item) =>
        item.code !== field.code &&
        (targetTypes.includes(item.paramType) ||
          (item.paramType === 'channelGroupCode' && targetTypes.includes(channelDynamicType(item))))
    )
  }

  const compoundChannelVariableOptions = (field: CompoundParamField) =>
    compoundFields().filter((item) => sameChannelVariableTarget(field, item))

  const projectDependencyParamOptions = (param: DevopsStepParam) => {
    const targetTypes = dynamicProjectParamTypes(param.paramType)
    if (!targetTypes.length) return []
    return formData.params.filter(
      (item) =>
        item.code !== param.code &&
        (targetTypes.includes(item.paramType) ||
          (item.paramType === 'channelGroupCode' && targetTypes.includes(channelDynamicType(item))))
    )
  }

  const dynamicProjectParamTypes = (paramType?: string) => {
    const spec = dynamicParamSpec[paramType || '']
    return spec?.projectParamTypes || (spec?.projectParamType ? [spec.projectParamType] : [])
  }

  const dependencyProjectLabel = (paramType?: string) =>
    dynamicParamSpec[paramType || '']?.projectLabel || '项目参数'

  const dependencyProjectPlaceholder = (paramType?: string) =>
    `请选择${dependencyProjectLabel(paramType)}`

  const handleCategoryVisible = async (visible: boolean) => {
    if (!visible) return
    await fetchCategories()
  }

  watch(
    () => formData.categoryId,
    () => {
      if (!dialogVisible.value || dialogType.value === 'edit') return
      const content = String(formData.stageContent || '').trim()
      if (isPostCategory.value && (!content || content === defaultStepsContent)) {
        formData.stageContent = defaultPostContent
      }
      if (!isPostCategory.value && (!content || content === defaultPostContent)) {
        formData.stageContent = defaultStepsContent
      }
    }
  )

  const normalizeRuntimeMode = (runtimeMode?: string, mode?: string) => {
    if (mode === 'env') return 'env'
    if (mode === 'params') return 'params'
    if (runtimeMode === 'params' || runtimeMode === 'env') return runtimeMode
    if (mode === 'paas') return 'env'
    return 'params'
  }

  const normalizeConfigCenterValueMode = (valueMode?: string) =>
    configCenterValueModeOptions.some((item) => item.value === valueMode) ? valueMode : 'string'
  const normalizeObjectListValueMode = (valueMode?: string) =>
    valueMode === 'base64' ? 'base64' : 'string'

  const normalizeParamPayload = (param: DevopsStepParam) => {
    const next = { ...param }
    if (!next.config) next.config = { ...defaultParam().config }
    if (next.paramType === 'objectList' && next.mode === 'env') {
      next.mode = 'paas'
    }
    next.config = {
      ...defaultParam().config,
      ...next.config,
      valueMode: normalizeConfigCenterValueMode(next.config.valueMode)
    }
    next.defaultValue =
      next.defaultValue === undefined || next.defaultValue === null ? '' : String(next.defaultValue)
    next.sortOrder = Number(next.sortOrder || 0)
    next.runtimeMode = normalizeRuntimeMode(next.runtimeMode, next.mode)
    next.runtimeConfig = canConfigureRuntimeParam(next) ? next.runtimeConfig === true : false
    if (next.mode === 'env') {
      next.runtimeMode = 'env'
      if (!envParamTypes.includes(next.paramType)) {
        next.paramType = 'string'
      }
      next.selectList = []
    }
    if (next.mode === 'params' && !paramsParamTypes.includes(next.paramType)) {
      next.paramType = 'string'
    }
    if (next.mode === 'params') next.runtimeMode = 'params'
    if (next.mode === 'paas' && !paasParamTypes.includes(next.paramType)) {
      next.paramType = 'list'
    }
    if (next.mode === 'paas' && isChannelPaasParamType(next.paramType)) {
      next.config.channelGroupCode = channelGroupCode(next.paramType)
      next.config.channelTypeFilter = channelTypeFilter(next.paramType)
      next.config.mappingField = 'endpoint'
    }
    if (next.mode === 'paas' && isDeployConfigParamType(next.paramType)) {
      const spec = deployConfigChannelSpec(next.paramType)
      next.runtimeMode = 'params'
      next.required = true
      next.config.fullRow = true
      next.config.channelGroupCode = spec.groupCode
      next.config.mappingField = spec.mappingField
    }
    if (next.mode === 'paas' && next.paramType === 'channelCredential') {
      normalizeChannelCredentialConfig(next)
    }
    if (next.mode === 'paas' && next.paramType === 'configCenter') {
      next.config.valueMode = normalizeConfigCenterValueMode(next.config.valueMode)
    }
    next.config.options = next.selectList || next.config.options || []
    next.config.compoundFields = (next.config.compoundFields || []).map((field) => {
      const normalized = {
        ...defaultCompoundField(),
        ...field,
        config: { ...defaultParam().config, ...(field.config || {}), compoundFields: [] }
      }
      normalized.defaultValue =
        normalized.defaultValue === undefined || normalized.defaultValue === null
          ? ''
          : String(normalized.defaultValue)
      if (!['env', 'paas'].includes(normalized.mode)) {
        normalized.mode = 'env'
        normalized.paramType = 'string'
      }
      if (normalized.mode === 'env' && !compoundEnvParamTypes.includes(normalized.paramType)) {
        normalized.paramType = 'string'
      }
      if (normalized.mode === 'paas' && !paasParamTypes.includes(normalized.paramType)) {
        normalized.paramType = 'list'
      }
      if (isDeployConfigParamType(normalized.paramType)) {
        normalized.paramType = 'list'
      }
      if (normalized.mode === 'paas' && isChannelPaasParamType(normalized.paramType)) {
        normalized.config.channelGroupCode = channelGroupCode(normalized.paramType)
        normalized.config.channelTypeFilter = channelTypeFilter(normalized.paramType)
        normalized.config.mappingField = 'endpoint'
      }
      if (normalized.mode === 'paas' && normalized.paramType === 'voucherModel') {
        normalizeVoucherConfig(normalized.config)
      }
      if (normalized.paramType === 'channelCredential') {
        normalized.paramType = 'list'
      }
      if (normalized.paramType === 'configCenter') {
        normalized.config.valueMode = 'string'
      } else {
        normalized.config.configTypeId = ''
        normalized.config.configTypeCode = ''
        normalized.config.valueMode = 'string'
      }
      normalized.runtimeMode = 'env'
      normalized.config.options = normalized.selectList || normalized.config.options || []
      return normalized
    })
    if (next.paramType === 'channelGroupCode') {
      next.config.channelGroupCode = next.config.channelGroupCode || ''
    }
    if (next.paramType === 'voucherModel') {
      next.config.voucherModel = next.config.voucherModel || ''
      normalizeVoucherConfig(next.config)
    } else if (next.paramType === 'channelCredential') {
      normalizeChannelCredentialConfig(next)
    } else {
      next.config.credentialMode = 'field'
      next.config.credentialSourceParamCode = ''
    }
    if (next.paramType === 'hostGroupTargets') {
      next.config.renderMode = ['json', 'jsonBase64', 'ansible', 'ansibleBase64'].includes(
        next.config.renderMode || ''
      )
        ? next.config.renderMode
        : 'json'
    } else if (next.paramType !== 'hostGroupTargets') {
      next.config.renderMode = next.config.renderMode || 'normal'
    }
    if (isDeployConfigParamType(next.paramType)) {
      const spec = deployConfigChannelSpec(next.paramType)
      next.config.fullRow = true
      next.config.channelGroupCode = spec.groupCode
      next.config.mappingField = spec.mappingField
    }
    if (!needProjectParam(next.paramType) && !needChannelProjectParam(next)) {
      next.config.projectParamCode = ''
    }
    if (next.paramType === 'objectList') {
      next.config.valueMode = normalizeObjectListValueMode(next.config.valueMode)
    } else if (next.paramType !== 'configCenter') {
      next.config.configTypeId = ''
      next.config.configTypeCode = ''
      next.config.valueMode = 'string'
    }
    next.config.componentParamCode = ''
    next.channelGroupCode = next.config.channelGroupCode
    next.mappingField = next.config.mappingField
    next.voucherModel = next.config.voucherModel
    return next
  }

  const normalizeStepPayload = () => ({
    ...formData,
    params: formData.params.map((item) => normalizeParamPayload(item))
  })

  const validateParamsBeforeSubmit = () => {
    const seen = new Set<string>()
    const params = formData.params.map((item) => normalizeParamPayload(item))
    for (const item of params) {
      if (!String(item.name || '').trim()) {
        ElMessage.warning('参数名称不能为空')
        return false
      }
      if (!String(item.code || '').trim()) {
        ElMessage.warning('参数编码不能为空')
        return false
      }
      if (seen.has(item.code)) {
        ElMessage.warning(`参数编码重复：${item.code}`)
        return false
      }
      seen.add(item.code)
      if (item.mode === 'params' && item.paramType === 'choice' && !item.config?.options?.length) {
        ElMessage.warning(`单选参数「${item.name}」必须配置选项`)
        return false
      }
      if (item.paramType === 'objectList') {
        const fields = item.config?.compoundFields || []
        if (!fields.length) {
          ElMessage.warning(`对象列表「${item.name}」至少需要一个字段`)
          return false
        }
        const fieldSeen = new Set<string>()
        for (const field of fields) {
          if (!String(field.name || '').trim()) {
            ElMessage.warning(`对象列表「${item.name}」存在未填写字段名称的项`)
            return false
          }
          if (!String(field.code || '').trim()) {
            ElMessage.warning(`对象列表「${item.name}」存在未填写字段编码的项`)
            return false
          }
          if (fieldSeen.has(field.code)) {
            ElMessage.warning(`对象列表「${item.name}」字段编码重复：${field.code}`)
            return false
          }
          fieldSeen.add(field.code)
          if (
            field.paramType === 'objectList' ||
            field.paramType === 'file' ||
            isDeployConfigParamType(field.paramType)
          ) {
            ElMessage.warning(`对象列表「${item.name}」字段类型不支持：${field.paramType}`)
            return false
          }
          if (!['env', 'paas'].includes(field.mode)) {
            ElMessage.warning(
              `对象列表「${item.name}」字段「${field.name}」只支持环境变量或平台变量`
            )
            return false
          }
          if (field.mode === 'paas' && field.paramType === 'list' && !field.selectList?.length) {
            ElMessage.warning(`对象列表「${item.name}」字段「${field.name}」必须配置选项`)
            return false
          }
          if (field.mode === 'paas' && field.paramType === 'channelGroupCode') {
            if (!field.config?.channelGroupCode || !field.config?.channelTypeFilter) {
              ElMessage.warning(
                `对象列表「${item.name}」字段「${field.name}」必须配置渠道分组和渠道类型`
              )
              return false
            }
            if (!field.config?.mappingField) {
              ElMessage.warning(`对象列表「${item.name}」字段「${field.name}」必须配置映射字段`)
              return false
            }
            for (const dep of channelParamDependencies(field)) {
              const paramCode = dependencyParamCode(field, dep.field)
              if (dep.required !== false && !paramCode) {
                ElMessage.warning(
                  `对象列表「${item.name}」字段「${field.name}」必须选择${dep.name || dep.field}`
                )
                return false
              }
              if (
                paramCode &&
                !channelDependencyOptions(field, dep.field, compoundFields()).some(
                  (option) => option.code === paramCode
                )
              ) {
                ElMessage.warning(
                  `对象列表「${item.name}」字段「${field.name}」的${dep.name || dep.field}不匹配`
                )
                return false
              }
            }
            if (
              field.config?.channelParamCode &&
              !compoundChannelVariableOptions(field).some(
                (option) => option.code === field.config?.channelParamCode
              )
            ) {
              ElMessage.warning(
                `对象列表「${item.name}」字段「${field.name}」的${channelBindingLabel(field)}不匹配`
              )
              return false
            }
            if (
              field.config?.projectParamCode &&
              !compoundChannelProjectOptions(field).some(
                (option) => option.code === field.config?.projectParamCode
              )
            ) {
              ElMessage.warning(
                `对象列表「${item.name}」字段「${field.name}」的${channelProjectLabel(field)}不匹配`
              )
              return false
            }
          }
          if (field.mode === 'paas' && field.paramType === 'voucherModel') {
            if (!field.config?.voucherModel) {
              ElMessage.warning(`对象列表「${item.name}」字段「${field.name}」必须配置凭证类型`)
              return false
            }
            if (isCredentialFieldMode(field.config) && !field.config?.mappingField) {
              ElMessage.warning(
                `对象列表「${item.name}」字段「${field.name}」必须配置凭证类型和映射字段`
              )
              return false
            }
          }
          if (field.mode === 'paas' && isChannelPaasParamType(field.paramType)) {
            const spec = {
              groupCode: channelGroupCode(field.paramType),
              channelTypeFilter: channelTypeFilter(field.paramType)
            }
            if (
              field.config?.channelGroupCode !== spec.groupCode ||
              field.config?.channelTypeFilter !== spec.channelTypeFilter
            ) {
              ElMessage.warning(
                `对象列表「${item.name}」字段「${field.name}」必须使用对应渠道分组和类型`
              )
              return false
            }
          }
          if (
            field.mode === 'paas' &&
            (isDynamicCompoundPaasField(field) || isConfigCenterPaasParam(field))
          ) {
            if (field.paramType === 'configCenter') {
              if (!field.config?.configTypeId && !field.config?.configTypeCode) {
                ElMessage.warning(`对象列表「${item.name}」字段「${field.name}」必须选择配置类型`)
                return false
              }
            }
            if (field.config?.channelParamCode) {
              const requiredChannel = requiredChannelParamType(field.paramType)
              const channelParam = (item.config?.compoundFields || []).find(
                (target) => target.code === field.config?.channelParamCode
              )
              if (!channelParam || channelParam.paramType !== requiredChannel) {
                ElMessage.warning(`对象列表「${item.name}」字段「${field.name}」的渠道字段不匹配`)
                return false
              }
            }
            if (field.config?.projectParamCode) {
              const targetTypes = dynamicProjectParamTypes(field.paramType)
              const projectParam = (item.config?.compoundFields || []).find(
                (target) => target.code === field.config?.projectParamCode
              )
              if (!projectParam || !targetTypes.includes(projectParam.paramType)) {
                ElMessage.warning(
                  `对象列表「${item.name}」字段「${field.name}」的${dependencyProjectLabel(field.paramType)}不匹配`
                )
                return false
              }
            }
          }
        }
      }
      if (item.mode === 'paas' && item.paramType === 'list' && !item.config?.options?.length) {
        ElMessage.warning(`平台列表「${item.name}」必须配置选项`)
        return false
      }
      if (item.mode === 'paas' && item.paramType === 'channelGroupCode') {
        if (!item.config?.channelGroupCode || !item.config?.channelTypeFilter) {
          ElMessage.warning(`渠道参数「${item.name}」必须配置渠道分组和渠道类型`)
          return false
        }
        if (!item.config?.mappingField) {
          ElMessage.warning(`渠道参数「${item.name}」必须配置映射字段`)
          return false
        }
        for (const dep of channelParamDependencies(item)) {
          const paramCode = dependencyParamCode(item, dep.field)
          if (dep.required !== false && !paramCode) {
            ElMessage.warning(`渠道参数「${item.name}」必须选择${dep.name || dep.field}`)
            return false
          }
          if (
            paramCode &&
            !channelDependencyOptions(item, dep.field, formData.params).some(
              (option) => option.code === paramCode
            )
          ) {
            ElMessage.warning(`渠道参数「${item.name}」的${dep.name || dep.field}不匹配`)
            return false
          }
        }
        if (
          item.config?.channelParamCode &&
          !channelVariableOptions(item).some(
            (option) => option.code === item.config?.channelParamCode
          )
        ) {
          ElMessage.warning(`渠道参数「${item.name}」的${channelBindingLabel(item)}不匹配`)
          return false
        }
        if (
          item.config?.projectParamCode &&
          !channelProjectOptions(item).some(
            (option) => option.code === item.config?.projectParamCode
          )
        ) {
          ElMessage.warning(`渠道参数「${item.name}」的${channelProjectLabel(item)}不匹配`)
          return false
        }
      }
      if (item.mode === 'paas' && item.paramType === 'voucherModel') {
        if (!item.config?.voucherModel) {
          ElMessage.warning(`凭证参数「${item.name}」必须配置凭证类型`)
          return false
        }
        if (isCredentialFieldMode(item.config) && !item.config?.mappingField) {
          ElMessage.warning(`凭证参数「${item.name}」必须配置凭证类型和映射字段`)
          return false
        }
      }
      if (item.mode === 'paas' && item.paramType === 'channelCredential') {
        if (!item.config?.credentialSourceParamCode) {
          ElMessage.warning(`渠道凭证「${item.name}」必须选择来源参数`)
          return false
        }
        if (
          !channelCredentialSourceOptions(item).some(
            (option) => option.code === item.config?.credentialSourceParamCode
          )
        ) {
          ElMessage.warning(`渠道凭证「${item.name}」来源参数不支持`)
          return false
        }
        if (isChannelCredentialFieldMode(item.config) && !item.config?.mappingField) {
          ElMessage.warning(`渠道凭证「${item.name}」必须选择映射字段`)
          return false
        }
      }
      if (item.mode === 'paas' && isChannelPaasParamType(item.paramType)) {
        const spec = {
          groupCode: channelGroupCode(item.paramType),
          channelTypeFilter: channelTypeFilter(item.paramType)
        }
        if (
          item.config?.channelGroupCode !== spec.groupCode ||
          item.config?.channelTypeFilter !== spec.channelTypeFilter
        ) {
          ElMessage.warning(`渠道参数「${item.name}」必须使用对应渠道分组和类型`)
          return false
        }
      }
      if (item.mode === 'paas' && isDeployConfigParamType(item.paramType)) {
        const spec = deployConfigChannelSpec(item.paramType)
        if (
          item.runtimeMode !== 'params' ||
          item.config?.fullRow !== true ||
          item.config?.channelGroupCode !== spec.groupCode ||
          item.config?.mappingField !== spec.mappingField
        ) {
          ElMessage.warning(`部署配置「${item.name}」必须使用执行变量、整行配置和固定渠道映射`)
          return false
        }
      }
      if (item.mode === 'paas' && (isDynamicPaasParam(item) || isConfigCenterPaasParam(item))) {
        if (item.paramType === 'configCenter') {
          if (!item.config?.configTypeId && !item.config?.configTypeCode) {
            ElMessage.warning(`配置中心参数「${item.name}」必须选择配置类型`)
            return false
          }
        }
      }
    }
    return true
  }

  const handleStatusChange = async (row: DevopsStepTemplate, value: string | number | boolean) => {
    await updateDevopsJenkinsStepStatusApi(row.id, value ? 1 : 0)
    ElMessage.success('状态已更新')
    await fetchList()
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    if (!validateStageContentSyntax()) return
    if (!validateParamsBeforeSubmit()) return
    submitLoading.value = true
    try {
      const payload = normalizeStepPayload()
      if (dialogType.value === 'add') {
        await createDevopsJenkinsStepApi(payload)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsJenkinsStepApi(currentId.value, payload)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await fetchList()
    } finally {
      submitLoading.value = false
    }
  }

  const handleDelete = async (row: DevopsStepTemplate) => {
    await ElMessageBox.confirm(`确认删除 Jenkins 步骤 ${row.name}？`, '删除确认', {
      type: 'warning'
    })
    await deleteDevopsJenkinsStepApi(row.id)
    ElMessage.success('删除成功')
    await fetchList()
  }

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
  .devops-jenkins-step-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .devops-jenkins-step-page :deep(.summary-list),
  .devops-jenkins-step-page :deep(.summary-item) {
    display: none !important;
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

    &:hover {
      border-color: var(--el-color-primary-light-5);
      box-shadow: 0 12px 28px rgb(15 23 42 / 8%);
    }
  }

  .step-card-head {
    display: grid;
    grid-template-columns: 46px minmax(0, 1fr) auto;
    gap: 10px;
    align-items: center;
    padding: 12px 14px;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .step-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 46px;
    height: 46px;
    font-size: 26px;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .step-title {
    min-width: 0;

    .name,
    .code {
      display: block;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .name {
      font-size: 16px;
      font-weight: 700;
    }

    .code {
      margin-top: 4px;
      color: var(--el-text-color-secondary);
    }
  }

  .step-card-body {
    display: flex;
    flex: 1;
    flex-direction: column;
    gap: 10px;
    min-height: 0;
    padding: 12px 14px;
  }

  .tag-row {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
  }

  .desc {
    display: -webkit-box;
    flex: 1;
    min-height: 38px;
    margin: 0;
    overflow: hidden;
    line-height: 1.5;
    text-overflow: ellipsis;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
  }

  .step-card-footer {
    display: grid;
    grid-template-columns: max-content auto;
    column-gap: 8px;
    align-items: center;
    justify-content: space-between;
    padding: 10px 14px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  .created-time {
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
  }

  .footer-actions {
    display: flex;
    flex: 0 0 auto;
    gap: 8px;
    align-items: center;
  }

  .footer-actions :deep(.el-button + .el-button) {
    margin-left: 0;
  }

  :deep(.devops-polish-dialog) {
    --el-dialog-padding-primary: 0;

    display: flex;
    flex-direction: column;
    max-height: min(88vh, 860px);
    overflow: hidden;
    border-radius: 8px;
  }

  :deep(.devops-polish-dialog .el-dialog__header) {
    flex: none;
    padding: 18px 22px 18px;
    margin-right: 0;
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  :deep(.devops-polish-dialog .el-dialog__title) {
    font-size: 18px;
    font-weight: 700;
    line-height: 1.2;
    color: var(--el-text-color-primary);
  }

  :deep(.devops-polish-dialog .el-dialog__headerbtn) {
    top: 12px;
    right: 14px;
  }

  :deep(.devops-polish-dialog .el-dialog__body) {
    display: flex;
    flex-direction: column;
    flex: 1;
    min-height: 0;
    padding: 28px 42px 0;
    overflow: hidden;
  }

  :deep(.devops-polish-dialog .el-dialog__footer) {
    flex: none;
    padding: 12px 22px 16px;
    border-top: 1px solid var(--el-border-color-lighter);
  }

  :deep(.devops-polish-dialog .el-form) {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
  }

  .dialog-footer-actions {
    display: flex;
    gap: 8px;
    justify-content: flex-end;
  }

  .wizard-steps {
    position: relative;
    display: flex;
    gap: 0;
    padding: 0 0 20px;
    margin: 0 0 12px;

    &::before {
      position: absolute;
      right: 64px;
      bottom: 23px;
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
    min-height: 42px;
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
    width: 28px;
    height: 28px;
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
    flex: 1;
    min-height: 0;
    padding: 0 2px 16px;
    overflow: auto;
  }

  .dialog-step-panel :deep(.el-row) {
    margin-bottom: 4px;
  }

  .dialog-step-panel :deep(.el-form-item) {
    margin-bottom: 18px;
  }

  .dialog-step-panel :deep(.el-form-item__label) {
    font-weight: 500;
    color: var(--el-text-color-regular);
  }

  .dialog-step-panel :deep(.el-input__wrapper),
  .dialog-step-panel :deep(.el-select__wrapper),
  .dialog-step-panel :deep(.el-textarea__inner) {
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  .dialog-step-panel :deep(.el-input__wrapper),
  .dialog-step-panel :deep(.el-select__wrapper),
  .param-edit-form :deep(.el-input__wrapper),
  .param-edit-form :deep(.el-select__wrapper),
  .compound-field-form :deep(.el-input__wrapper),
  .compound-field-form :deep(.el-select__wrapper) {
    min-height: 42px;
    padding: 0 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  .dialog-step-panel :deep(.el-select__wrapper),
  .param-edit-form :deep(.el-select__wrapper),
  .compound-field-form :deep(.el-select__wrapper) {
    padding: 10px 12px;
  }

  .dialog-step-panel :deep(.el-select__selection),
  .param-edit-form :deep(.el-select__selection),
  .compound-field-form :deep(.el-select__selection) {
    min-width: 0;
  }

  .dialog-step-panel :deep(.el-textarea__inner),
  .param-edit-form :deep(.el-textarea__inner),
  .compound-field-form :deep(.el-textarea__inner) {
    padding: 10px 12px;
    background: #ffffff;
    border-radius: 14px;
    box-shadow: 0 0 0 1px #e2e8f0 inset;
    transition:
      box-shadow 0.18s ease,
      background-color 0.18s ease;
  }

  .dialog-step-panel :deep(.el-input__wrapper:hover),
  .dialog-step-panel :deep(.el-select__wrapper:hover),
  .dialog-step-panel :deep(.el-textarea__inner:hover),
  .param-edit-form :deep(.el-input__wrapper:hover),
  .param-edit-form :deep(.el-select__wrapper:hover),
  .param-edit-form :deep(.el-textarea__inner:hover),
  .compound-field-form :deep(.el-input__wrapper:hover),
  .compound-field-form :deep(.el-select__wrapper:hover),
  .compound-field-form :deep(.el-textarea__inner:hover) {
    box-shadow: 0 0 0 1px #94a3b8 inset;
  }

  .dialog-step-panel :deep(.el-input__wrapper.is-focus),
  .dialog-step-panel :deep(.el-select__wrapper.is-focused),
  .dialog-step-panel :deep(.el-textarea__inner:focus),
  .param-edit-form :deep(.el-input__wrapper.is-focus),
  .param-edit-form :deep(.el-select__wrapper.is-focused),
  .param-edit-form :deep(.el-textarea__inner:focus),
  .compound-field-form :deep(.el-input__wrapper.is-focus),
  .compound-field-form :deep(.el-select__wrapper.is-focused),
  .compound-field-form :deep(.el-textarea__inner:focus) {
    box-shadow:
      0 0 0 1px #2563eb inset,
      0 0 0 4px rgb(37 99 235 / 14%);
  }

  .dialog-step-panel :deep(.el-select__caret),
  .param-edit-form :deep(.el-select__caret),
  .compound-field-form :deep(.el-select__caret) {
    width: 16px;
    height: 16px;
    color: #334155;
  }

  .dialog-step-panel :deep(.el-select__caret svg),
  .param-edit-form :deep(.el-select__caret svg),
  .compound-field-form :deep(.el-select__caret svg) {
    width: 14px;
    height: 14px;
  }

  .param-step-panel {
    display: flex;
    flex-direction: column;
    gap: 10px;
    height: 100%;
    min-height: 0;
    padding-bottom: 14px;
    overflow: auto;
  }

  .full-form-item {
    margin-bottom: 0;

    :deep(.el-form-item__label) {
      display: none;
    }

    :deep(.el-form-item__content) {
      width: 100%;
      margin-left: 0 !important;
    }
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
    grid-template-columns: repeat(auto-fill, minmax(92px, 1fr));
    gap: 10px;
    width: 100%;

    button {
      display: flex;
      flex-direction: column;
      gap: 6px;
      align-items: center;
      justify-content: center;
      min-width: 0;
      height: 72px;
      padding: 8px 6px;
      font-size: 24px;
      cursor: pointer;
      background: var(--el-bg-color);
      border: 1px solid var(--el-border-color-light);
      border-radius: 8px;

      span {
        max-width: 100%;
        overflow: hidden;
        font-size: 12px;
        line-height: 1;
        color: var(--el-text-color-secondary);
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      &.active {
        background: var(--el-color-primary-light-9);
        border-color: var(--el-color-primary);
        box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
      }
    }
  }

  .param-section {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    min-height: 0;
    padding: 0;
    margin-bottom: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .section-head,
  .section-actions,
  .option-head,
  .option-row {
    display: flex;
    gap: 10px;
    align-items: center;
  }

  .section-head {
    flex-wrap: wrap;
    justify-content: space-between;
    padding: 12px 14px;
    margin-bottom: 0;
    background: var(--el-fill-color-extra-light);
    border-bottom: 1px solid var(--el-border-color-lighter);
    border-radius: 8px 8px 0 0;

    > div:first-child {
      min-width: 280px;
    }

    > div:first-child > span {
      display: block;
      margin-top: 4px;
      color: var(--el-text-color-secondary);
    }
  }

  .section-actions {
    flex-wrap: wrap;
    justify-content: flex-end;
  }

  .param-stat {
    display: inline-grid;
    grid-auto-columns: max-content;
    grid-auto-flow: column;
    column-gap: 2px;
    align-items: center;
    justify-content: center;
    min-width: 84px;
    height: 34px;
    padding: 0 12px;
    font-variant-numeric: tabular-nums;
    line-height: 1;
    color: var(--el-text-color-secondary);
    white-space: nowrap;
    background: var(--el-fill-color-light);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;

    strong {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      min-width: 16px;
      height: 18px;
      font-size: 16px;
      line-height: 18px;
      color: var(--el-text-color-primary);
    }

    em {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      height: 18px;
      margin: 0;
      font-size: 13px;
      font-style: normal;
      line-height: 18px;
    }

    &.warning {
      color: var(--el-color-warning);
      background: var(--el-color-warning-light-9);
      border-color: var(--el-color-warning-light-7);
    }
  }

  .param-table-wrap {
    flex: 1 1 auto;
    min-height: 0;
    padding: 12px 14px 24px;
    overflow: auto;
  }

  .param-table-wrap :deep(.el-empty) {
    height: 100%;
  }

  .param-table {
    width: 100%;
  }

  .param-table :deep(.el-button.is-text) {
    padding: 0 2px;
    margin-left: 0;
  }

  :deep(.param-edit-dialog) {
    min-height: min(82vh, 640px);
    max-height: min(88vh, 760px);
  }

  :deep(.param-edit-dialog .el-dialog__body) {
    min-height: 360px;
    max-height: min(70vh, 560px);
    padding: 18px 22px 8px;
    overflow: auto;
  }

  :deep(.param-edit-dialog .el-dialog__footer) {
    padding: 12px 22px 16px;
  }

  :deep(.compound-field-dialog) {
    max-height: min(86vh, 720px);
  }

  :deep(.compound-field-dialog .el-dialog__body) {
    max-height: min(68vh, 520px);
    padding: 18px 22px 8px;
    overflow: auto;
  }

  :deep(.compound-field-dialog .el-dialog__footer) {
    padding: 12px 22px 16px;
  }

  .param-edit-form {
    padding: 0;
  }

  .param-edit-form :deep(.el-form-item) {
    margin-bottom: 14px;
  }

  .param-edit-form :deep(.el-segmented) {
    width: 100%;
  }

  .param-edit-form :deep(.el-segmented__item) {
    min-width: 0;
  }

  .param-edit-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    column-gap: 20px;
  }

  .param-edit-full {
    grid-column: 1 / -1;
  }

  .param-switch-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: 14px;
    margin-bottom: 2px;
  }

  .artifact-section {
    flex: none;
    margin: 0;
    overflow: hidden;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .artifact-head {
    display: flex;
    gap: 16px;
    align-items: center;
    justify-content: space-between;
    min-height: 58px;
    padding: 12px 14px;
    background: var(--el-fill-color-extra-light);
    border-bottom: 1px solid var(--el-border-color-lighter);

    strong,
    span {
      display: block;
    }

    strong {
      font-size: 15px;
      color: var(--el-text-color-primary);
    }

    > div:first-child span {
      margin-top: 4px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .artifact-switch {
    display: inline-flex;
    flex: none;
    gap: 10px;
    align-items: center;
    font-weight: 600;
    color: var(--el-text-color-regular);
    white-space: nowrap;
  }

  .artifact-grid {
    padding: 14px 14px 2px;
  }

  .param-special-grid {
    align-items: flex-start;
    margin-bottom: 0;
  }

  .param-special-grid :deep(.el-form-item__label) {
    flex: 0 0 128px;
    width: 128px !important;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .param-special-grid .wide-label-field :deep(.el-form-item__label) {
    flex-basis: 168px;
    width: 168px !important;
  }

  .channel-dependency-form-item :deep(.el-form-item__content) {
    min-width: 0;
  }

  .channel-dependency-editor {
    display: flex;
    flex-direction: column;
    gap: 10px;
    width: 100%;
    min-width: 0;
    padding: 10px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 6px;
  }

  .channel-dependency-editor :deep(.el-segmented) {
    width: fit-content;
    max-width: 100%;
  }

  .channel-dependency-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 10px;
  }

  .channel-dependency-grid.single {
    grid-template-columns: minmax(0, 1fr);
  }

  .param-field-tip {
    padding: 0 0 12px;
    font-size: 12px;
    line-height: 18px;
    color: var(--el-text-color-secondary);
  }

  .param-field-tip.compact {
    padding: 0 0 8px;
    margin-top: -2px;
  }

  .param-switch-field {
    padding: 10px 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .param-switch-field :deep(.el-form-item__label) {
    margin-bottom: 8px;
  }

  .param-switch-field :deep(.el-form-item__content) {
    justify-content: flex-start;
    min-height: 32px;
  }

  .param-description {
    margin-bottom: 10px;
  }

  .option-editor {
    padding: 10px 12px 2px;
    margin-bottom: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px dashed var(--el-border-color-light);
    border-radius: 8px;
  }

  .option-head {
    justify-content: space-between;
    margin-bottom: 8px;
  }

  .option-row {
    margin-bottom: 8px;

    .el-input {
      flex: 1;
    }
  }

  .compound-editor {
    padding: 10px 12px 2px;
    margin-bottom: 12px;
    background: var(--el-fill-color-extra-light);
    border: 1px dashed var(--el-border-color-light);
    border-radius: 8px;
  }

  .compound-field-table {
    margin-bottom: 10px;
  }

  .compound-field-table :deep(.el-button.is-text) {
    padding: 0 2px;
    margin-left: 0;
  }

  .compound-field-card {
    padding: 12px;
    margin-bottom: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .compound-field-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 10px;
  }

  .compound-field-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 14px 18px;
  }

  .compound-field-form {
    padding: 0;
  }

  .option-editor.compact {
    margin-top: 8px;
    margin-bottom: 8px;
  }

  .param-switch-grid.compact {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    margin-top: 8px;
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
