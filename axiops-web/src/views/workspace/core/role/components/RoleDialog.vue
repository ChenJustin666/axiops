<template>
  <ElDialog
    :model-value="visible"
    :title="dialogType === 'add' ? '创建 Role' : '编辑 Role'"
    width="900px"
    top="5vh"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
    @closed="handleClosed"
  >
    <!-- 编辑模式切换 -->
    <div class="mode-switch">
      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <div class="radio-content">
            <FormInput :size="13" />
            <span>表单编辑</span>
          </div>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <div class="radio-content">
            <Code :size="13" />
            <span>YAML 编辑</span>
          </div>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态 -->
    <div v-if="dataLoading" style="text-align: center; padding: 30px">
      <ElSkeleton :rows="6" animated />
      <div style="margin-top: 12px; color: #909399; font-size: 13px">正在加载 Role 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="form-content">
      <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="140px">
        <!-- 基础信息 -->
        <div class="form-section">
          <div class="section-header">
            <div class="header-left">
              <ShieldCheck :size="16" />
              <span>基础信息</span>
            </div>
          </div>

          <ElFormItem label="名称" prop="name" required>
            <ElInput
              v-model="formData.name"
              placeholder="例如: pod-reader"
              :disabled="dialogType === 'edit'"
              maxlength="63"
              show-word-limit
            >
              <template #suffix>
                <ElTooltip content="小写字母、数字、连字符，以字母或数字开头结尾" placement="top">
                  <HelpCircle :size="13" class="help-icon" />
                </ElTooltip>
              </template>
            </ElInput>
          </ElFormItem>
        </div>

        <!-- 策略规则 -->
        <div class="form-section">
          <div class="section-header">
            <div class="header-left">
              <Lock :size="16" />
              <span>策略规则 (Policy Rules)</span>
            </div>
            <ElTooltip placement="top">
              <template #content>
                <div>定义 Role 可以访问的资源和操作</div>
                <div>至少需要添加一条规则</div>
              </template>
              <HelpCircle :size="13" class="help-icon" />
            </ElTooltip>
          </div>

          <div v-if="formData.rules.length > 0" class="rules-list">
            <div v-for="(rule, index) in formData.rules" :key="index" class="rule-card">
              <div class="rule-card-header">
                <span>规则 {{ index + 1 }}</span>
                <ElButton
                  type="danger"
                  :icon="Trash2"
                  circle
                  size="small"
                  @click="removeRule(index)"
                />
              </div>

              <div class="rule-card-body">
                <!-- API Groups -->
                <div class="form-item">
                  <label class="field-label">
                    <span style="color: #f56c6c">* </span>
                    API Groups
                    <ElTooltip placement="top">
                      <template #content>
                        <div>API 组，如: "", "apps", "batch"</div>
                        <div>"" 表示核心 API 组</div>
                        <div>"*" 表示所有 API 组</div>
                      </template>
                      <HelpCircle :size="11" class="help-icon" />
                    </ElTooltip>
                  </label>
                  <ElSelect
                    v-model="rule.apiGroups"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    placeholder="输入 API 组或选择"
                    style="width: 100%"
                  >
                    <ElOption label="核心组 (空字符串)" value="" />
                    <ElOption label="apps" value="apps" />
                    <ElOption label="batch" value="batch" />
                    <ElOption label="extensions" value="extensions" />
                    <ElOption label="networking.k8s.io" value="networking.k8s.io" />
                    <ElOption label="rbac.authorization.k8s.io" value="rbac.authorization.k8s.io" />
                    <ElOption label="storage.k8s.io" value="storage.k8s.io" />
                    <ElOption label="* (所有组)" value="*" />
                  </ElSelect>
                </div>

                <!-- Resources -->
                <div class="form-item">
                  <label class="field-label">
                    <span style="color: #f56c6c">* </span>
                    Resources
                    <ElTooltip placement="top">
                      <template #content>
                        <div>资源类型，如: pods, deployments, services</div>
                        <div>"*" 表示所有资源</div>
                      </template>
                      <HelpCircle :size="11" class="help-icon" />
                    </ElTooltip>
                  </label>
                  <ElSelect
                    v-model="rule.resources"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    placeholder="输入资源类型或选择"
                    style="width: 100%"
                  >
                    <ElOption label="pods" value="pods" />
                    <ElOption label="deployments" value="deployments" />
                    <ElOption label="services" value="services" />
                    <ElOption label="configmaps" value="configmaps" />
                    <ElOption label="secrets" value="secrets" />
                    <ElOption label="persistentvolumeclaims" value="persistentvolumeclaims" />
                    <ElOption label="ingresses" value="ingresses" />
                    <ElOption label="jobs" value="jobs" />
                    <ElOption label="cronjobs" value="cronjobs" />
                    <ElOption label="* (所有资源)" value="*" />
                  </ElSelect>
                </div>

                <!-- Verbs -->
                <div class="form-item">
                  <label class="field-label">
                    <span style="color: #f56c6c">* </span>
                    Verbs (操作)
                    <ElTooltip placement="top">
                      <template #content>
                        <div>允许的操作，如: get, list, watch, create</div>
                        <div>"*" 表示所有操作</div>
                      </template>
                      <HelpCircle :size="11" class="help-icon" />
                    </ElTooltip>
                  </label>
                  <ElSelect
                    v-model="rule.verbs"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    placeholder="选择操作"
                    style="width: 100%"
                  >
                    <ElOption label="get" value="get" />
                    <ElOption label="list" value="list" />
                    <ElOption label="watch" value="watch" />
                    <ElOption label="create" value="create" />
                    <ElOption label="update" value="update" />
                    <ElOption label="patch" value="patch" />
                    <ElOption label="delete" value="delete" />
                    <ElOption label="deletecollection" value="deletecollection" />
                    <ElOption label="* (所有操作)" value="*" />
                  </ElSelect>
                </div>

                <!-- Resource Names (可选) -->
                <div class="form-item">
                  <label class="field-label">
                    Resource Names (可选)
                    <ElTooltip placement="top">
                      <template #content>
                        <div>限制只能访问指定名称的资源</div>
                        <div>留空表示可以访问所有该类型的资源</div>
                      </template>
                      <HelpCircle :size="11" class="help-icon" />
                    </ElTooltip>
                  </label>
                  <ElSelect
                    v-model="rule.resourceNames"
                    multiple
                    filterable
                    allow-create
                    default-first-option
                    placeholder="输入资源名称（可选）"
                    style="width: 100%"
                  />
                </div>
              </div>
            </div>
          </div>

          <ElButton type="primary" :icon="Plus" plain @click="addRule"> 添加规则 </ElButton>
        </div>

        <!-- 标签和注解 -->
        <ElCollapse v-model="activeCollapse" class="metadata-collapse">
          <ElCollapseItem name="metadata">
            <template #title>
              <div class="collapse-title">
                <Tag :size="16" />
                <span>标签和注解</span>
                <span class="optional-mark">可选</span>
              </div>
            </template>

            <!-- 标签 -->
            <div class="metadata-section">
              <div class="metadata-header">
                <span class="metadata-label">标签 (Labels)</span>
                <ElButton type="primary" :icon="Plus" size="small" text @click="addLabel">
                  添加
                </ElButton>
              </div>
              <div v-if="formData.labels.length > 0" class="metadata-list">
                <div v-for="(item, index) in formData.labels" :key="index" class="metadata-row">
                  <ElInput
                    v-model="item.key"
                    placeholder="键"
                    style="width: 200px"
                    maxlength="63"
                  />
                  <span class="separator">=</span>
                  <ElInput v-model="item.value" placeholder="值" style="flex: 1" maxlength="63" />
                  <ElButton
                    type="danger"
                    :icon="Trash2"
                    circle
                    size="small"
                    @click="removeLabel(index)"
                  />
                </div>
              </div>
              <ElEmpty v-else description="暂无标签" :image-size="40" />
            </div>

            <!-- 注解 -->
            <div class="metadata-section">
              <div class="metadata-header">
                <span class="metadata-label">注解 (Annotations)</span>
                <ElButton type="primary" :icon="Plus" size="small" text @click="addAnnotation">
                  添加
                </ElButton>
              </div>
              <div v-if="formData.annotations.length > 0" class="metadata-list">
                <div
                  v-for="(item, index) in formData.annotations"
                  :key="index"
                  class="metadata-row"
                >
                  <ElInput
                    v-model="item.key"
                    placeholder="键"
                    style="width: 200px"
                    maxlength="253"
                  />
                  <span class="separator">=</span>
                  <ElInput v-model="item.value" placeholder="值" style="flex: 1" />
                  <ElButton
                    type="danger"
                    :icon="Trash2"
                    circle
                    size="small"
                    @click="removeAnnotation(index)"
                  />
                </div>
              </div>
              <ElEmpty v-else description="暂无注解" :image-size="40" />
            </div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
    </div>

    <!-- YAML 模式 -->
    <div v-else-if="editMode === 'yaml'" class="yaml-content">
      <ArtYamlEditor v-model="yamlContent" height="600px" @change="handleYamlChange" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <ElButton @click="handleCancel">取消</ElButton>
      <ElButton type="primary" :loading="submitting" @click="handleSubmit">
        {{ dialogType === 'add' ? '创建' : '更新' }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, watch, nextTick } from 'vue'
  import { ElMessage } from 'element-plus'
  import yaml from 'js-yaml'
  import {
    ShieldCheck,
    Lock,
    Tag,
    FileText,
    FormInput,
    Code,
    Plus,
    Trash2,
    HelpCircle
  } from 'lucide-vue-next'
  import {
    createRoleApi,
    updateRoleApi,
    getRoleYamlApi,
    type RoleListItem,
    type ProjectWorkspace,
    type PolicyRuleInfo
  } from '@/api/workload/core'
  import ArtYamlEditor from '@/components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../utils/k8s-validator'

  interface Props {
    visible: boolean
    dialogType: 'add' | 'edit'
    roleData?: RoleListItem
    workspace: ProjectWorkspace | null
  }

  interface KeyValue {
    key: string
    value: string
  }

  const props = withDefaults(defineProps<Props>(), {
    roleData: undefined
  })

  const emit = defineEmits(['close', 'success'])

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const activeCollapse = ref<string[]>(['metadata'])

  const formData = ref({
    name: '',
    rules: [] as PolicyRuleInfo[],
    labels: [] as KeyValue[],
    annotations: [] as KeyValue[]
  })

  const formRules = {
    name: createDns1123LabelRules(true, 'Role 名称')
  }

  const handleClose = (val: boolean) => {
    if (!val) {
      emit('close')
    }
  }

  // Rules 管理
  const addRule = () => {
    formData.value.rules.push({
      apiGroups: [''],
      resources: ['pods'],
      verbs: ['get', 'list']
    })
  }

  const removeRule = (index: number) => {
    formData.value.rules.splice(index, 1)
  }

  // Labels 管理
  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  // Annotations 管理
  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  // 工具函数
  const objectToArray = (obj?: Record<string, string>): KeyValue[] => {
    if (!obj) return []
    return Object.entries(obj)
      .filter(([key]) => key)
      .map(([key, value]) => ({ key, value }))
  }

  const arrayToObject = (arr: KeyValue[]): Record<string, string> => {
    const obj: Record<string, string> = {}
    arr.forEach((item) => {
      if (item.key && item.key.trim()) {
        obj[item.key.trim()] = item.value
      }
    })
    return obj
  }

  // 表单转 YAML
  const formToYaml = () => {
    const roleObj: any = {
      apiVersion: 'rbac.authorization.k8s.io/v1',
      kind: 'Role',
      metadata: {
        name: formData.value.name || 'role-name',
        namespace: props.workspace?.namespace || 'default'
      },
      rules: formData.value.rules
        .filter((r) => r.apiGroups.length > 0 && r.resources.length > 0 && r.verbs.length > 0)
        .map((r) => {
          const rule: any = {
            apiGroups: r.apiGroups,
            resources: r.resources,
            verbs: r.verbs
          }
          if (r.resourceNames && r.resourceNames.length > 0) {
            rule.resourceNames = r.resourceNames
          }
          return rule
        })
    }

    // Labels
    const labelsObj = arrayToObject(formData.value.labels)
    if (Object.keys(labelsObj).length > 0) {
      roleObj.metadata.labels = labelsObj
    }

    // Annotations
    const annotationsObj = arrayToObject(formData.value.annotations)
    if (Object.keys(annotationsObj).length > 0) {
      roleObj.metadata.annotations = annotationsObj
    }

    yamlContent.value = yaml.dump(roleObj, { indent: 2 })
  }

  // YAML 转表单
  const yamlToForm = (yamlStr: string) => {
    try {
      const obj = yaml.load(yamlStr) as any

      formData.value.name = obj.metadata?.name || ''

      // Rules
      formData.value.rules = (obj.rules || []).map((r: any) => ({
        apiGroups: r.apiGroups || [],
        resources: r.resources || [],
        verbs: r.verbs || [],
        resourceNames: r.resourceNames || []
      }))

      // Labels
      formData.value.labels = objectToArray(obj.metadata?.labels)

      // Annotations
      formData.value.annotations = objectToArray(obj.metadata?.annotations)
    } catch (error) {
      console.error('YAML 解析失败:', error)
      throw error
    }
  }

  // 加载 YAML
  const loadRoleYAML = async () => {
    if (!props.workspace || !props.roleData) {
      return
    }

    dataLoading.value = true
    try {
      const yamlStr = await getRoleYamlApi({
        clusterUuid: props.workspace.clusterUuid!,
        namespace: props.workspace.namespace!,
        name: props.roleData.name
      })

      yamlContent.value = yamlStr
      yamlToForm(yamlStr)
    } catch (error: any) {
      console.error('加载失败:', error)
      emit('close')
    } finally {
      dataLoading.value = false
    }
  }

  // 监听对话框打开
  watch(
    () => props.visible,
    async (val) => {
      if (val) {
        if (props.dialogType === 'edit' && props.roleData && props.workspace) {
          await loadRoleYAML()
        } else if (props.dialogType === 'add') {
          // 初始化一个空的 Rule
          formData.value.rules = [
            {
              apiGroups: [''],
              resources: ['pods'],
              verbs: ['get', 'list']
            }
          ]
          await nextTick()
          formToYaml()
        }
      }
    },
    { immediate: true }
  )

  // 监听编辑模式切换
  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return

    if (newMode === 'yaml' && oldMode === 'form') {
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      try {
        yamlToForm(yamlContent.value)
        ElMessage.success('已切换到表单模式')
      } catch {
        ElMessage.error('YAML 格式错误，无法切换到表单模式')
        nextTick(() => {
          editMode.value = 'yaml'
        })
      }
    }
  })

  // 监听表单数据变化
  watch(
    () => formData.value,
    () => {
      if (editMode.value === 'form' && !dataLoading.value) {
        formToYaml()
      }
    },
    { deep: true }
  )

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  // 提交
  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    if (editMode.value === 'form') {
      try {
        await formRef.value?.validate()
      } catch {
        return
      }

      // 验证至少有一条规则
      if (formData.value.rules.length === 0) {
        ElMessage.error('至少需要添加一条策略规则')
        return
      }

      // 验证所有规则都填写了必填字段
      const hasInvalidRule = formData.value.rules.some((r) => {
        return r.apiGroups.length === 0 || r.resources.length === 0 || r.verbs.length === 0
      })

      if (hasInvalidRule) {
        ElMessage.error('请完整填写所有规则的必填字段（API Groups、Resources、Verbs）')
        return
      }

      formToYaml()
    }

    submitting.value = true
    try {
      if (props.dialogType === 'add') {
        await createRoleApi({
          clusterUuid: props.workspace.clusterUuid!,
          namespace: props.workspace.namespace!,
          yamlStr: yamlContent.value
        })
        ElMessage.success('创建成功')
      } else {
        await updateRoleApi({
          clusterUuid: props.workspace.clusterUuid!,
          namespace: props.workspace.namespace!,
          name: formData.value.name,
          yamlStr: yamlContent.value
        })
        ElMessage.success('更新成功')
      }

      emit('success')
      emit('close')
    } finally {
      submitting.value = false
    }
  }

  const handleCancel = () => {
    emit('close')
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    formData.value = {
      name: '',
      rules: [],
      labels: [],
      annotations: []
    }
    yamlContent.value = ''
    editMode.value = 'form'
  }
</script>

<style lang="scss" scoped>
  .mode-switch {
    margin-bottom: 16px;
    display: flex;
    justify-content: flex-end;

    .radio-content {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 12px;
    }
  }

  .form-content {
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;
    }
  }

  .form-section {
    margin-bottom: 16px;
    padding: 12px 16px;
    background: #fafafa;
    border-radius: 4px;
    border: 1px solid #e4e7ed;

    .section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;
      padding-bottom: 8px;
      border-bottom: 1px solid #e4e7ed;

      .header-left {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 14px;
        font-weight: 500;
        color: #303133;

        svg {
          color: #409eff;
        }
      }

      .optional-mark {
        margin-left: 8px;
        font-size: 11px;
        font-weight: 400;
        color: #909399;
        background: #f4f4f5;
        padding: 2px 6px;
        border-radius: 2px;
      }
    }
  }

  .help-icon {
    cursor: help;
    color: #909399;
    transition: color 0.2s;

    &:hover {
      color: #606266;
    }
  }

  .rules-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
  }

  // 标签和注解折叠面板样式
  .metadata-collapse {
    border: none;
    background: transparent;

    :deep(.el-collapse-item) {
      background: #fff;
      border: 1px solid #dcdfe6;
      border-radius: 4px;
      overflow: hidden;

      &.is-active {
        border-color: #409eff;
      }
    }

    :deep(.el-collapse-item__header) {
      height: 48px;
      line-height: 48px;
      background: #f5f7fa;
      border-bottom: 1px solid #e4e7ed;
      padding: 0 16px;
      font-size: 14px;
      font-weight: 500;
      color: #303133;
      transition: all 0.3s;

      &:hover {
        background: #ecf5ff;
      }

      .el-collapse-item__arrow {
        margin-right: 8px;
        color: #909399;
        transition: transform 0.3s;
      }

      &.is-active {
        .el-collapse-item__arrow {
          transform: rotate(90deg);
          color: #409eff;
        }
      }
    }

    :deep(.el-collapse-item__wrap) {
      background: #fff;
      border-bottom: none;
    }

    :deep(.el-collapse-item__content) {
      padding: 16px 20px;
    }

    .collapse-title {
      display: flex;
      align-items: center;
      gap: 8px;

      svg {
        color: #409eff;
      }

      .optional-mark {
        margin-left: 4px;
        font-size: 11px;
        font-weight: 400;
        color: #909399;
        background: #f4f4f5;
        padding: 2px 6px;
        border-radius: 2px;
      }
    }
  }

  .metadata-section {
    margin-bottom: 16px;

    &:last-child {
      margin-bottom: 0;
    }

    .metadata-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 8px;

      .metadata-label {
        font-size: 13px;
        font-weight: 500;
        color: #606266;
      }
    }

    .metadata-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .metadata-row {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 8px 12px;
      background: #fafafa;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      transition: border-color 0.2s;

      &:hover {
        border-color: #409eff;
      }

      .separator {
        color: #909399;
        font-weight: 500;
        flex-shrink: 0;
      }
    }
  }

  .rule-card {
    background: white;
    border-radius: 4px;
    border: 1px solid #dcdfe6;
    overflow: hidden;
  }

  .rule-card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 8px 12px;
    background: #67c23a;
    color: white;
    font-weight: 500;
    font-size: 13px;
  }

  .rule-card-body {
    padding: 12px;
    display: flex;
    flex-direction: column;
    gap: 10px;
  }

  .form-item {
    display: flex;
    flex-direction: column;
    gap: 4px;
  }

  .field-label {
    display: flex;
    align-items: center;
    gap: 4px;
    font-size: 12px;
    font-weight: 500;
    color: #606266;
  }

  .label-row,
  .annotation-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: white;
    border-radius: 4px;
    border: 1px solid #dcdfe6;
    transition: border-color 0.2s;

    &:hover {
      border-color: #409eff;
    }

    .separator {
      color: #909399;
      font-weight: 500;
      flex-shrink: 0;
    }
  }

  .yaml-content {
    min-height: 500px;
  }

  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    font-weight: 500;
    color: #303133;
  }
</style>
