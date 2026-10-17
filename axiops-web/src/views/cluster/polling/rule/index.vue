<template>
  <div class="inspection-page" :class="{ 'is-detail': currentTemplateId }">
    <template v-if="!currentTemplateId">
      <div class="filter-bar">
        <ArtSearchBar
          :model-value="templateFilters"
          :items="templateSearchItems"
          @search="searchTemplates"
          @reset="resetTemplateFilters"
        />
        <ElButton type="primary" :icon="Plus" @click="openTemplateDialog()">新建规则库</ElButton>
      </div>

      <div class="ruleset-grid" v-loading="templateLoading || metricLoading">
        <article
          v-for="item in templates"
          :key="item.id"
          class="ruleset-card"
          @click="enterTemplate(item)"
        >
          <div class="ruleset-top">
            <div class="ruleset-title">
              <strong>{{ item.name }}</strong>
              <span>{{ item.code }}</span>
            </div>
            <div class="ruleset-tags">
              <ElTag size="small" effect="plain">{{ scopeText(item.scopeType) }}</ElTag>
              <ElTag size="small" effect="plain" :type="templateModeTagType(item)">
                {{ templateModeText(templateInspectionMode(item)) }}
              </ElTag>
              <ElTag
                size="small"
                effect="plain"
                :type="templateRecordAll(item) ? 'success' : 'warning'"
              >
                {{ templateRecordAll(item) ? '全量明细' : '仅异常' }}
              </ElTag>
              <ElTag v-if="item.isBuiltin" size="small" type="warning">内置</ElTag>
              <div class="inline-switch" @click.stop>
                <span :class="{ muted: !item.enabled }">{{ item.enabled ? '启用' : '禁用' }}</span>
                <ElSwitch
                  size="small"
                  :model-value="item.enabled"
                  :loading="switchingTemplateId === item.id"
                  @change="toggleTemplateEnabled(item, Boolean($event))"
                />
              </div>
            </div>
          </div>
          <p class="ruleset-desc">{{ item.description || '未填写规则库描述' }}</p>
          <div class="ruleset-stats">
            <div>
              <strong>{{ templateGroupCount(item.id) }}</strong>
              <span>分组</span>
            </div>
            <div>
              <strong>{{ templateItemCount(item.id) }}</strong>
              <span>规则</span>
            </div>
            <div>
              <strong>{{ templateTaskCount(item.id) }}</strong>
              <span>任务</span>
            </div>
          </div>
          <div class="ruleset-footer">
            <span>更新于 {{ formatTime(item.updatedAt) }}</span>
            <ElSpace class="row-actions" @click.stop>
              <ElButton link type="primary" :icon="Edit" @click="openTemplateDialog(item)">
                编辑
              </ElButton>
              <ElPopconfirm title="确认删除该规则库吗？" @confirm="removeTemplate(item)">
                <template #reference>
                  <ElButton link type="danger" :icon="Delete">删除</ElButton>
                </template>
              </ElPopconfirm>
            </ElSpace>
          </div>
        </article>
        <ElEmpty v-if="!templateLoading && templates.length === 0" description="暂无规则库" />
      </div>

      <div class="pagination-row">
        <span>共 {{ templatePagination.total }} 个规则库</span>
        <ElPagination
          v-model:current-page="templatePagination.page"
          v-model:page-size="templatePagination.pageSize"
          background
          layout="prev, pager, next"
          :total="templatePagination.total"
          @current-change="loadTemplates"
        />
      </div>
    </template>

    <template v-else>
      <div class="detail-header">
        <ElButton text :icon="ArrowLeft" @click="backToLibrary">返回规则库</ElButton>
        <div class="detail-title">
          <strong>{{ currentTemplate?.name }}</strong>
          <span>{{ currentTemplate?.code }}</span>
        </div>
        <div class="detail-actions">
          <ElTag effect="plain">{{ scopeText(currentTemplate?.scopeType) }}</ElTag>
          <div v-if="currentTemplate" class="inline-switch">
            <span :class="{ muted: !currentTemplate.enabled }">
              {{ currentTemplate.enabled ? '规则库启用' : '规则库禁用' }}
            </span>
            <ElSwitch
              size="small"
              :model-value="currentTemplate.enabled"
              :loading="switchingTemplateId === currentTemplate.id"
              @change="toggleTemplateEnabled(currentTemplate, Boolean($event))"
            />
          </div>
          <ElButton
            text
            type="primary"
            :icon="Edit"
            @click="openTemplateDialog(currentTemplate || undefined)"
          >
            编辑规则库
          </ElButton>
        </div>
      </div>

      <div class="detail-layout">
        <aside class="group-panel">
          <div class="group-summary">
            <span>当前规则库</span>
            <strong>{{ currentTemplate?.name }}</strong>
            <p>{{ currentTemplate?.description || '暂无描述' }}</p>
          </div>
          <div class="group-stats">
            <div>
              <strong>{{ currentTemplateGroups.length }}</strong>
              <span>分组</span>
            </div>
            <div>
              <strong>{{ currentTemplateItems.length }}</strong>
              <span>规则</span>
            </div>
            <div>
              <strong>{{ templateTaskCount(currentTemplateId) }}</strong>
              <span>任务</span>
            </div>
          </div>
          <div class="tree-title-row">
            <span class="tree-title">规则分组</span>
            <ElButton text type="primary" :icon="Plus" @click="openGroupDialog()">
              新建分组
            </ElButton>
          </div>
          <ElTree
            class="category-tree"
            :data="categoryTree"
            node-key="key"
            :props="treeProps"
            highlight-current
            default-expand-all
            :current-node-key="selectedCategoryKey"
            @node-click="handleCategoryClick"
          >
            <template #default="{ data }">
              <span class="tree-node" :class="{ 'is-disabled': data.group && !data.group.enabled }">
                <span class="tree-label">{{ data.label }}</span>
                <span class="tree-actions" @click.stop>
                  <em>{{ data.count }}</em>
                  <template v-if="data.group">
                    <ElSwitch
                      size="small"
                      :model-value="data.group.enabled"
                      :loading="switchingGroupId === data.group.id"
                      @change="toggleGroupEnabled(data.group, Boolean($event))"
                    />
                    <ElButton
                      link
                      type="primary"
                      :icon="Edit"
                      @click="openGroupDialog(data.group)"
                    />
                    <ElButton
                      link
                      type="primary"
                      @click="openGroupTransferDialog(data.group, 'copy')"
                    >
                      复制
                    </ElButton>
                    <ElButton
                      link
                      type="warning"
                      :disabled="Boolean(currentTemplate?.isBuiltin)"
                      @click="openGroupTransferDialog(data.group, 'move')"
                    >
                      迁移
                    </ElButton>
                    <ElPopconfirm title="确认删除该分组吗？" @confirm="removeGroup(data.group)">
                      <template #reference>
                        <ElButton link type="danger" :icon="Delete" />
                      </template>
                    </ElPopconfirm>
                  </template>
                </span>
              </span>
            </template>
          </ElTree>
        </aside>

        <section class="rules-panel">
          <div class="rules-filter">
            <ArtSearchBar
              v-show="showItemSearchBar"
              :model-value="itemFilters"
              :items="itemSearchItems"
              @search="searchItems"
              @reset="resetItemFilters"
            />
          </div>

          <div class="rules-table-wrap">
            <ArtTableHeader
              v-model:columns="itemColumns"
              v-model:showSearchBar="showItemSearchBar"
              :loading="itemLoading"
              :showZebra="true"
              :showBorder="true"
              :showHeaderBackground="true"
              :fullClass="'art-page-view'"
              :layout="'search,refresh,size,fullscreen,columns,settings'"
              @refresh="loadItems"
            >
              <template #left>
                <ElSpace wrap>
                  <span class="card-title">{{ selectedCategoryName }}</span>
                  <ElTag effect="plain" type="info"
                    >当前分组 {{ itemPagination.total }} 条规则</ElTag
                  >
                  <ElButton type="primary" :icon="Plus" @click="openItemDialog()">
                    新建规则
                  </ElButton>
                </ElSpace>
              </template>
            </ArtTableHeader>
            <ArtTable
              :loading="itemLoading"
              :data="items"
              :columns="visibleItemColumns"
              :pagination="{
                current: itemPagination.page,
                size: itemPagination.pageSize,
                total: itemPagination.total
              }"
              :pagination-options="{ align: 'right' }"
              :show-table-header="false"
              empty-text="暂无巡检规则"
              @pagination:size-change="handleItemSizeChange"
              @pagination:current-change="handleItemCurrentChange"
            >
              <template #item="{ row }">
                <div class="meta-block">
                  <strong>{{ row.itemName }}</strong>
                  <span>{{ row.category || '默认分组' }} · {{ row.itemCode }}</span>
                </div>
              </template>
              <template #checkType="{ row }">
                <ElTag effect="plain">{{ checkTypeText(row.checkType) }}</ElTag>
              </template>
              <template #targetType="{ row }">{{ targetTypeText(row.targetType) }}</template>
              <template #severity="{ row }">
                <ElTag :type="severityTagType(row.severity)">{{
                  severityText(row.severity)
                }}</ElTag>
              </template>
              <template #condition="{ row }">
                <div class="meta-block">
                  <strong>{{
                    [row.operator, row.threshold, row.unit].filter(Boolean).join(' ') || '-'
                  }}</strong>
                  <span>{{ row.promql || row.configJson || '-' }}</span>
                </div>
              </template>
              <template #enabled="{ row }">
                <div class="inline-switch">
                  <span :class="{ muted: !row.enabled }">{{ row.enabled ? '启用' : '禁用' }}</span>
                  <ElSwitch
                    size="small"
                    :model-value="row.enabled"
                    :loading="switchingItemId === row.id"
                    @change="toggleItemEnabled(row, Boolean($event))"
                  />
                </div>
              </template>
              <template #updatedAt="{ row }">{{ formatTime(row.updatedAt) }}</template>
              <template #operation="{ row }">
                <ElSpace class="row-actions">
                  <ElButton link type="primary" @click="openItemDialog(row)">编辑</ElButton>
                  <ElPopconfirm title="确认删除该规则吗？" @confirm="removeItem(row)">
                    <template #reference>
                      <ElButton link type="danger">删除</ElButton>
                    </template>
                  </ElPopconfirm>
                </ElSpace>
              </template>
            </ArtTable>
          </div>
        </section>
      </div>
    </template>

    <ElDialog
      v-model="templateDialogVisible"
      :title="templateDialogMode === 'create' ? '新建规则库' : '编辑规则库'"
      width="620px"
    >
      <ElForm :model="templateForm" label-width="100px">
        <ElFormItem label="规则库名称" required>
          <ElInput v-model="templateForm.name" placeholder="如 Kubernetes 基础巡检" />
        </ElFormItem>
        <ElFormItem label="规则库编码" required>
          <ElInput v-model="templateForm.code" placeholder="如 k8s_base" />
        </ElFormItem>
        <ElFormItem label="适用范围">
          <ElRadioGroup v-model="templateForm.scopeType">
            <ElRadioButton label="cluster">单集群</ElRadioButton>
            <ElRadioButton label="global">全局</ElRadioButton>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="巡检模式">
          <ElRadioGroup v-model="templateForm.inspectionMode">
            <ElRadioButton label="cluster">按集群</ElRadioButton>
            <ElRadioButton label="instance">按实例</ElRadioButton>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem label="保存明细">
          <ElSwitch
            v-model="templateForm.recordAllResults"
            active-text="全部结果"
            inactive-text="仅异常"
          />
        </ElFormItem>
        <ElFormItem label="启用状态">
          <ElSwitch v-model="templateForm.enabled" />
        </ElFormItem>
        <ElFormItem label="描述">
          <ElInput v-model="templateForm.description" type="textarea" :rows="3" />
        </ElFormItem>
        <ElCollapse class="rule-advanced-collapse">
          <ElCollapseItem name="template-config" title="高级配置">
            <ElFormItem label="扩展 JSON">
              <ElInput
                v-model="templateForm.configJson"
                type="textarea"
                :rows="3"
                placeholder="JSON"
              />
            </ElFormItem>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="templateDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="savingTemplate" @click="saveTemplate">保存</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="groupTransferDialogVisible"
      :title="groupTransferForm.operation === 'copy' ? '复制规则分组' : '迁移规则分组'"
      width="620px"
    >
      <ElForm :model="groupTransferForm" label-width="120px">
        <ElFormItem label="目标规则库" required>
          <ElSelect
            v-model="groupTransferForm.targetTemplateId"
            filterable
            placeholder="选择目标规则库"
          >
            <ElOption
              v-for="item in groupTransferTargetTemplates"
              :key="item.id"
              :label="item.name"
              :value="item.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="分组名称" required>
              <ElInput v-model="groupTransferForm.groupName" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="分组编码" required>
              <ElInput v-model="groupTransferForm.groupCode" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem v-if="groupTransferForm.operation === 'copy'" label="规则编码前缀">
          <ElInput v-model="groupTransferForm.itemCodePrefix" placeholder="冲突时填写，如 prod_" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="groupTransferDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="savingGroupTransfer" @click="saveGroupTransfer">
            确认
          </ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="groupDialogVisible"
      :title="groupDialogMode === 'create' ? '新建规则分组' : '编辑规则分组'"
      width="620px"
    >
      <ElForm :model="groupForm" label-width="100px">
        <ElFormItem label="分组名称" required>
          <ElInput v-model="groupForm.groupName" placeholder="如 etcd 巡检" />
        </ElFormItem>
        <ElFormItem label="分组编码" required>
          <ElInput v-model="groupForm.groupCode" placeholder="如 etcd" />
        </ElFormItem>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="启用状态">
              <ElSwitch v-model="groupForm.enabled" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="排序">
              <ElInputNumber v-model="groupForm.orderNum" :min="1" :max="9999" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="描述">
          <ElInput v-model="groupForm.description" type="textarea" :rows="3" />
        </ElFormItem>
        <ElFormItem label="扩展配置">
          <ElInput v-model="groupForm.configJson" type="textarea" :rows="3" placeholder="JSON" />
        </ElFormItem>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="groupDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="savingGroup" @click="saveGroup">保存</ElButton>
        </div>
      </template>
    </ElDialog>

    <ElDialog
      v-model="itemDialogVisible"
      :title="itemDialogMode === 'create' ? '新建巡检规则' : '编辑巡检规则'"
      width="780px"
    >
      <ElForm :model="itemForm" label-width="110px">
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="规则名称" required>
              <ElInput v-model="itemForm.itemName" placeholder="如 etcd Leader 数量" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="规则编码" required>
              <ElInput v-model="itemForm.itemCode" placeholder="如 etcd_leader_count" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="12">
            <ElFormItem label="规则分组">
              <ElSelect
                v-model="itemForm.groupId"
                clearable
                filterable
                placeholder="选择分组"
                @change="handleItemGroupChange"
              >
                <ElOption
                  v-for="group in currentTemplateGroups"
                  :key="group.id"
                  :label="group.groupName"
                  :value="group.id"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="12">
            <ElFormItem label="检查类型">
              <ElSelect v-model="itemForm.checkType">
                <ElOption
                  v-for="item in checkTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="8">
            <ElFormItem label="目标类型">
              <ElSelect v-model="itemForm.targetType">
                <ElOption
                  v-for="item in targetTypeOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="严重级别">
              <ElSelect v-model="itemForm.severity">
                <ElOption
                  v-for="item in severityOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="启用状态">
              <ElSwitch v-model="itemForm.enabled" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="8">
            <ElFormItem label="权重">
              <ElInputNumber v-model="itemForm.weight" :min="1" :max="100" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="超时秒数">
              <ElInputNumber v-model="itemForm.timeoutSec" :min="1" :max="600" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="排序">
              <ElInputNumber v-model="itemForm.orderNum" :min="1" :max="9999" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElRow :gutter="16">
          <ElCol :span="8">
            <ElFormItem label="运算符">
              <ElSelect v-model="itemForm.operator" clearable>
                <ElOption
                  v-for="item in operatorOptions"
                  :key="item.value"
                  :label="item.label"
                  :value="item.value"
                />
              </ElSelect>
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="阈值">
              <ElInput v-model="itemForm.threshold" placeholder="如 1、80、true" />
            </ElFormItem>
          </ElCol>
          <ElCol :span="8">
            <ElFormItem label="单位">
              <ElInput v-model="itemForm.unit" placeholder="%、个、ms" />
            </ElFormItem>
          </ElCol>
        </ElRow>
        <ElFormItem label="PromQL">
          <ElInput
            v-model="itemForm.promql"
            type="textarea"
            :rows="4"
            placeholder="三级巡检表达式，例如 sum(up{job='etcd'} == 1)"
          />
        </ElFormItem>
        <ElFormItem label="处理建议">
          <ElInput v-model="itemForm.advice" type="textarea" :rows="3" />
        </ElFormItem>
        <ElCollapse v-model="itemAdvancedPanels" class="rule-advanced-collapse">
          <ElCollapseItem name="advanced" title="高级配置">
            <ElFormItem label="目标展开 PromQL">
              <ElInput
                v-model="itemForm.detailPromql"
                type="textarea"
                :rows="3"
                placeholder="为空时自动按 PromQL 推导"
              />
            </ElFormItem>
            <ElFormItem label="对象名 Label">
              <ElSelect
                v-model="itemForm.targetLabels"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="instance、pod、endpoint"
              >
                <ElOption v-for="item in labelOptions" :key="item" :label="item" :value="item" />
              </ElSelect>
            </ElFormItem>
            <ElRow :gutter="16">
              <ElCol :span="8">
                <ElFormItem label="明细运算符">
                  <ElSelect v-model="itemForm.detailOperator" clearable placeholder="默认自动">
                    <ElOption
                      v-for="item in operatorOptions"
                      :key="item.value"
                      :label="item.label"
                      :value="item.value"
                    />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
              <ElCol :span="8">
                <ElFormItem label="明细阈值">
                  <ElInput v-model="itemForm.detailThreshold" placeholder="默认自动" />
                </ElFormItem>
              </ElCol>
              <ElCol :span="8">
                <ElFormItem label="明细期望">
                  <ElInput v-model="itemForm.detailExpected" placeholder="默认 运算符 + 阈值" />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElFormItem label="扩展 JSON">
              <ElInput v-model="itemForm.configJson" type="textarea" :rows="3" placeholder="JSON" />
            </ElFormItem>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
      <template #footer>
        <div class="dialog-footer-actions">
          <ElButton @click="itemDialogVisible = false">取消</ElButton>
          <ElButton type="primary" :loading="savingItem" @click="saveItem">保存</ElButton>
        </div>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, nextTick, onMounted, reactive, ref } from 'vue'
  import { ElMessage } from 'element-plus'
  import { ArrowLeft, Delete, Edit, Plus } from '@element-plus/icons-vue'
  import {
    createInspectionGroupApi,
    createInspectionItemApi,
    createInspectionTemplateApi,
    deleteInspectionGroupApi,
    deleteInspectionItemApi,
    deleteInspectionTemplateApi,
    searchInspectionGroupsApi,
    searchInspectionItemsApi,
    searchInspectionTasksApi,
    searchInspectionTemplatesApi,
    transferInspectionGroupApi,
    updateInspectionGroupApi,
    updateInspectionItemApi,
    updateInspectionTemplateApi,
    type InspectionGroup,
    type InspectionItem,
    type InspectionTask,
    type InspectionTemplate
  } from '@/api/manager/inspection'
  import {
    checkTypeOptions,
    checkTypeText,
    formatTime,
    operatorOptions,
    scopeText,
    severityOptions,
    severityTagType,
    severityText,
    targetTypeOptions,
    targetTypeText
  } from '../shared/constants'

  defineOptions({ name: 'InspectionRuleLibrary' })

  interface CategoryTreeNode {
    key: string
    label: string
    category: string
    count: number
    groupId: number
    group?: InspectionGroup
  }

  const metricPageSize = 5000
  const treeProps = { label: 'label' }

  const templateLoading = ref(false)
  const metricLoading = ref(false)
  const itemLoading = ref(false)
  const showItemSearchBar = ref(true)
  const savingTemplate = ref(false)
  const savingGroup = ref(false)
  const savingGroupTransfer = ref(false)
  const savingItem = ref(false)
  const switchingTemplateId = ref(0)
  const switchingGroupId = ref(0)
  const switchingItemId = ref(0)
  const templates = ref<InspectionTemplate[]>([])
  const allTemplates = ref<InspectionTemplate[]>([])
  const allGroups = ref<InspectionGroup[]>([])
  const allItems = ref<InspectionItem[]>([])
  const allTasks = ref<InspectionTask[]>([])
  const items = ref<InspectionItem[]>([])
  const currentTemplateId = ref<number>(0)
  const selectedGroupId = ref<number>(0)
  const selectedCategory = ref('')
  const itemColumns = ref([
    { prop: 'item', label: '规则名称', minWidth: 240, useSlot: true },
    { prop: 'checkType', label: '类型', width: 120, useSlot: true },
    { prop: 'targetType', label: '目标', width: 120, useSlot: true },
    { prop: 'severity', label: '级别', width: 100, useSlot: true },
    { prop: 'condition', label: '判定条件', minWidth: 260, useSlot: true },
    { prop: 'weight', label: '权重', width: 80 },
    { prop: 'enabled', label: '状态', width: 120, useSlot: true },
    { prop: 'updatedAt', label: '更新时间', width: 180, useSlot: true },
    { prop: 'operation', label: '操作', fixed: 'right' as const, width: 150, useSlot: true }
  ])
  const visibleItemColumns = computed(() =>
    itemColumns.value.filter((column: any) => column.visible !== false)
  )

  const templatePagination = reactive({ page: 1, pageSize: 12, total: 0 })
  const itemPagination = reactive({ page: 1, pageSize: 20, total: 0 })
  const templateFilters = reactive({ name: '', scopeType: '' })
  const itemFilters = reactive<{ checkType: string; enabled?: boolean }>({
    checkType: '',
    enabled: undefined
  })
  const templateSearchItems = computed(() => [
    {
      label: '规则库',
      key: 'name',
      type: 'input',
      props: {
        clearable: true,
        placeholder: '输入名称查询'
      }
    },
    {
      label: '范围',
      key: 'scopeType',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部范围',
        options: [
          { label: '全局巡检', value: 'global' },
          { label: '单集群', value: 'cluster' }
        ]
      }
    }
  ])
  const itemSearchItems = computed(() => [
    {
      label: '检查类型',
      key: 'checkType',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部类型',
        options: checkTypeOptions
      }
    },
    {
      label: '状态',
      key: 'enabled',
      type: 'select',
      props: {
        clearable: true,
        placeholder: '全部状态',
        options: [
          { label: '启用', value: true },
          { label: '禁用', value: false }
        ]
      }
    }
  ])

  const currentTemplate = computed(
    () => templates.value.find((item) => item.id === currentTemplateId.value) || null
  )
  const currentTemplateItems = computed(() =>
    allItems.value.filter((item) => item.templateId === currentTemplateId.value)
  )
  const currentTemplateGroups = computed(() => templateGroups(currentTemplateId.value))
  const groupTransferTargetTemplates = computed(() =>
    (allTemplates.value.length ? allTemplates.value : templates.value).filter(
      (item) => item.id !== currentTemplateId.value
    )
  )
  const selectedCategoryKey = computed(() =>
    selectedGroupId.value > 0
      ? groupKey(selectedGroupId.value)
      : categoryKey(selectedCategory.value)
  )
  const selectedCategoryName = computed(() => {
    if (selectedGroupId.value > 0) {
      return (
        currentTemplateGroups.value.find((group) => group.id === selectedGroupId.value)
          ?.groupName || '全部分组'
      )
    }
    return selectedCategory.value || '全部分组'
  })
  const categoryTree = computed<CategoryTreeNode[]>(() => {
    const groupedNames = new Set(currentTemplateGroups.value.map((group) => group.groupName))
    const orphanNames = groupNames(
      currentTemplateItems.value.filter(
        (item) => !item.groupId && !groupedNames.has(categoryName(item.category))
      )
    )

    return [
      {
        key: categoryKey(''),
        label: '全部分组',
        category: '',
        count: currentTemplateItems.value.length,
        groupId: 0
      },
      ...currentTemplateGroups.value.map((group) => ({
        key: groupKey(group.id),
        label: group.groupName,
        category: group.groupName,
        count: currentTemplateItems.value.filter(
          (item) => item.groupId === group.id || categoryName(item.category) === group.groupName
        ).length,
        groupId: group.id,
        group
      })),
      ...orphanNames.map((name) => ({
        key: categoryKey(name),
        label: name,
        category: name === '默认分组' ? '' : name,
        count: currentTemplateItems.value.filter((item) => categoryName(item.category) === name)
          .length,
        groupId: 0
      }))
    ]
  })

  const templateDialogVisible = ref(false)
  const templateDialogMode = ref<'create' | 'edit'>('create')
  const templateForm = reactive({
    id: 0,
    name: '',
    code: '',
    description: '',
    scopeType: 'cluster',
    enabled: true,
    inspectionMode: 'cluster',
    recordAllResults: true,
    configJson: ''
  })

  const itemDialogVisible = ref(false)
  const itemDialogMode = ref<'create' | 'edit'>('create')
  const itemAdvancedPanels = ref<string[]>([])
  const groupDialogVisible = ref(false)
  const groupDialogMode = ref<'create' | 'edit'>('create')
  const groupTransferDialogVisible = ref(false)
  const groupForm = reactive({
    id: 0,
    templateId: 0,
    groupCode: '',
    groupName: '',
    description: '',
    enabled: true,
    orderNum: 100,
    configJson: ''
  })
  const groupTransferForm = reactive({
    sourceGroupId: 0,
    operation: 'copy' as 'copy' | 'move',
    targetTemplateId: 0,
    groupCode: '',
    groupName: '',
    itemCodePrefix: ''
  })
  const itemForm = reactive({
    id: 0,
    templateId: 0,
    groupId: 0,
    itemCode: '',
    itemName: '',
    category: '',
    checkType: 'builtin',
    targetType: 'cluster',
    severity: 'warning',
    weight: 10,
    enabled: true,
    timeoutSec: 30,
    orderNum: 100,
    promql: '',
    operator: '',
    threshold: '',
    unit: '',
    detailPromql: '',
    targetKind: '',
    targetLabels: [] as string[],
    detailOperator: '',
    detailThreshold: '',
    detailExpected: '',
    configJson: '',
    advice: ''
  })
  const labelOptions = [
    'instance',
    'node',
    'pod',
    'namespace',
    'endpoint',
    'service',
    'job',
    'device',
    'mountpoint',
    'container',
    'database',
    'cluster',
    'queue',
    'topic'
  ]

  async function loadTemplates() {
    templateLoading.value = true
    try {
      const resp = await searchInspectionTemplatesApi({
        page: templatePagination.page,
        pageSize: templatePagination.pageSize,
        orderField: 'id',
        isAsc: false,
        name: templateFilters.name || undefined,
        scopeType: templateFilters.scopeType || undefined
      })
      templates.value = resp.data || []
      templatePagination.total = resp.total || 0
      if (
        currentTemplateId.value &&
        !templates.value.some((item) => item.id === currentTemplateId.value)
      ) {
        currentTemplateId.value = 0
        items.value = []
        itemPagination.total = 0
      }
    } finally {
      templateLoading.value = false
    }
  }

  async function loadOverviewMetrics() {
    metricLoading.value = true
    try {
      const [templateResp, groupResp, itemResp, taskResp] = await Promise.all([
        searchInspectionTemplatesApi({
          page: 1,
          pageSize: metricPageSize,
          orderField: 'id',
          isAsc: false
        }),
        searchInspectionGroupsApi({
          page: 1,
          pageSize: metricPageSize,
          orderField: 'order_num',
          isAsc: true
        }),
        searchInspectionItemsApi({
          page: 1,
          pageSize: metricPageSize,
          orderField: 'order_num',
          isAsc: true
        }),
        searchInspectionTasksApi({
          page: 1,
          pageSize: metricPageSize,
          orderField: 'id',
          isAsc: false
        })
      ])
      allTemplates.value = templateResp.data || []
      allGroups.value = groupResp.data || []
      allItems.value = itemResp.data || []
      allTasks.value = taskResp.data || []
    } finally {
      metricLoading.value = false
    }
  }

  async function loadItems() {
    if (!currentTemplateId.value) {
      items.value = []
      itemPagination.total = 0
      return
    }
    itemLoading.value = true
    try {
      const resp = await searchInspectionItemsApi({
        templateId: currentTemplateId.value,
        page: itemPagination.page,
        pageSize: itemPagination.pageSize,
        orderField: 'order_num',
        isAsc: true,
        groupId: selectedGroupId.value || undefined,
        category: selectedGroupId.value ? undefined : selectedCategory.value || undefined,
        checkType: itemFilters.checkType || undefined,
        enabled: itemFilters.enabled
      })
      items.value = resp.data || []
      itemPagination.total = resp.total || 0
    } finally {
      itemLoading.value = false
    }
  }

  async function refreshData() {
    await Promise.all([loadTemplates(), loadOverviewMetrics()])
    if (currentTemplateId.value) {
      await loadItems()
    }
  }

  function searchTemplates() {
    templatePagination.page = 1
    loadTemplates()
  }

  function resetTemplateFilters() {
    Object.assign(templateFilters, {
      name: '',
      scopeType: ''
    })
    searchTemplates()
  }

  function searchItems() {
    itemPagination.page = 1
    loadItems()
  }

  function resetItemFilters() {
    Object.assign(itemFilters, {
      checkType: '',
      enabled: undefined
    })
    searchItems()
  }

  async function enterTemplate(row: InspectionTemplate) {
    currentTemplateId.value = row.id
    selectedGroupId.value = 0
    selectedCategory.value = ''
    itemPagination.page = 1
    itemFilters.checkType = ''
    itemFilters.enabled = undefined
    await loadItems()
    await nextTick()
  }

  function backToLibrary() {
    currentTemplateId.value = 0
    selectedGroupId.value = 0
    selectedCategory.value = ''
    items.value = []
    itemPagination.total = 0
  }

  function handleCategoryClick(node: CategoryTreeNode) {
    selectedGroupId.value = node.groupId || 0
    selectedCategory.value = node.groupId ? '' : node.category
    itemPagination.page = 1
    loadItems()
  }

  function handleItemSizeChange(size: number) {
    itemPagination.pageSize = size
    itemPagination.page = 1
    loadItems()
  }

  function handleItemCurrentChange(page: number) {
    itemPagination.page = page
    loadItems()
  }

  function openTemplateDialog(row?: InspectionTemplate) {
    const config = parseTemplateConfig(row?.configJson || '')
    templateDialogMode.value = row ? 'edit' : 'create'
    Object.assign(templateForm, {
      id: row?.id || 0,
      name: row?.name || '',
      code: row?.code || '',
      description: row?.description || '',
      scopeType: row?.scopeType || 'cluster',
      enabled: row?.enabled ?? true,
      inspectionMode: config.inspectionMode,
      recordAllResults: config.recordAllResults,
      configJson: row?.configJson || ''
    })
    templateDialogVisible.value = true
  }

  function parseTemplateConfig(raw: string) {
    try {
      const parsed = JSON.parse(raw || '{}')
      const inspectionMode = parsed.inspectionMode === 'instance' ? 'instance' : 'cluster'
      return {
        inspectionMode,
        recordAllResults: parsed.recordAllResults !== false
      }
    } catch {
      return {
        inspectionMode: 'cluster',
        recordAllResults: true
      }
    }
  }

  function buildTemplateConfigJson() {
    let config: Record<string, any> = {}
    try {
      config = templateForm.configJson ? JSON.parse(templateForm.configJson) : {}
    } catch {
      config = {}
    }
    config.inspectionMode = templateForm.inspectionMode
    config.recordAllResults = Boolean(templateForm.recordAllResults)
    templateForm.configJson = JSON.stringify(config)
    return templateForm.configJson
  }

  async function saveTemplate() {
    if (!templateForm.name || !templateForm.code) {
      ElMessage.warning('规则库名称和编码不能为空')
      return
    }
    savingTemplate.value = true
    try {
      const payload = {
        name: templateForm.name,
        code: templateForm.code,
        description: templateForm.description,
        scopeType: templateForm.scopeType,
        enabled: templateForm.enabled,
        configJson: buildTemplateConfigJson()
      }
      if (templateDialogMode.value === 'edit') {
        await updateInspectionTemplateApi(templateForm.id, payload)
      } else {
        await createInspectionTemplateApi(payload)
      }
      ElMessage.success('保存成功')
      templateDialogVisible.value = false
      await refreshData()
    } finally {
      savingTemplate.value = false
    }
  }

  async function removeTemplate(row: InspectionTemplate) {
    await deleteInspectionTemplateApi(row.id)
    ElMessage.success('删除成功')
    if (currentTemplateId.value === row.id) {
      backToLibrary()
    }
    await refreshData()
  }

  function templateInspectionMode(row?: InspectionTemplate | null) {
    return parseTemplateConfig(row?.configJson || '').inspectionMode
  }

  function templateRecordAll(row?: InspectionTemplate | null) {
    return parseTemplateConfig(row?.configJson || '').recordAllResults
  }

  function templateModeText(value: string) {
    return value === 'instance' ? '按实例' : '按集群'
  }

  function templateModeTagType(row: InspectionTemplate) {
    return templateInspectionMode(row) === 'instance' ? 'success' : 'info'
  }

  function openGroupDialog(row?: InspectionGroup) {
    if (!currentTemplateId.value) return
    groupDialogMode.value = row ? 'edit' : 'create'
    Object.assign(groupForm, {
      id: row?.id || 0,
      templateId: row?.templateId || currentTemplateId.value,
      groupCode: row?.groupCode || '',
      groupName: row?.groupName || '',
      description: row?.description || '',
      enabled: row?.enabled ?? true,
      orderNum: row?.orderNum || 100,
      configJson: row?.configJson || ''
    })
    groupDialogVisible.value = true
  }

  async function saveGroup() {
    if (!groupForm.groupName || !groupForm.groupCode) {
      ElMessage.warning('分组名称和编码不能为空')
      return
    }
    savingGroup.value = true
    try {
      const payload = {
        templateId: groupForm.templateId,
        groupCode: groupForm.groupCode,
        groupName: groupForm.groupName,
        description: groupForm.description,
        enabled: groupForm.enabled,
        orderNum: groupForm.orderNum,
        configJson: groupForm.configJson
      }
      if (groupDialogMode.value === 'edit') {
        await updateInspectionGroupApi(groupForm.id, payload)
      } else {
        await createInspectionGroupApi(payload)
      }
      ElMessage.success('保存成功')
      groupDialogVisible.value = false
      await Promise.all([loadOverviewMetrics(), loadItems()])
    } finally {
      savingGroup.value = false
    }
  }

  async function removeGroup(row: InspectionGroup) {
    await deleteInspectionGroupApi(row.id)
    ElMessage.success('删除成功')
    if (selectedGroupId.value === row.id) {
      selectedGroupId.value = 0
      selectedCategory.value = ''
    }
    await Promise.all([loadOverviewMetrics(), loadItems()])
  }

  function openGroupTransferDialog(row: InspectionGroup, operation: 'copy' | 'move') {
    if (operation === 'move' && currentTemplate.value?.isBuiltin) {
      ElMessage.warning('内置规则库分组不允许迁移')
      return
    }
    const target = groupTransferTargetTemplates.value[0]
    Object.assign(groupTransferForm, {
      sourceGroupId: row.id,
      operation,
      targetTemplateId: target?.id || 0,
      groupCode: operation === 'copy' ? `${row.groupCode}_copy` : row.groupCode,
      groupName: operation === 'copy' ? `${row.groupName}副本` : row.groupName,
      itemCodePrefix: operation === 'copy' ? `${row.groupCode}_` : ''
    })
    groupTransferDialogVisible.value = true
  }

  async function saveGroupTransfer() {
    if (!groupTransferForm.sourceGroupId || !groupTransferForm.targetTemplateId) {
      ElMessage.warning('请选择源分组和目标规则库')
      return
    }
    if (!groupTransferForm.groupCode || !groupTransferForm.groupName) {
      ElMessage.warning('分组名称和编码不能为空')
      return
    }
    savingGroupTransfer.value = true
    try {
      const resp = await transferInspectionGroupApi(groupTransferForm.sourceGroupId, {
        targetTemplateId: groupTransferForm.targetTemplateId,
        groupCode: groupTransferForm.groupCode,
        groupName: groupTransferForm.groupName,
        itemCodePrefix: groupTransferForm.itemCodePrefix,
        operation: groupTransferForm.operation
      })
      ElMessage.success(
        groupTransferForm.operation === 'copy'
          ? `复制成功，生成 ${resp.itemCount || 0} 条规则`
          : `迁移成功，处理 ${resp.itemCount || 0} 条规则`
      )
      groupTransferDialogVisible.value = false
      await Promise.all([loadOverviewMetrics(), loadItems()])
    } finally {
      savingGroupTransfer.value = false
    }
  }

  function openItemDialog(row?: InspectionItem) {
    if (!currentTemplateId.value) return
    const groupId = row?.groupId || selectedGroupId.value || 0
    const group = currentTemplateGroups.value.find((item) => item.id === groupId)
    const ruleConfig = parseRuleConfig(row?.configJson || '')
    const targetType = row?.targetType || 'cluster'
    itemDialogMode.value = row ? 'edit' : 'create'
    Object.assign(itemForm, {
      id: row?.id || 0,
      templateId: row?.templateId || currentTemplateId.value,
      groupId,
      itemCode: row?.itemCode || '',
      itemName: row?.itemName || '',
      category: group?.groupName || row?.category || selectedCategory.value || '',
      checkType: row?.checkType || 'builtin',
      targetType,
      severity: row?.severity || 'warning',
      weight: row?.weight || 10,
      enabled: row?.enabled ?? true,
      timeoutSec: row?.timeoutSec || 30,
      orderNum: row?.orderNum || 100,
      promql: row?.promql || '',
      operator: row?.operator || '',
      threshold: row?.threshold || '',
      unit: row?.unit || '',
      detailPromql: ruleConfig.detailPromql || '',
      targetKind: ruleConfig.targetKind || '',
      targetLabels: ruleConfig.targetLabels.length
        ? ruleConfig.targetLabels
        : defaultTargetLabels(targetType),
      detailOperator: ruleConfig.detailOperator || '',
      detailThreshold: ruleConfig.detailThreshold || '',
      detailExpected: ruleConfig.detailExpected || '',
      configJson: row?.configJson || '',
      advice: row?.advice || ''
    })
    itemAdvancedPanels.value = []
    itemDialogVisible.value = true
  }

  function handleItemGroupChange(value: number | string | undefined) {
    const groupId = Number(value || 0)
    itemForm.groupId = groupId
    const group = currentTemplateGroups.value.find((item) => item.id === groupId)
    itemForm.category = group?.groupName || ''
  }

  function parseRuleConfig(raw: string) {
    try {
      const parsed = JSON.parse(raw || '{}')
      return {
        detailPromql: String(parsed.detailPromql || ''),
        targetKind: String(parsed.targetKind || ''),
        targetLabels: normalizeStringList(parsed.targetLabels),
        detailOperator: String(parsed.detailOperator || ''),
        detailThreshold: String(parsed.detailThreshold || ''),
        detailExpected: String(parsed.detailExpected || '')
      }
    } catch {
      return {
        detailPromql: '',
        targetKind: '',
        targetLabels: [] as string[],
        detailOperator: '',
        detailThreshold: '',
        detailExpected: ''
      }
    }
  }

  function buildRuleConfigJson() {
    let config: Record<string, any> = {}
    try {
      config = itemForm.configJson ? JSON.parse(itemForm.configJson) : {}
    } catch {
      config = {}
    }
    const detailPromql = itemForm.detailPromql.trim()
    const targetKind = itemForm.targetKind.trim()
    if (detailPromql) {
      config.detailPromql = detailPromql
    } else {
      delete config.detailPromql
    }
    if (targetKind) {
      config.targetKind = targetKind
    } else {
      delete config.targetKind
    }
    const targetLabels = normalizeStringList(itemForm.targetLabels)
    if (targetLabels.length) {
      config.targetLabels = targetLabels
    } else {
      delete config.targetLabels
    }
    const detailOperator = itemForm.detailOperator.trim()
    const detailThreshold = itemForm.detailThreshold.trim()
    const detailExpected = itemForm.detailExpected.trim()
    if (detailOperator && detailThreshold) {
      config.detailOperator = detailOperator
      config.detailThreshold = detailThreshold
    } else {
      delete config.detailOperator
      delete config.detailThreshold
    }
    if (detailExpected) {
      config.detailExpected = detailExpected
    } else {
      delete config.detailExpected
    }
    delete config.keepLabels
    itemForm.configJson = JSON.stringify(config)
    return itemForm.configJson
  }

  function normalizeStringList(value: unknown): string[] {
    if (!Array.isArray(value)) return []
    return Array.from(new Set(value.map((item) => String(item || '').trim()).filter(Boolean)))
  }

  function defaultTargetLabels(targetType: string) {
    if (targetType === 'node' || targetType === 'host') return ['node', 'instance']
    if (targetType === 'pod') return ['namespace', 'pod']
    if (targetType === 'workload')
      return ['namespace', 'deployment', 'statefulset', 'daemonset', 'job']
    if (targetType === 'database' || targetType === 'middleware') {
      return ['instance', 'endpoint', 'namespace', 'pod', 'service', 'job']
    }
    return ['instance', 'endpoint', 'pod', 'service', 'job']
  }

  async function saveItem() {
    if (!itemForm.itemName || !itemForm.itemCode) {
      ElMessage.warning('规则名称和编码不能为空')
      return
    }
    savingItem.value = true
    try {
      const payload = {
        templateId: itemForm.templateId,
        groupId: itemForm.groupId,
        itemCode: itemForm.itemCode,
        itemName: itemForm.itemName,
        category: itemForm.category,
        checkType: itemForm.checkType,
        targetType: itemForm.targetType,
        severity: itemForm.severity,
        weight: itemForm.weight,
        enabled: itemForm.enabled,
        timeoutSec: itemForm.timeoutSec,
        orderNum: itemForm.orderNum,
        promql: itemForm.promql,
        operator: itemForm.operator,
        threshold: itemForm.threshold,
        unit: itemForm.unit,
        configJson: buildRuleConfigJson(),
        advice: itemForm.advice
      }
      if (itemDialogMode.value === 'edit') {
        await updateInspectionItemApi(itemForm.id, payload)
      } else {
        await createInspectionItemApi(payload)
      }
      ElMessage.success('保存成功')
      itemDialogVisible.value = false
      await Promise.all([loadOverviewMetrics(), loadItems()])
    } finally {
      savingItem.value = false
    }
  }

  async function removeItem(row: InspectionItem) {
    await deleteInspectionItemApi(row.id)
    ElMessage.success('删除成功')
    await Promise.all([loadOverviewMetrics(), loadItems()])
  }

  async function toggleTemplateEnabled(row: InspectionTemplate, enabled: boolean) {
    const oldEnabled = row.enabled
    row.enabled = enabled
    switchingTemplateId.value = row.id
    try {
      await updateInspectionTemplateApi(row.id, {
        name: row.name,
        code: row.code,
        description: row.description,
        scopeType: row.scopeType,
        enabled,
        configJson: row.configJson
      })
      ElMessage.success(enabled ? '规则库已启用' : '规则库已禁用')
    } catch {
      row.enabled = oldEnabled
    } finally {
      switchingTemplateId.value = 0
    }
  }

  async function toggleGroupEnabled(row: InspectionGroup, enabled: boolean) {
    const oldEnabled = row.enabled
    row.enabled = enabled
    switchingGroupId.value = row.id
    try {
      await updateInspectionGroupApi(row.id, {
        templateId: row.templateId,
        groupCode: row.groupCode,
        groupName: row.groupName,
        description: row.description,
        enabled,
        orderNum: row.orderNum,
        configJson: row.configJson
      })
      ElMessage.success(enabled ? '分组已启用' : '分组已禁用')
    } catch {
      row.enabled = oldEnabled
    } finally {
      switchingGroupId.value = 0
    }
  }

  async function toggleItemEnabled(row: InspectionItem, enabled: boolean) {
    const oldEnabled = row.enabled
    row.enabled = enabled
    switchingItemId.value = row.id
    try {
      await updateInspectionItemApi(row.id, {
        templateId: row.templateId,
        groupId: row.groupId,
        itemCode: row.itemCode,
        itemName: row.itemName,
        category: row.category,
        checkType: row.checkType,
        targetType: row.targetType,
        severity: row.severity,
        weight: row.weight,
        enabled,
        timeoutSec: row.timeoutSec,
        orderNum: row.orderNum,
        promql: row.promql,
        operator: row.operator,
        threshold: row.threshold,
        unit: row.unit,
        configJson: row.configJson,
        advice: row.advice
      })
      const cached = allItems.value.find((item) => item.id === row.id)
      if (cached) cached.enabled = enabled
      ElMessage.success(enabled ? '规则已启用' : '规则已禁用')
    } catch {
      row.enabled = oldEnabled
      const cached = allItems.value.find((item) => item.id === row.id)
      if (cached) cached.enabled = oldEnabled
    } finally {
      switchingItemId.value = 0
    }
  }

  function templateItems(templateId: number) {
    return allItems.value.filter((item) => item.templateId === templateId)
  }

  function templateGroups(templateId: number) {
    return allGroups.value
      .filter((item) => item.templateId === templateId)
      .sort((a, b) => a.orderNum - b.orderNum)
  }

  function templateItemCount(templateId: number) {
    return templateItems(templateId).length
  }

  function templateGroupCount(templateId: number) {
    return templateGroups(templateId).length || groupNames(templateItems(templateId)).length
  }

  function templateTaskCount(templateId: number) {
    return allTasks.value.filter((item) => item.templateId === templateId).length
  }

  function groupNames(source: InspectionItem[]) {
    return Array.from(new Set(source.map((item) => categoryName(item.category)))).sort()
  }

  function categoryName(value?: string) {
    return value?.trim() || '默认分组'
  }

  function categoryKey(value: string) {
    return `category:${value || '__all__'}`
  }

  function groupKey(value: number) {
    return `group:${value}`
  }

  onMounted(() => {
    refreshData()
  })
</script>

<style scoped lang="scss">
  @use '../shared/polling-ui' as *;

  .ruleset-footer,
  .detail-header,
  .detail-actions,
  .rules-panel-head {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .filter-bar,
  .rules-filter {
    padding: 16px;
  }

  .inspection-page.is-detail {
    display: flex;
    flex-direction: column;
    height: calc(100vh - 176px);
    min-height: 0;
    overflow: hidden;
  }

  .filter-bar {
    display: flex;
    gap: 12px;
    align-items: flex-start;
    justify-content: space-between;

    :deep(.art-search-bar) {
      flex: 1;
      min-width: 0;
    }

    > .el-button {
      flex: none;
      margin-top: 15px;
    }
  }

  .ruleset-grid {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 12px;
    align-items: start;
    min-height: 260px;
  }

  .ruleset-card {
    min-height: 206px;
    padding: 14px;
    cursor: pointer;
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    box-shadow: 0 1px 2px rgb(15 23 42 / 3%);
    transition:
      border-color 0.16s ease,
      box-shadow 0.16s ease,
      transform 0.16s ease;

    &:hover {
      border-color: var(--el-color-primary-light-6);
      box-shadow: 0 8px 18px rgb(15 23 42 / 6%);
      transform: translateY(-1px);
    }
  }

  .ruleset-top,
  .ruleset-footer {
    display: flex;
    gap: 12px;
    align-items: center;
    justify-content: space-between;
  }

  .ruleset-title {
    min-width: 0;

    strong {
      display: block;
      overflow: hidden;
      font-size: 14px;
      text-overflow: ellipsis;
      white-space: nowrap;
    }
  }

  .ruleset-title span {
    display: block;
    margin-top: 4px;
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ruleset-tags {
    display: flex;
    flex: none;
    flex-wrap: wrap;
    gap: 6px;
    justify-content: flex-end;
    max-width: 190px;
  }

  .ruleset-desc {
    display: -webkit-box;
    min-height: 38px;
    margin: 12px 0;
    overflow: hidden;
    font-size: 13px;
    line-height: 1.45;
    color: var(--el-text-color-secondary);
    -webkit-box-orient: vertical;
    -webkit-line-clamp: 2;
  }

  .ruleset-stats,
  .group-stats {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;

    div {
      padding: 8px 9px;
      background: var(--el-fill-color-extra-light);
      border: 1px solid var(--el-border-color-extra-light);
      border-radius: 8px;
    }

    strong,
    span {
      display: block;
    }

    strong {
      font-size: 17px;
      line-height: 1;
    }

    span {
      margin-top: 6px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .ruleset-footer {
    padding-top: 12px;
    margin-top: 12px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    border-top: 1px solid var(--el-border-color-extra-light);
  }

  .detail-header {
    flex: none;
    padding: 8px 10px;
    margin-bottom: 10px;
  }

  .inline-switch {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    min-height: 24px;
    font-size: 12px;
    color: var(--el-color-success);

    .muted {
      color: var(--el-text-color-secondary);
    }
  }

  .detail-title {
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
    }

    span {
      margin-top: 3px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
    }
  }

  .detail-layout {
    display: grid;
    flex: 1;
    grid-template-columns: 300px minmax(0, 1fr);
    gap: 12px;
    align-items: stretch;
    height: auto;
    min-height: 0;
    overflow: hidden;
  }

  .group-panel,
  .rules-panel {
    min-height: 0;
    padding: 12px;
    overflow: hidden;
  }

  .group-panel {
    display: flex;
    flex-direction: column;
  }

  .group-summary {
    span,
    p {
      font-size: 13px;
      color: var(--el-text-color-secondary);
    }

    strong {
      display: block;
      margin: 5px 0;
      font-size: 15px;
    }

    p {
      display: -webkit-box;
      margin: 0 0 10px;
      overflow: hidden;
      line-height: 1.5;
      -webkit-box-orient: vertical;
      -webkit-line-clamp: 3;
    }
  }

  .tree-title-row {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    margin: 12px 0 6px;
  }

  .tree-title {
    font-size: 13px;
    font-weight: 700;
  }

  .category-tree {
    flex: 1;
    min-height: 0;
    padding-right: 2px;
    overflow: auto;

    :deep(.el-tree-node__content) {
      height: 34px;
      border-radius: 6px;
    }
  }

  .tree-node {
    display: flex;
    gap: 8px;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    min-width: 0;

    &.is-disabled .tree-label {
      color: var(--el-text-color-secondary);
    }

    .tree-label {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .tree-actions {
      display: inline-flex;
      flex: none;
      gap: 6px;
      align-items: center;
      justify-content: flex-end;
    }

    em {
      flex: none;
      font-size: 12px;
      font-style: normal;
      color: var(--el-text-color-secondary);
    }
  }

  .rules-panel-head {
    flex: none;
    margin-bottom: 10px;
  }

  .rules-panel {
    display: flex;
    flex-direction: column;
  }

  .rules-filter,
  .pagination-row {
    flex: none;
  }

  .rules-filter {
    padding: 12px;
    margin-bottom: 10px;
  }

  .rules-table-wrap {
    flex: 1;
    min-height: 0;
    overflow: hidden;
  }

  .rules-table-wrap :deep(.el-table) {
    height: 100%;
  }

  .rules-table-wrap :deep(.el-table__body-wrapper) {
    overflow-y: auto;
  }

  .rule-advanced-collapse {
    margin-bottom: 16px;
    border-top: 1px solid var(--el-border-color-lighter);
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  @media (width <= 1500px) {
    .ruleset-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @media (width <= 1200px) {
    .inspection-page.is-detail {
      height: auto;
      overflow: visible;
    }

    .detail-layout {
      grid-template-columns: 1fr;
      height: auto;
      max-height: none;
      overflow: visible;
    }

    .group-panel,
    .rules-panel {
      min-height: 360px;
    }

    .detail-header {
      flex-direction: column;
      align-items: stretch;
    }

    .detail-header > .el-button,
    .detail-actions {
      align-self: flex-end;
    }

    .detail-actions {
      justify-content: flex-end;
    }
  }

  @media (width <= 1100px) {
    .ruleset-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (width <= 760px) {
    .ruleset-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
