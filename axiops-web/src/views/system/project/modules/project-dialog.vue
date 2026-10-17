<template>
  <ElDialog
    :title="isEdit ? '编辑项目' : '新建项目'"
    :model-value="visible"
    @update:model-value="handleCancel"
    width="600px"
    align-center
    class="project-dialog"
    @closed="handleClosed"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <ElForm ref="formRef" :model="form" :rules="rules" label-width="100px">
      <ElFormItem label="项目名称" prop="name">
        <ElInput
          v-model="form.name"
          placeholder="请输入项目名称"
          :disabled="submitLoading"
          maxlength="100"
          show-word-limit
        />
      </ElFormItem>

      <ElFormItem label="项目描述" prop="description">
        <ElInput
          v-model="form.description"
          type="textarea"
          placeholder="请输入项目描述"
          :disabled="submitLoading"
          :rows="4"
          maxlength="500"
          show-word-limit
        />
      </ElFormItem>
    </ElForm>

    <template #footer>
      <span class="dialog-footer">
        <ElButton @click="handleCancel" :disabled="submitLoading">取 消</ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="submitLoading">
          {{ submitLoading ? (isEdit ? '更新中...' : '创建中...') : '确 定' }}
        </ElButton>
      </span>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch } from 'vue'
  import type { FormInstance, FormRules } from 'element-plus'

  interface ProjectFormData {
    id?: number
    name: string
    description: string
  }

  interface Props {
    visible: boolean
    editData?: ProjectFormData | null
    submitLoading: boolean
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', data: ProjectFormData): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const formRef = ref<FormInstance>()
  const isEdit = computed(() => !!props.editData?.id)

  const form = reactive<ProjectFormData>({
    id: undefined,
    name: '',
    description: ''
  })

  const rules: FormRules = {
    name: [{ required: true, message: '请输入项目名称', trigger: 'blur' }]
  }

  watch(
    () => props.visible,
    (val) => {
      if (val) {
        if (props.editData) {
          form.id = props.editData.id
          form.name = props.editData.name || ''
          form.description = props.editData.description || ''
        } else {
          form.id = undefined
          form.name = ''
          form.description = ''
        }
      }
    }
  )

  const handleCancel = () => {
    emit('update:visible', false)
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
  }

  const handleSubmit = async () => {
    if (!formRef.value) return
    await formRef.value.validate((valid) => {
      if (valid) {
        emit('submit', { ...form })
      }
    })
  }
</script>
