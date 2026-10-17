<template>
  <ElDialog
    :model-value="visible"
    :title="dialogType === 'add' ? '创建 ConfigMap' : '编辑 ConfigMap'"
    width="900px"
    top="5vh"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
    @closed="handleClosed"
  >
    <!-- 编辑模式切换 -->
    <div class="unified-mode-switch">
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
    <div v-if="dataLoading" class="unified-loading">
      <ElIcon class="is-loading" :size="32">
        <Loader2 />
      </ElIcon>
      <div class="loading-text">正在加载 ConfigMap 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="configmap-form-container">
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
        class="configmap-form"
      >
        <ElCollapse v-model="activeNames" class="form-collapse">
          <!-- 基础信息 -->
          <ElCollapseItem name="basic" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><InfoFilled /></ElIcon>
                <span>基础信息</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="名称" prop="name">
                <ElInput
                  v-model="formData.name"
                  placeholder="请输入 ConfigMap 名称"
                  :disabled="dialogType === 'edit'"
                  maxlength="63"
                  show-word-limit
                />
              </ElFormItem>

              <ElFormItem label="命名空间">
                <ElInput :value="workspace?.namespace" disabled />
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 数据配置 -->
          <ElCollapseItem name="data" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Document /></ElIcon>
                <span>数据配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 16px">
                <template #title>支持多行文本和大文本内容</template>
              </ElAlert>

              <div class="data-list">
                <ElCollapse v-model="activeDataItems" accordion>
                  <ElCollapseItem
                    v-for="(item, index) in formData.data"
                    :key="index"
                    :name="index"
                    class="data-collapse-item"
                  >
                    <template #title>
                      <div class="data-collapse-title">
                        <div class="title-left">
                          <Key :size="14" />
                          <span class="data-key">{{ item.key || `数据项 ${index + 1}` }}</span>
                          <ElTag v-if="item.value" size="small" type="info">
                            {{ item.value.length }} 字符
                          </ElTag>
                        </div>
                        <ElButton
                          type="danger"
                          :icon="Trash2"
                          circle
                          size="small"
                          @click.stop="removeData(index)"
                        />
                      </div>
                    </template>

                    <div class="data-item-content">
                      <ElFormItem label="键（Key）" :prop="`data.${index}.key`">
                        <ElInput
                          v-model="item.key"
                          placeholder="例如: config.yaml"
                          maxlength="253"
                          show-word-limit
                        />
                      </ElFormItem>

                      <ElFormItem label="值（Value）">
                        <ElInput
                          v-model="item.value"
                          type="textarea"
                          placeholder="支持多行文本，例如配置文件内容"
                          :rows="8"
                          :autosize="{ minRows: 8, maxRows: 20 }"
                          class="data-value-textarea"
                        />
                        <div class="value-info">
                          <span>当前长度: {{ item.value?.length || 0 }} 字符</span>
                        </div>
                      </ElFormItem>
                    </div>
                  </ElCollapseItem>
                </ElCollapse>
              </div>

              <ElButton type="primary" :icon="Plus" plain @click="addData" style="width: 100%">
                添加数据项
              </ElButton>
            </div>
          </ElCollapseItem>

          <!-- 标签和注解（可选） -->
          <ElCollapseItem name="metadata" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><PriceTag /></ElIcon>
                <span>标签和注解（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="标签 (Labels)">
                <ElTable :data="formData.labels" border style="width: 100%">
                  <ElTableColumn label="标签名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="标签名" maxlength="63" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="标签值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="标签值" maxlength="63" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeLabel($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addLabel"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加标签
                </ElButton>
              </ElFormItem>

              <ElFormItem label="注解 (Annotations)">
                <ElTable :data="formData.annotations" border style="width: 100%">
                  <ElTableColumn label="注解名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="注解名" maxlength="253" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="注解值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="注解值" maxlength="253" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeAnnotation($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addAnnotation"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加注解
                </ElButton>
              </ElFormItem>
            </div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
    </div>

    <!-- YAML 模式 -->
    <div v-else class="unified-yaml-container">
      <ArtYamlEditor v-model="yamlContent" height="600px" @change="handleYamlChange" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="unified-dialog-footer">
        <ElButton @click="handleCancel">取 消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ dialogType === 'add' ? '创 建' : '更 新' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage, ElIcon } from 'element-plus'
  import { InfoFilled, Document, PriceTag } from '@element-plus/icons-vue'
  import { Loader2, FormInput, Code, Plus, Trash2, Key } from 'lucide-vue-next'
  import yaml from 'js-yaml'
  import {
    createConfigMapApi,
    updateConfigMapApi,
    getConfigMapYamlApi,
    type ConfigMapListItem,
    type ProjectWorkspace
  } from '@/api'
  import ArtYamlEditor from '@/components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../utils/k8s-validator'
  import {
    parseYaml,
    stringifyYaml,
    mergeFormToYaml,
    objectToKeyValueArray,
    keyValueArrayToObject
  } from '../../utils/yaml-merger'

  interface Props {
    modelValue: boolean
    dialogType: 'add' | 'edit'
    configmapData?: ConfigMapListItem
    workspace: ProjectWorkspace | null
  }

  interface KeyValue {
    key: string
    value: string
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['update:modelValue', 'success'])

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const activeDataItems = ref<number[]>([0])
  const activeNames = ref<string[]>(['basic', 'data', 'metadata'])

  /** 原始 YAML 字符串（编辑模式下用于保留未显示字段） */
  const rawSourceYaml = ref('')

  const formData = ref({
    name: '',
    labels: [] as KeyValue[],
    annotations: [] as KeyValue[],
    data: [] as KeyValue[]
  })

  const formRules = {
    name: createDns1123LabelRules(true, 'ConfigMap 名称')
  }

  // 监听弹窗打开，加载数据
  watch(visible, async (val) => {
    if (val && props.dialogType === 'edit' && props.configmapData && props.workspace) {
      dataLoading.value = true
      try {
        // 获取完整 YAML 以保留所有字段
        const yamlStr = await getConfigMapYamlApi({
          workloadId: props.workspace.id,
          name: props.configmapData.name
        })
        rawSourceYaml.value = yamlStr

        const obj = parseYaml<any>(yamlStr)
        if (obj) {
          formData.value = {
            name: obj.metadata?.name || '',
            labels: objectToKeyValueArray(obj.metadata?.labels),
            annotations: objectToKeyValueArray(obj.metadata?.annotations),
            data: objectToKeyValueArray(obj.data)
          }
        }

        formToYaml()
      } catch (error) {
        console.error('加载 ConfigMap 详情失败:', error)
        visible.value = false
      } finally {
        dataLoading.value = false
      }
    } else if (val && props.dialogType === 'add') {
      rawSourceYaml.value = ''
      formData.value.data = [{ key: '', value: '' }]
      formToYaml()
    }
  })

  // 监听编辑模式切换
  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return

    if (newMode === 'yaml' && oldMode === 'form') {
      // 从表单切换到 YAML，同步表单数据到 YAML
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      // 从 YAML 切换到表单，同步 YAML 数据到表单
      yamlToForm()
    }
  })

  // 监听表单数据变化，同步到 YAML
  watch(
    () => [
      formData.value.name,
      formData.value.labels,
      formData.value.annotations,
      formData.value.data
    ],
    () => {
      if (editMode.value === 'form') {
        formToYaml()
      }
    },
    { deep: true }
  )

  const objectToArray = (obj?: Record<string, string>): KeyValue[] => {
    return objectToKeyValueArray(obj)
  }

  const arrayToObject = (arr: KeyValue[]): Record<string, string> => {
    return keyValueArrayToObject(arr)
  }

  const formToYaml = () => {
    const labelsObj = arrayToObject(formData.value.labels)
    const annotationsObj = arrayToObject(formData.value.annotations)
    const dataObj = arrayToObject(formData.value.data)

    const configMapObj: any = {
      apiVersion: 'v1',
      kind: 'ConfigMap',
      metadata: {
        name: formData.value.name || 'configmap-name',
        namespace: props.workspace?.namespace || 'default'
      },
      data: dataObj
    }

    if (Object.keys(labelsObj).length > 0) {
      configMapObj.metadata.labels = labelsObj
    }
    if (Object.keys(annotationsObj).length > 0) {
      configMapObj.metadata.annotations = annotationsObj
    }

    yamlContent.value = stringifyYaml(configMapObj)
  }

  const yamlToForm = () => {
    try {
      const obj = parseYaml<any>(yamlContent.value)

      if (!obj || typeof obj !== 'object') {
        ElMessage.error('YAML 格式不正确')
        return
      }

      if (obj.kind !== 'ConfigMap') {
        ElMessage.error('YAML 类型必须是 ConfigMap')
        return
      }

      if (props.workspace) {
        if (!obj.metadata) {
          obj.metadata = {}
        }
        obj.metadata.namespace = props.workspace.namespace
      }

      formData.value = {
        name: obj.metadata?.name || '',
        labels: objectToArray(obj.metadata?.labels),
        annotations: objectToArray(obj.metadata?.annotations),
        data: objectToArray(obj.data)
      }

      formToYaml()
    } catch (error) {
      console.error('YAML 解析失败:', error)
      ElMessage.error('YAML 解析失败，请检查格式')
    }
  }

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  const addData = () => {
    formData.value.data.push({ key: '', value: '' })
  }

  const removeData = (index: number) => {
    formData.value.data.splice(index, 1)
  }

  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    // YAML 模式下，先同步到表单
    if (editMode.value === 'yaml') {
      yamlToForm()
    }

    // 表单验证
    try {
      await formRef.value?.validate()
    } catch {
      return
    }

    // 验证数据项
    if (formData.value.data.length === 0) {
      ElMessage.warning('至少需要添加一个数据项')
      return
    }

    const hasEmptyDataKey = formData.value.data.some((item) => !item.key || !item.key.trim())
    if (hasEmptyDataKey) {
      ElMessage.warning('数据项的键不能为空')
      return
    }

    submitting.value = true
    try {
      const requestData = {
        workloadId: props.workspace.id,
        name: formData.value.name,
        data: arrayToObject(formData.value.data),
        labels: arrayToObject(formData.value.labels),
        annotations: arrayToObject(formData.value.annotations)
      }

      if (props.dialogType === 'add') {
        await createConfigMapApi(requestData)
        ElMessage.success('创建成功')
      } else {
        await updateConfigMapApi(requestData)
        ElMessage.success('更新成功')
      }

      emit('success')
      visible.value = false
    } finally {
      submitting.value = false
    }
  }

  const handleClose = () => {
    visible.value = false
  }

  const handleCancel = () => {
    visible.value = false
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    formData.value = {
      name: '',
      labels: [],
      annotations: [],
      data: []
    }
    yamlContent.value = ''
    rawSourceYaml.value = ''
    editMode.value = 'form'
    activeNames.value = ['basic', 'data', 'metadata']
  }
</script>

<style lang="scss" scoped>
  .configmap-form-container {
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;

      &:hover {
        background: #c0c4cc;
      }
    }
  }

  .configmap-form {
    :deep(.form-collapse) {
      border: none;
      background: transparent;

      .el-collapse-item {
        background: #fff;
        border: 1px solid #dcdfe6;
        border-radius: 4px;
        margin-bottom: 12px;
        overflow: hidden;

        &.is-active {
          border-color: #409eff;
        }

        &:last-child {
          margin-bottom: 0;
        }
      }

      .el-collapse-item__header {
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

      .el-collapse-item__wrap {
        background: #fff;
        border-bottom: none;
      }

      .el-collapse-item__content {
        padding: 0;
      }
    }

    .collapse-section-item {
      .collapse-title {
        display: flex;
        align-items: center;
        width: 100%;

        .title-icon {
          margin-right: 8px;
          font-size: 16px;
          color: #409eff;
        }

        span {
          flex: 1;
        }
      }
    }

    .collapse-content {
      padding: 20px 24px;
    }
  }

  // 数据列表样式
  .data-list {
    margin-bottom: 12px;

    .data-collapse-item {
      margin-bottom: 8px;
      background: white;
      border-radius: 4px;
      border: 1px solid #dcdfe6;

      :deep(.el-collapse-item__header) {
        padding: 8px 12px;
        height: auto;
        line-height: 1.5;
        background: white;
        border: none;
        border-radius: 4px;

        &:hover {
          background: #f5f7fa;
        }
      }

      :deep(.el-collapse-item__wrap) {
        border: none;
      }

      :deep(.el-collapse-item__content) {
        padding: 12px;
        padding-top: 0;
      }

      .data-collapse-title {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        padding-right: 8px;

        .title-left {
          display: flex;
          align-items: center;
          gap: 8px;
          flex: 1;
          min-width: 0;

          .data-key {
            font-weight: 500;
            color: #303133;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }

      .data-item-content {
        .data-value-textarea {
          :deep(.el-textarea__inner) {
            font-family: var(--ax-font-mono);
            font-size: 13px;
            line-height: 1.6;
          }
        }

        .value-info {
          margin-top: 4px;
          font-size: 12px;
          color: #909399;
          text-align: right;
        }
      }
    }
  }
</style>
