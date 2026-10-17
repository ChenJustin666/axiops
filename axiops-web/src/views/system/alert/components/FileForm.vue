<template>
  <ElDialog
    v-model="visible"
    :title="isEdit ? '编辑告警规则文件' : '添加告警规则文件'"
    width="600px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <ElForm
      ref="formRef"
      :model="form"
      :rules="rules"
      label-width="100px"
      @submit.prevent="handleSubmit"
    >
      <ElFormItem label="文件代码" prop="fileCode">
        <ElInput
          v-model="form.fileCode"
          placeholder="请输入文件代码，如：node-rules、pod-rules"
          :disabled="isEdit"
          clearable
        />
      </ElFormItem>

      <ElFormItem label="文件名称" prop="fileName">
        <ElInput v-model="form.fileName" placeholder="请输入文件名称" clearable />
      </ElFormItem>

      <ElFormItem label="命名空间" prop="namespace">
        <ElInput v-model="form.namespace" placeholder="请输入命名空间，如：monitoring" clearable />
      </ElFormItem>

      <ElFormItem label="标签" prop="labels">
        <ElInput
          v-model="form.labels"
          type="textarea"
          :rows="2"
          placeholder='JSON 格式，例如：{"app": "prometheus", "release": "kube-prometheus"}'
        />
        <div class="form-tip">💡 JSON 格式的键值对，用于 PrometheusRule 资源的 labels</div>
      </ElFormItem>

      <ElFormItem label="是否启用" prop="isEnabled">
        <ElSwitch v-model="form.isEnabled" />
      </ElFormItem>

      <ElFormItem label="文件描述" prop="description">
        <ElInput
          v-model="form.description"
          type="textarea"
          :rows="3"
          placeholder="请输入文件描述"
          maxlength="500"
          show-word-limit
        />
      </ElFormItem>

      <ElFormItem label="排序序号" prop="sortOrder">
        <ElInputNumber
          v-model="form.sortOrder"
          :min="0"
          :max="9999"
          placeholder="请输入排序序号"
          style="width: 100%"
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <ElButton @click="visible = false">取消</ElButton>
      <ElButton type="primary" :loading="loading" @click="handleSubmit">
        {{ isEdit ? '更新' : '添加' }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, watch, computed } from 'vue'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import {
    addAlertRuleFileApi,
    updateAlertRuleFileApi,
    type AddAlertRuleFileRequest,
    type UpdateAlertRuleFileRequest,
    type AlertRuleFile
  } from '@/api/manager/alert'

  interface Props {
    modelValue: boolean
    formData?: Partial<AlertRuleFile>
    isEdit?: boolean
  }

  const props = withDefaults(defineProps<Props>(), {
    formData: () => ({}),
    isEdit: false
  })

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    success: []
  }>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const formRef = ref<FormInstance>()
  const loading = ref(false)

  const form = reactive<Partial<AlertRuleFile>>({
    fileCode: '',
    fileName: '',
    namespace: 'monitoring',
    labels: '',
    isEnabled: true,
    description: '',
    sortOrder: 0
  })

  const rules: FormRules = {
    fileCode: [
      { required: true, message: '请输入文件代码', trigger: 'blur' },
      { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' },
      {
        pattern: /^[a-z][a-z0-9_-]*$/,
        message: '只能包含小写字母、数字、下划线和短横线，且必须以字母开头',
        trigger: 'blur'
      }
    ],
    fileName: [
      { required: true, message: '请输入文件名称', trigger: 'blur' },
      { min: 2, max: 100, message: '长度在 2 到 100 个字符', trigger: 'blur' }
    ],
    namespace: [
      { required: true, message: '请输入命名空间', trigger: 'blur' },
      { max: 63, message: '长度不能超过 63 个字符', trigger: 'blur' }
    ]
  }

  const resetForm = () => {
    Object.assign(form, {
      fileCode: '',
      fileName: '',
      namespace: 'monitoring',
      labels: '',
      isEnabled: true,
      description: '',
      sortOrder: 0
    })
    formRef.value?.clearValidate()
  }

  watch(
    () => props.formData,
    (val) => {
      if (val && Object.keys(val).length > 0) {
        Object.assign(form, val)
      } else {
        resetForm()
      }
    },
    { immediate: true, deep: true }
  )

  const validateJSON = (value: string): boolean => {
    if (!value) return true
    try {
      JSON.parse(value)
      return true
    } catch {
      return false
    }
  }

  const handleSubmit = async () => {
    if (!formRef.value) return

    try {
      await formRef.value.validate()

      if (form.labels && !validateJSON(form.labels)) {
        return
      }

      loading.value = true

      if (props.isEdit && form.id) {
        await updateAlertRuleFileApi(form as UpdateAlertRuleFileRequest)
        ElMessage.success('更新成功')
      } else {
        await addAlertRuleFileApi(form as AddAlertRuleFileRequest)
        ElMessage.success('添加成功')
      }

      visible.value = false
      emit('success')
    } finally {
      loading.value = false
    }
  }

  const handleClosed = () => {
    resetForm()
  }
</script>

<style scoped lang="scss">
  .form-tip {
    margin-top: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1.5;
  }
</style>
