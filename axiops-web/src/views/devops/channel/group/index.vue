<template>
  <div class="devops-channel-group-page art-full-height">
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
      empty-text="暂无渠道分组"
      @add="openDialog('add')"
      @refresh="refreshAll"
      @pagination:size-change="handleSizeChange"
      @pagination:current-change="handleCurrentChange"
    >
      <template #default="{ item }">
        <div class="group-card">
          <div class="group-card-head">
            <div class="group-icon" :style="{ color: groupColor(item) }">
              <ArtSvgIcon :icon="groupIcon(item)" />
            </div>
            <div class="group-title">
              <div class="group-name-row">
                <span class="group-name" :title="item.name">{{ item.name }}</span>
                <ElTag v-if="item.isSystem" size="small" type="warning">系统</ElTag>
              </div>
              <span class="group-code" :title="item.code">{{ item.code }}</span>
            </div>
            <ElSwitch
              :model-value="item.status === 1"
              :disabled="item.isSystem"
              @change="(val) => handleStatusChange(item, Boolean(val))"
            />
          </div>

          <p class="group-desc" :title="item.description || '暂无描述'">
            {{ item.description || '暂无描述' }}
          </p>

          <div class="group-stats">
            <div>
              <span>类型</span>
              <strong>{{ groupTypeCount(item.code) }}</strong>
            </div>
            <div>
              <span>实例</span>
              <strong>{{ channelCount(item.id) }}</strong>
            </div>
            <div>
              <span>正常</span>
              <strong>{{ groupHealthCount(item.id, 'healthy') }}</strong>
            </div>
            <div>
              <span>异常</span>
              <strong>{{ groupHealthCount(item.id, 'unhealthy') }}</strong>
            </div>
          </div>

          <div class="group-actions">
            <ElButton text type="success" @click="openManageDrawer(item)">管理</ElButton>
            <ElButton text type="primary" @click="openDialog('edit', item)">编辑</ElButton>
            <ElButton text type="danger" :disabled="item.isSystem" @click="handleDelete(item)">
              删除
            </ElButton>
          </div>
        </div>
      </template>
    </ArtCard>

    <ElDialog
      v-model="dialogVisible"
      :title="dialogType === 'add' ? '新增分组' : '编辑分组'"
      width="640px"
      align-center
      :close-on-click-modal="false"
      @close="resetForm"
    >
      <ElForm ref="formRef" :model="formData" :rules="rules" label-width="124px">
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
                placeholder="例如 build_engine"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="分组类型" prop="groupType">
              <template #label>
                <span class="form-label-help">
                  分组类型
                  <ElTooltip :content="formHelpText('groupType')" placement="top">
                    <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
                  </ElTooltip>
                </span>
              </template>
              <ElInput v-model="formData.groupType" :disabled="submitLoading" placeholder="类型" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="排序" prop="sortOrder">
              <ElInputNumber
                v-model="formData.sortOrder"
                :disabled="submitLoading"
                :min="0"
                style="width: 100%"
              />
            </ElFormItem>
          </ElCol>
        </ElRow>

        <ElFormItem label="图标">
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

        <ElFormItem label="允许类型" prop="allowedChannelTypes">
          <template #label>
            <span class="form-label-help">
              允许类型
              <ElTooltip :content="formHelpText('allowedChannelTypes')" placement="top">
                <ElIcon class="field-help-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </span>
          </template>
          <ElSelect
            v-model="formData.allowedChannelTypes"
            :disabled="submitLoading"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="请选择允许的渠道类型"
            style="width: 100%"
          >
            <ElOption
              v-for="item in channelTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>

        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="系统分组" prop="isSystem">
              <ElSwitch
                v-model="formData.isSystem"
                :disabled="dialogType === 'edit' || submitLoading"
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

        <ElFormItem label="描述" prop="description">
          <ElInput
            v-model="formData.description"
            :disabled="submitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入分组描述"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="submitLoading" @click="dialogVisible = false">取消</ElButton>
        <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
          {{ dialogType === 'add' ? '新增' : '保存' }}
        </ElButton>
      </template>
    </ElDialog>

    <ElDrawer
      v-model="manageVisible"
      :title="manageDrawerTitle"
      size="78%"
      class="channel-manage-drawer"
      @close="resetManage"
    >
      <div v-if="managedGroup" class="manage-drawer">
        <div class="manage-hero">
          <div class="manage-hero-icon" :style="{ color: groupColor(managedGroup) }">
            <ArtSvgIcon :icon="groupIcon(managedGroup)" />
          </div>
          <div class="manage-hero-main">
            <div class="manage-hero-title">
              <strong>{{ managedGroup.name }}</strong>
              <ElTag v-if="managedGroup.isSystem" size="small" type="warning">系统</ElTag>
            </div>
            <div class="manage-hero-code">{{ managedGroup.code }}</div>
            <div class="manage-hero-desc">{{ managedGroup.description || '暂无描述' }}</div>
          </div>
        </div>

        <div class="manage-summary">
          <div>
            <span>渠道类型</span>
            <strong>{{ managedChannelTypes.length }}</strong>
          </div>
          <div>
            <span>变量字段</span>
            <strong>{{ managedMappingCount }}</strong>
          </div>
          <div>
            <span>渠道实例</span>
            <strong>{{ channelCount(managedGroup.id) }}</strong>
          </div>
          <div>
            <span>正常实例</span>
            <strong>{{ groupHealthCount(managedGroup.id, 'healthy') }}</strong>
          </div>
        </div>

        <div class="manage-section">
          <div class="manage-section-head">
            <div>
              <strong>渠道类型</strong>
              <span>默认字段由系统注入，自定义动态变量在类型编辑中维护</span>
            </div>
            <div class="manage-toolbar">
              <ElButton type="primary" @click="openTypeDialog('add')">新增类型</ElButton>
              <ElButton @click="openParamMappingDialog">参数映射</ElButton>
              <ElButton :loading="manageLoading" @click="refreshManage">刷新</ElButton>
            </div>
          </div>
          <ElTable :data="managedChannelTypes" border height="460" v-loading="manageLoading">
            <ElTableColumn prop="name" label="名称" min-width="140" show-overflow-tooltip />
            <ElTableColumn prop="code" label="编码" min-width="130" show-overflow-tooltip />
            <ElTableColumn label="凭据类型" min-width="180" show-overflow-tooltip>
              <template #default="{ row }">
                {{ credentialTypeNames(row.credentialTypes || []) }}
              </template>
            </ElTableColumn>
            <ElTableColumn label="变量字段" min-width="260">
              <template #default="{ row }">
                <div class="table-tags">
                  <ElTag
                    v-for="field in mappingFieldRowsOf(row).slice(0, 3)"
                    :key="field.field"
                    size="small"
                    type="info"
                    :title="field.field"
                  >
                    {{ field.name || field.field }}
                  </ElTag>
                  <ElTag v-if="mappingFieldRowsOf(row).length > 3" size="small" type="info">
                    +{{ mappingFieldRowsOf(row).length - 3 }}
                  </ElTag>
                  <span v-if="!mappingFieldRowsOf(row).length" class="muted-text">-</span>
                </div>
              </template>
            </ElTableColumn>
            <ElTableColumn prop="testStrategy" label="测试策略" width="110" />
            <ElTableColumn label="状态" width="90">
              <template #default="{ row }">
                <ElTag :type="row.status === 1 ? 'success' : 'info'" size="small">
                  {{ row.status === 1 ? '启用' : '停用' }}
                </ElTag>
              </template>
            </ElTableColumn>
            <ElTableColumn label="操作" width="160" fixed="right">
              <template #default="{ row }">
                <ElButton link type="primary" @click="openTypeDialog('edit', row)">编辑</ElButton>
                <ElButton
                  link
                  type="danger"
                  :disabled="row.isSystem"
                  @click="handleTypeDelete(row)"
                >
                  删除
                </ElButton>
              </template>
            </ElTableColumn>
          </ElTable>
        </div>
      </div>
      <ElEmpty v-else description="请选择渠道分组" />
    </ElDrawer>

    <ElDialog
      v-model="typeDialogVisible"
      :title="typeDialogType === 'add' ? '新增渠道类型' : '编辑渠道类型'"
      width="1180px"
      align-center
      :close-on-click-modal="false"
      @close="resetTypeForm"
    >
      <ElForm ref="typeFormRef" :model="typeFormData" :rules="typeRules" label-width="120px">
        <div class="type-form-section">
          <div class="section-title">基础信息</div>
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="名称" prop="name">
                <ElInput v-model="typeFormData.name" :disabled="managerSubmitLoading" />
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="编码" prop="code">
                <ElInput
                  v-model="typeFormData.code"
                  :disabled="typeDialogType === 'edit' || managerSubmitLoading"
                  placeholder="例如 gitlab"
                />
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElRow :gutter="16">
            <ElCol :span="12">
              <ElFormItem label="测试策略" prop="testStrategy">
                <ElSelect
                  v-model="typeFormData.testStrategy"
                  :disabled="managerSubmitLoading"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="item in testStrategyOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="连接模式" prop="connectionMode">
                <ElSelect
                  v-model="typeFormData.connectionMode"
                  :disabled="managerSubmitLoading"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="item in connectionModeOptions"
                    :key="item.value"
                    :label="item.label"
                    :value="item.value"
                  />
                </ElSelect>
              </ElFormItem>
            </ElCol>
          </ElRow>
          <ElFormItem label="凭据类型" prop="credentialTypes">
            <ElSelect
              v-model="typeFormData.credentialTypes"
              :disabled="managerSubmitLoading"
              multiple
              filterable
              collapse-tags
              collapse-tags-tooltip
              style="width: 100%"
            >
              <ElOption
                v-for="item in credentialTypeOptions"
                :key="item.value"
                :label="item.label"
                :value="item.value"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="配置 Schema" prop="configSchema">
            <ElInput
              v-model="typeFormData.configSchema"
              :disabled="managerSubmitLoading"
              type="textarea"
              :rows="3"
              placeholder='{"fields":[{"key":"apiPath","label":"API 路径","type":"input"}]}'
            />
          </ElFormItem>
          <ElFormItem label="状态" prop="status">
            <ElRadioGroup v-model="typeFormData.status" :disabled="managerSubmitLoading">
              <ElRadio :value="1">启用</ElRadio>
              <ElRadio :value="0">停用</ElRadio>
            </ElRadioGroup>
          </ElFormItem>
        </div>

        <div class="type-form-section">
          <div class="section-title-row">
            <div>
              <div class="section-title">自定义动态变量</div>
              <div class="section-subtitle"
                >默认字段 endpoint、name、code、metadata.version 由系统自动注入</div
              >
            </div>
            <ElButton :disabled="managerSubmitLoading" @click="addMappingFieldRow">
              新增变量
            </ElButton>
          </div>
          <div class="editable-list">
            <div class="editable-list-head mapping-field-grid">
              <span>字段编码</span>
              <span>显示名称</span>
              <span>类型</span>
              <span>控件</span>
              <span>Provider</span>
              <span>依赖字段</span>
              <span>地址规则</span>
              <span>必填</span>
              <span>手输</span>
              <span>操作</span>
            </div>
            <div
              v-for="(row, index) in mappingFieldRows"
              :key="`mapping-field-${index}`"
              class="editable-row mapping-field-grid"
            >
              <ElInput
                v-model="row.field"
                :disabled="managerSubmitLoading"
                placeholder="dynamic.project"
              />
              <ElInput v-model="row.name" :disabled="managerSubmitLoading" placeholder="项目" />
              <ElSelect v-model="row.kind" :disabled="managerSubmitLoading" style="width: 100%">
                <ElOption label="动态下拉" value="dynamic" />
                <ElOption label="地址选择" value="address" />
                <ElOption label="组合配置" value="composite" />
                <ElOption label="默认字段" value="default" />
              </ElSelect>
              <ElSelect
                v-model="row.uiControl"
                :disabled="managerSubmitLoading"
                style="width: 100%"
              >
                <ElOption label="下拉" value="select" />
                <ElOption label="输入+选择" value="addressPicker" />
                <ElOption label="部署配置" value="deployConfig" />
                <ElOption label="只读" value="readonly" />
              </ElSelect>
              <ElInput
                v-model="row.provider"
                :disabled="managerSubmitLoading"
                placeholder="registry.images"
              />
              <ElInput
                v-model="row.dependenciesText"
                :disabled="managerSubmitLoading"
                type="textarea"
                :rows="2"
                placeholder='[{"field":"dynamic.project","source":"param","name":"项目","required":true}]'
              />
              <ElInput
                v-model="row.addressBuilder"
                :disabled="managerSubmitLoading"
                placeholder="endpoint + repository"
              />
              <ElSwitch v-model="row.required" :disabled="managerSubmitLoading" />
              <ElSwitch v-model="row.allowManualInput" disabled />
              <ElButton
                link
                type="danger"
                :disabled="managerSubmitLoading"
                @click="removeMappingFieldRow(index)"
              >
                删除
              </ElButton>
            </div>
            <ElEmpty v-if="!mappingFieldRows.length" description="暂无自定义动态变量" />
          </div>
        </div>
      </ElForm>
      <template #footer>
        <ElButton :disabled="managerSubmitLoading" @click="typeDialogVisible = false">
          取消
        </ElButton>
        <ElButton type="primary" :loading="managerSubmitLoading" @click="handleTypeSubmit">
          保存
        </ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="paramMappingDialogVisible"
      :title="paramMappingDialogTitle"
      width="980px"
      align-center
      :close-on-click-modal="false"
    >
      <div class="param-mapping-toolbar">
        <ElInput
          v-model="paramMappingKeyword"
          clearable
          placeholder="搜索参数类型"
          @keyup.enter="handleParamMappingSearch"
          @clear="handleParamMappingSearch"
        />
        <ElButton @click="handleParamMappingSearch">搜索</ElButton>
        <ElButton type="primary" @click="openParamMappingForm('add')">新增映射</ElButton>
      </div>
      <ElTable
        v-loading="paramMappingLoading"
        :data="paramMappingData"
        border
        height="420"
        empty-text="暂无参数映射"
      >
        <ElTableColumn prop="paramName" label="参数名称" min-width="140" show-overflow-tooltip />
        <ElTableColumn prop="paramType" label="参数类型" min-width="150" show-overflow-tooltip />
        <ElTableColumn label="渠道类型" min-width="150" show-overflow-tooltip>
          <template #default="{ row }">
            {{ channelTypeFilterName(row.channelTypeFilter) }}
          </template>
        </ElTableColumn>
        <ElTableColumn prop="description" label="说明" min-width="220" show-overflow-tooltip />
        <ElTableColumn prop="sortOrder" label="排序" width="80" />
        <ElTableColumn label="状态" width="90">
          <template #default="{ row }">
            <ElTag :type="row.status === 1 ? 'success' : 'info'" size="small">
              {{ row.status === 1 ? '启用' : '停用' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="来源" width="90">
          <template #default="{ row }">
            <ElTag :type="row.isSystem ? 'warning' : 'info'" size="small">
              {{ row.isSystem ? '系统' : '自定义' }}
            </ElTag>
          </template>
        </ElTableColumn>
        <ElTableColumn label="操作" width="150" fixed="right">
          <template #default="{ row }">
            <ElButton link type="primary" @click="openParamMappingForm('edit', row)">
              编辑
            </ElButton>
            <ElButton
              link
              type="danger"
              :disabled="row.isSystem"
              @click="handleParamMappingDelete(row)"
            >
              删除
            </ElButton>
          </template>
        </ElTableColumn>
      </ElTable>
      <div class="param-mapping-pagination">
        <ElPagination
          v-model:current-page="paramMappingPagination.current"
          v-model:page-size="paramMappingPagination.size"
          background
          layout="total, sizes, prev, pager, next"
          :total="paramMappingPagination.total"
          :page-sizes="[10, 20, 50]"
          @size-change="handleParamMappingSizeChange"
          @current-change="handleParamMappingCurrentChange"
        />
      </div>
    </ElDialog>

    <ElDialog
      v-model="paramMappingFormVisible"
      :title="paramMappingFormType === 'add' ? '新增参数映射' : '编辑参数映射'"
      width="620px"
      align-center
      :close-on-click-modal="false"
      @close="resetParamMappingForm"
    >
      <ElForm
        ref="paramMappingFormRef"
        :model="paramMappingFormData"
        :rules="paramMappingRules"
        label-width="112px"
      >
        <ElFormItem label="参数类型" prop="paramType">
          <ElInput
            v-model="paramMappingFormData.paramType"
            :disabled="paramMappingFormType === 'edit' || paramMappingSubmitLoading"
            placeholder="例如 gitRepositoryUrl"
          />
        </ElFormItem>
        <ElFormItem label="参数名称" prop="paramName">
          <ElInput
            v-model="paramMappingFormData.paramName"
            :disabled="paramMappingSubmitLoading"
            placeholder="例如 Git 仓库地址"
          />
        </ElFormItem>
        <ElFormItem label="渠道分组" prop="groupCode">
          <ElInput v-model="paramMappingFormData.groupCode" disabled />
        </ElFormItem>
        <ElFormItem label="渠道类型">
          <ElSelect
            v-model="paramMappingFormData.channelTypeFilter"
            :disabled="paramMappingSubmitLoading"
            clearable
            filterable
            placeholder="不限制渠道类型"
            style="width: 100%"
          >
            <ElOption label="不限制" value="" />
            <ElOption
              v-for="item in managedChannelTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="排序" prop="sortOrder">
          <ElInputNumber
            v-model="paramMappingFormData.sortOrder"
            :disabled="paramMappingSubmitLoading"
            :min="0"
            style="width: 100%"
          />
        </ElFormItem>
        <ElFormItem label="状态" prop="status">
          <ElRadioGroup v-model="paramMappingFormData.status" :disabled="paramMappingSubmitLoading">
            <ElRadio :value="1">启用</ElRadio>
            <ElRadio :value="0">停用</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="说明">
          <ElInput
            v-model="paramMappingFormData.description"
            :disabled="paramMappingSubmitLoading"
            type="textarea"
            :rows="3"
            placeholder="请输入说明"
          />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton :disabled="paramMappingSubmitLoading" @click="paramMappingFormVisible = false">
          取消
        </ElButton>
        <ElButton
          type="primary"
          :loading="paramMappingSubmitLoading"
          @click="handleParamMappingSubmit"
        >
          保存
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onActivated, onMounted, reactive, ref } from 'vue'
  import { ElButton, ElMessage, ElMessageBox, ElSwitch, ElTag } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import { QuestionFilled } from '@element-plus/icons-vue'
  import {
    createStepChannelParamMappingApi,
    deleteStepChannelParamMappingApi,
    listStepChannelParamMappingApi,
    updateStepChannelParamMappingApi,
    type StepChannelParamMappingForm,
    type StepChannelParamMappingItem
  } from '@/api/devops/pipeline'
  import {
    createDevopsChannelGroupApi,
    createDevopsChannelTypeApi,
    deleteDevopsChannelGroupApi,
    deleteDevopsChannelTypeApi,
    listDevopsChannelApi,
    listDevopsChannelGroupApi,
    listDevopsChannelTypeApi,
    updateDevopsChannelGroupApi,
    updateDevopsChannelTypeApi,
    type DevopsChannel,
    type DevopsChannelGroup,
    type DevopsChannelGroupForm,
    type DevopsChannelType,
    type DevopsChannelTypeForm
  } from '@/api/devops/channel'
  defineOptions({ name: 'DevopsChannelGroup' })

  const loading = ref(false)
  const submitLoading = ref(false)
  const showSearchBar = ref(true)
  const searchRef = ref()
  const formRef = ref<FormInstance>()
  const dialogVisible = ref(false)
  const dialogType = ref<'add' | 'edit'>('add')
  const currentId = ref('')
  const tableData = ref<DevopsChannelGroup[]>([])
  const channelTypes = ref<DevopsChannelType[]>([])
  const channelData = ref<DevopsChannel[]>([])
  const searchParams = ref<Record<string, any>>({})
  const manageVisible = ref(false)
  const manageLoading = ref(false)
  const managerSubmitLoading = ref(false)
  const managedGroup = ref<DevopsChannelGroup | null>(null)
  const typeFormRef = ref<FormInstance>()
  const typeDialogVisible = ref(false)
  const typeDialogType = ref<'add' | 'edit'>('add')
  const typeCurrentId = ref('')
  const paramMappingDialogVisible = ref(false)
  const paramMappingFormVisible = ref(false)
  const paramMappingLoading = ref(false)
  const paramMappingSubmitLoading = ref(false)
  const paramMappingKeyword = ref('')
  const paramMappingData = ref<StepChannelParamMappingItem[]>([])
  const paramMappingFormRef = ref<FormInstance>()
  const paramMappingFormType = ref<'add' | 'edit'>('add')
  const paramMappingCurrentId = ref('')
  const paramMappingPagination = ref({
    current: 1,
    size: 10,
    total: 0
  })

  type MappingFieldRow = {
    field: string
    name: string
    kind: string
    uiControl: string
    provider: string
    dependenciesText: string
    addressBuilder: string
    required: boolean
    allowManualInput: boolean
  }

  const mappingFieldRows = ref<MappingFieldRow[]>([])

  const searchForm = ref({
    name: undefined,
    code: undefined,
    status: undefined
  })

  const pagination = ref({
    current: 1,
    size: 15,
    total: 0
  })

  const iconOptions = [
    { icon: 'ri:flow-chart', color: '#2563eb' },
    { icon: 'ri:git-repository-line', color: '#f97316' },
    { icon: 'ri:archive-stack-line', color: '#16a34a' },
    { icon: 'simple-icons:docker', color: '#0369a1' },
    { icon: 'ri:shield-check-line', color: '#0f766e' },
    { icon: 'simple-icons:kubernetes', color: '#326ce5' },
    { icon: 'ri:server-line', color: '#475569' },
    { icon: 'ri:rocket-line', color: '#7c3aed' },
    { icon: 'ri:apps-2-line', color: '#64748b' }
  ]

  const defaultForm = (): DevopsChannelGroupForm => ({
    name: '',
    code: '',
    description: '',
    sortOrder: 0,
    isSystem: false,
    status: 1,
    groupType: '',
    allowedChannelTypes: [],
    icon: 'ri:apps-2-line',
    iconColor: '#64748b'
  })

  const formData = reactive<DevopsChannelGroupForm>(defaultForm())
  const defaultTypeForm = (): DevopsChannelTypeForm => ({
    code: '',
    name: '',
    groupCode: managedGroup.value?.code || '',
    credentialTypes: ['username_password', 'token'],
    configSchema: '',
    mappingFields: '',
    testStrategy: 'http',
    icon: managedGroup.value?.icon || 'ri:apps-2-line',
    iconColor: managedGroup.value?.iconColor || '#64748b',
    connectionMode: 'server',
    metadataStrategy: 'product_version',
    isSystem: false,
    status: 1
  })

  const typeFormData = reactive<DevopsChannelTypeForm>(defaultTypeForm())
  const defaultParamMappingForm = (): StepChannelParamMappingForm => ({
    paramType: '',
    paramName: '',
    groupCode: managedGroup.value?.code || '',
    channelTypeFilter: '',
    description: '',
    sortOrder: 0,
    status: 1
  })
  const paramMappingFormData = reactive<StepChannelParamMappingForm>(defaultParamMappingForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
  }
  const typeRules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入编码', trigger: 'blur' }]
  }
  const paramMappingRules: FormRules = {
    paramType: [{ required: true, message: '请输入参数类型', trigger: 'blur' }],
    paramName: [{ required: true, message: '请输入参数名称', trigger: 'blur' }],
    groupCode: [{ required: true, message: '请选择渠道分组', trigger: 'change' }]
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

  const testStrategyOptions = [
    { label: 'HTTP', value: 'http' },
    { label: 'TCP', value: 'tcp' },
    { label: 'Kubernetes', value: 'kubernetes' },
    { label: '工具', value: 'tool' },
    { label: '主机', value: 'host' },
    { label: '主机组', value: 'host_group' },
    { label: '无', value: 'none' }
  ]

  const connectionModeOptions = [
    { label: '服务端', value: 'server' },
    { label: 'API', value: 'api' },
    { label: 'Kubernetes', value: 'kubernetes' },
    { label: '工具', value: 'tool' },
    { label: '协议', value: 'protocol' },
    { label: '主机', value: 'host' },
    { label: '主机组', value: 'host_group' }
  ]

  const channelTypeOptions = computed(() =>
    channelTypes.value.map((item) => ({
      label: `${item.name}（${item.code}）`,
      value: item.code
    }))
  )
  const managedChannelTypeOptions = computed(() =>
    managedChannelTypes.value.map((item) => ({
      label: `${item.name}（${item.code}）`,
      value: item.code
    }))
  )
  const businessChannels = computed(() => channelData.value.filter((item) => !item.isSystem))
  const manageDrawerTitle = computed(() =>
    managedGroup.value ? `管理 ${managedGroup.value.name}` : '管理渠道分组'
  )
  const paramMappingDialogTitle = computed(() =>
    managedGroup.value ? `${managedGroup.value.name} 参数映射` : '参数映射'
  )
  const managedChannelTypes = computed(() => {
    const code = managedGroup.value?.code || ''
    return channelTypes.value.filter((item) => item.groupCode === code)
  })
  const managedMappingCount = computed(() =>
    managedChannelTypes.value.reduce((total, item) => total + mappingFieldRowsOf(item).length, 0)
  )

  const searchItems = computed(() => [
    { label: '名称', key: 'name', type: 'input', props: { clearable: true } },
    { label: '编码', key: 'code', type: 'input', props: { clearable: true } },
    {
      label: '状态',
      key: 'status',
      type: 'select',
      props: { clearable: true, options: statusOptions }
    }
  ])

  const credentialTypeName = (code: string) =>
    credentialTypeOptions.find((item) => item.value === code)?.label || code
  const credentialTypeNames = (codes: string[]) =>
    codes.length ? codes.map((item) => credentialTypeName(item)).join('、') : '-'
  const channelTypeFilterName = (code?: string) => {
    const value = String(code || '').trim()
    if (!value) return '不限制'
    return channelTypes.value.find((item) => item.code === value)?.name || value
  }

  const groupIcon = (row: DevopsChannelGroup) => row.icon || 'ri:apps-2-line'
  const groupColor = (row: DevopsChannelGroup) => row.iconColor || '#64748b'

  const formHelpText = (key: string) => {
    const texts: Record<string, string> = {
      code: '分组业务 Key，例如 build_engine。项目绑定构建渠道时会按这个编码查询实例，创建后不可修改。',
      groupType: '分组的业务分类，用于后续扩展筛选和治理，不影响当前允许类型校验。',
      allowedChannelTypes: '限制该分组下能创建的渠道类型，渠道中心新增实例时只展示这里配置的类型。'
    }
    return texts[key] || ''
  }

  const channelCount = (groupId: string) =>
    businessChannels.value.filter((item) => item.groupId === groupId).length

  const groupTypeCount = (groupCode: string) =>
    channelTypes.value.filter((item) => item.groupCode === groupCode).length

  const groupHealthCount = (groupId: string, status: string) =>
    businessChannels.value.filter(
      (item) => item.groupId === groupId && item.healthStatus === status
    ).length

  const pickParams = () =>
    Object.entries(searchForm.value).reduce(
      (acc, [key, value]) => {
        if (value !== undefined && value !== null && value !== '') acc[key] = value
        return acc
      },
      {} as Record<string, any>
    )

  const getData = async () => {
    loading.value = true
    try {
      const res = await listDevopsChannelGroupApi({
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...searchParams.value
      })
      tableData.value = res.items || []
      pagination.value.total = res.total || 0
    } finally {
      loading.value = false
    }
  }

  const getChannelTypes = async () => {
    const res = await listDevopsChannelTypeApi({ page: 1, pageSize: 200, status: -1 })
    channelTypes.value = res.items || []
  }

  const getChannels = async () => {
    const res = await listDevopsChannelApi({ page: 1, pageSize: 200, status: -1 })
    channelData.value = res.items || []
  }

  const refreshAll = async () => {
    await Promise.all([getChannelTypes(), getChannels()])
    await getData()
  }

  const refreshManage = async () => {
    manageLoading.value = true
    try {
      await getChannelTypes()
      if (paramMappingDialogVisible.value) await getParamMappings()
    } finally {
      manageLoading.value = false
    }
  }

  const handleSearch = async () => {
    await searchRef.value?.validate?.()
    searchParams.value = pickParams()
    pagination.value.current = 1
    await getData()
  }

  const handleReset = async () => {
    searchParams.value = {}
    pagination.value.current = 1
    await getData()
  }

  const handleSizeChange = async (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    await getData()
  }

  const handleCurrentChange = async (current: number) => {
    pagination.value.current = current
    await getData()
  }

  const openDialog = (type: 'add' | 'edit', row?: DevopsChannelGroup) => {
    dialogType.value = type
    currentId.value = row?.id || ''
    Object.assign(formData, defaultForm(), row || {}, {
      allowedChannelTypes: row?.allowedChannelTypes || []
    })
    dialogVisible.value = true
  }

  const resetForm = () => {
    currentId.value = ''
    Object.assign(formData, defaultForm())
    formRef.value?.clearValidate()
  }

  const pickIcon = (icon: string, color: string) => {
    formData.icon = icon
    formData.iconColor = color
  }

  const openManageDrawer = async (row: DevopsChannelGroup) => {
    managedGroup.value = row
    manageVisible.value = true
    await refreshManage()
  }

  const resetManage = () => {
    managedGroup.value = null
  }

  const openTypeDialog = (type: 'add' | 'edit', row?: DevopsChannelType) => {
    typeDialogType.value = type
    typeCurrentId.value = row?.id || ''
    Object.assign(typeFormData, defaultTypeForm(), row || {}, {
      groupCode: managedGroup.value?.code || row?.groupCode || '',
      credentialTypes: row?.credentialTypes || defaultTypeForm().credentialTypes
    })
    mappingFieldRows.value = parseMappingFieldRows(typeFormData.mappingFields)
    typeDialogVisible.value = true
  }

  const openParamMappingDialog = async () => {
    paramMappingDialogVisible.value = true
    paramMappingKeyword.value = ''
    paramMappingPagination.value.current = 1
    await getParamMappings()
  }

  const getParamMappings = async () => {
    if (!managedGroup.value) return
    paramMappingLoading.value = true
    try {
      const res = await listStepChannelParamMappingApi({
        groupCode: managedGroup.value.code,
        paramType: paramMappingKeyword.value.trim() || undefined,
        status: -1,
        page: paramMappingPagination.value.current,
        pageSize: paramMappingPagination.value.size
      })
      paramMappingData.value = res.items || []
      paramMappingPagination.value.total = res.total || 0
    } finally {
      paramMappingLoading.value = false
    }
  }

  const handleParamMappingSearch = async () => {
    paramMappingPagination.value.current = 1
    await getParamMappings()
  }

  const handleParamMappingSizeChange = async (size: number) => {
    paramMappingPagination.value.size = size
    paramMappingPagination.value.current = 1
    await getParamMappings()
  }

  const handleParamMappingCurrentChange = async (current: number) => {
    paramMappingPagination.value.current = current
    await getParamMappings()
  }

  const openParamMappingForm = (type: 'add' | 'edit', row?: StepChannelParamMappingItem) => {
    paramMappingFormType.value = type
    paramMappingCurrentId.value = row?.id || ''
    Object.assign(paramMappingFormData, defaultParamMappingForm(), row || {}, {
      groupCode: managedGroup.value?.code || row?.groupCode || '',
      channelTypeFilter: row?.channelTypeFilter || '',
      status: row?.status ?? 1
    })
    paramMappingFormVisible.value = true
  }

  const resetParamMappingForm = () => {
    paramMappingCurrentId.value = ''
    Object.assign(paramMappingFormData, defaultParamMappingForm())
    paramMappingFormRef.value?.clearValidate()
  }

  const handleParamMappingSubmit = async () => {
    await paramMappingFormRef.value?.validate()
    paramMappingSubmitLoading.value = true
    try {
      paramMappingFormData.groupCode = managedGroup.value?.code || paramMappingFormData.groupCode
      const payload: StepChannelParamMappingForm = {
        paramType: paramMappingFormData.paramType?.trim(),
        paramName: paramMappingFormData.paramName.trim(),
        groupCode: paramMappingFormData.groupCode,
        channelTypeFilter: paramMappingFormData.channelTypeFilter || '',
        description: paramMappingFormData.description || '',
        sortOrder: paramMappingFormData.sortOrder || 0,
        status: paramMappingFormData.status ?? 1
      }
      if (paramMappingFormType.value === 'add') {
        await createStepChannelParamMappingApi(payload)
        ElMessage.success('新增成功')
      } else {
        await updateStepChannelParamMappingApi(paramMappingCurrentId.value, payload)
        ElMessage.success('保存成功')
      }
      paramMappingFormVisible.value = false
      await getParamMappings()
    } finally {
      paramMappingSubmitLoading.value = false
    }
  }

  const handleParamMappingDelete = async (row: StepChannelParamMappingItem) => {
    if (row.isSystem) {
      ElMessage.warning('系统映射不能删除')
      return
    }
    await ElMessageBox.confirm(`确定删除映射"${row.paramName}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteStepChannelParamMappingApi(row.id)
    ElMessage.success('删除成功')
    await getParamMappings()
  }

  const resetTypeForm = () => {
    typeCurrentId.value = ''
    Object.assign(typeFormData, defaultTypeForm())
    mappingFieldRows.value = []
    typeFormRef.value?.clearValidate()
  }

  const handleTypeSubmit = async () => {
    await typeFormRef.value?.validate()
    managerSubmitLoading.value = true
    try {
      typeFormData.groupCode = managedGroup.value?.code || typeFormData.groupCode
      typeFormData.configSchema = normalizeJsonText(typeFormData.configSchema)
      typeFormData.mappingFields = serializeMappingFieldRows()
      if (typeDialogType.value === 'add') {
        await createDevopsChannelTypeApi(typeFormData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsChannelTypeApi(typeCurrentId.value, typeFormData)
        ElMessage.success('保存成功')
      }
      typeDialogVisible.value = false
      await refreshAll()
      await refreshManage()
    } catch (error) {
      if (error instanceof SyntaxError) {
        ElMessage.error(error.message || '配置 Schema 或映射字段不是合法 JSON')
        return
      }
      throw error
    } finally {
      managerSubmitLoading.value = false
    }
  }

  const normalizeJsonText = (content?: string) => {
    const text = String(content || '').trim()
    if (!text) return ''
    JSON.parse(text)
    return text
  }

  const parseMappingFieldRows = (content?: string): MappingFieldRow[] => {
    const text = String(content || '').trim()
    if (!text) return []
    try {
      const parsed = JSON.parse(text) as { fields?: Array<Record<string, unknown>> }
      return (parsed.fields || [])
        .map((item) => ({
          field: String(item.field || '').trim(),
          name: String(item.name || '').trim(),
          kind: String(item.kind || 'dynamic').trim(),
          uiControl: String(item.uiControl || 'select').trim(),
          provider: String(item.provider || '').trim(),
          dependenciesText: JSON.stringify(item.dependencies || []),
          addressBuilder: String(item.addressBuilder || '').trim(),
          required: Boolean(item.required),
          allowManualInput: true
        }))
        .filter((item) => item.field)
    } catch {
      return []
    }
  }

  const serializeMappingFieldRows = () => {
    const fields = mappingFieldRows.value
      .map((item) => {
        const dependencies = parseDependenciesText(item.dependenciesText)
        return {
          field: item.field.trim(),
          name: (item.name || item.field).trim(),
          kind: item.kind || 'dynamic',
          uiControl: item.uiControl || (item.kind === 'address' ? 'addressPicker' : 'select'),
          provider: item.provider.trim(),
          dependencies,
          allowManualInput: true,
          addressBuilder: item.addressBuilder.trim(),
          required: item.required
        }
      })
      .filter((item) => item.field)
    const duplicated = fields.find(
      (item, index) => fields.findIndex((field) => field.field === item.field) !== index
    )
    if (duplicated) throw new SyntaxError(`映射字段重复：${duplicated.field}`)
    return fields.length ? JSON.stringify({ version: 2, fields }) : ''
  }

  const parseDependenciesText = (text: string) => {
    const raw = String(text || '').trim()
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) throw new SyntaxError('依赖字段必须是 JSON 数组')
    return parsed.map((item) => ({
      field: String(item.field || '').trim(),
      source: String(item.source || 'param').trim(),
      name: String(item.name || '').trim(),
      required: Boolean(item.required)
    }))
  }

  const addMappingFieldRow = () => {
    mappingFieldRows.value.push({
      field: '',
      name: '',
      kind: 'dynamic',
      uiControl: 'select',
      provider: '',
      dependenciesText: '[]',
      addressBuilder: '',
      required: false,
      allowManualInput: true
    })
  }

  const removeMappingFieldRow = (index: number) => {
    mappingFieldRows.value.splice(index, 1)
  }

  const mappingFieldRowsOf = (row: DevopsChannelType) => parseMappingFieldRows(row.mappingFields)

  const handleTypeDelete = async (row: DevopsChannelType) => {
    if (row.isSystem) {
      ElMessage.warning('系统类型不能删除')
      return
    }
    await ElMessageBox.confirm(`确定删除类型"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsChannelTypeApi(row.id)
    ElMessage.success('删除成功')
    await refreshAll()
    await refreshManage()
  }

  const handleSubmit = async () => {
    await formRef.value?.validate()
    submitLoading.value = true
    try {
      if (dialogType.value === 'add') {
        await createDevopsChannelGroupApi(formData)
        ElMessage.success('新增成功')
      } else {
        await updateDevopsChannelGroupApi(currentId.value, formData)
        ElMessage.success('保存成功')
      }
      dialogVisible.value = false
      await refreshAll()
    } finally {
      submitLoading.value = false
    }
  }

  const handleStatusChange = async (row: DevopsChannelGroup, val: boolean) => {
    await updateDevopsChannelGroupApi(row.id, { ...row, status: val ? 1 : 0 })
    row.status = val ? 1 : 0
    ElMessage.success('状态更新成功')
    await refreshAll()
  }

  const handleDelete = async (row: DevopsChannelGroup) => {
    if (row.isSystem) {
      ElMessage.warning('系统分组不能删除')
      return
    }
    await ElMessageBox.confirm(`确定删除分组"${row.name}"吗？`, '删除确认', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning'
    })
    await deleteDevopsChannelGroupApi(row.id)
    ElMessage.success('删除成功')
    await refreshAll()
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

<style lang="scss" scoped>
  .devops-channel-group-page {
    display: flex;
    flex-direction: column;
    height: 100%;
    min-height: 0;
    overflow: hidden;
  }

  .group-card {
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .group-card {
    display: flex;
    flex-direction: column;
    min-width: 0;
    min-height: 214px;
    overflow: hidden;
    padding: 16px;
  }

  .group-card-head {
    display: flex;
    gap: 12px;
    align-items: center;
    min-width: 0;
  }

  .group-icon {
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

  .group-title {
    flex: 1;
    min-width: 0;
  }

  .group-name-row {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .group-name {
    overflow: hidden;
    font-weight: 600;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-code {
    display: block;
    margin-top: 5px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .group-desc {
    display: -webkit-box;
    min-height: 40px;
    margin: 14px 0;
    overflow: hidden;
    color: var(--el-text-color-regular);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .group-stats {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px;
    margin-bottom: 12px;
  }

  .group-stats div {
    min-width: 0;
    padding: 8px;
    background: var(--el-fill-color-lighter);
    border-radius: 6px;
  }

  .group-stats span {
    display: block;
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .group-actions {
    display: flex;
    justify-content: flex-end;
    gap: 8px;
    padding-top: 12px;
    margin-top: auto;
    border-top: 1px solid var(--el-border-color-lighter);
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

  .manage-drawer {
    display: flex;
    flex-direction: column;
    gap: 14px;
    height: 100%;
    min-height: 0;
  }

  .manage-hero {
    display: flex;
    gap: 14px;
    align-items: center;
    padding: 14px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .manage-hero-icon {
    display: flex;
    flex: 0 0 48px;
    align-items: center;
    justify-content: center;
    width: 48px;
    height: 48px;
    font-size: 28px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .manage-hero-main {
    min-width: 0;
  }

  .manage-hero-title {
    display: flex;
    gap: 8px;
    align-items: center;
    min-width: 0;
  }

  .manage-hero-title strong {
    overflow: hidden;
    font-size: 16px;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .manage-hero-code,
  .manage-hero-desc {
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .manage-hero-code {
    margin-top: 4px;
  }

  .manage-hero-desc {
    margin-top: 6px;
  }

  .manage-summary {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 10px;
  }

  .manage-summary div {
    min-width: 0;
    padding: 12px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .manage-summary span {
    display: block;
    margin-bottom: 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .manage-summary strong {
    display: block;
    overflow: hidden;
    font-size: 18px;
    color: var(--el-text-color-primary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .manage-section {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: 14px;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .manage-section-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
    margin-bottom: 12px;
  }

  .manage-section-head strong {
    display: block;
    color: var(--el-text-color-primary);
  }

  .manage-section-head span {
    display: block;
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .manage-toolbar {
    display: flex;
    flex: 0 0 auto;
    gap: 8px;
    align-items: center;
  }

  .table-tags {
    display: flex;
    flex-wrap: wrap;
    gap: 6px;
    align-items: center;
  }

  .muted-text {
    color: var(--el-text-color-placeholder);
  }

  .type-form-section {
    padding: 14px 14px 4px;
    margin-bottom: 14px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .section-title,
  .section-title-row {
    margin-bottom: 12px;
  }

  .section-subtitle {
    margin-top: 4px;
    font-size: 12px;
    font-weight: 400;
    color: var(--el-text-color-secondary);
  }

  .section-title,
  .section-title-row .section-title {
    font-weight: 700;
    color: var(--el-text-color-primary);
  }

  .section-title-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
  }

  .editable-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 10px;
  }

  .editable-list-head,
  .editable-row {
    display: grid;
    gap: 8px;
    align-items: center;
  }

  .editable-list-head {
    padding: 0 8px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .editable-row {
    padding: 8px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .mapping-field-grid {
    grid-template-columns:
      minmax(150px, 1fr) minmax(120px, 0.8fr) 105px 115px minmax(130px, 1fr)
      minmax(180px, 1.2fr) minmax(130px, 1fr) 56px 56px 60px;
  }

  .param-mapping-toolbar {
    display: grid;
    grid-template-columns: minmax(220px, 1fr) auto auto;
    gap: 8px;
    margin-bottom: 12px;
  }

  .param-mapping-pagination {
    display: flex;
    justify-content: center;
    padding-top: 12px;
  }
</style>
