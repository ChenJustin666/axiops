<template>
  <ElDialog
    :title="dialogTitle"
    :model-value="visible"
    @update:model-value="handleCancel"
    width="600px"
    align-center
    class="platform-dialog"
    @closed="handleClosed"
    destroy-on-close
  >
    <ArtForm
      ref="formRef"
      v-model="form"
      :items="formItems"
      :rules="rules"
      :span="24"
      :gutter="20"
      label-width="100px"
      :show-reset="false"
      :show-submit="false"
    >
      <template #platformIcon>
        <ElInput v-model="form.platformIcon" placeholder="请输入图标名称，如：ri:apps-2-line">
          <template #prefix>
            <ArtSvgIcon v-if="form.platformIcon" :name="form.platformIcon" :size="18" />
            <ArtSvgIcon v-else name="ri:apps-2-line" :size="18" class="opacity-50" />
          </template>
        </ElInput>
      </template>
    </ArtForm>

    <template #footer>
      <span class="dialog-footer">
        <ElButton @click="handleCancel">取消</ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="loading">确定</ElButton>
      </span>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import type { FormRules } from 'element-plus'
  import type { PlatformSysPlatform } from '@/api/portal/platform'
  import type { FormItem } from '@/components/core/forms/art-form/index.vue'
  import ArtForm from '@/components/core/forms/art-form/index.vue'

  interface PlatformFormData {
    id: number
    platformCode: string
    platformName: string
    platformDesc: string
    platformIcon: string
    sort: number
    isEnable: boolean
    isDefault: boolean
  }

  interface Props {
    visible: boolean
    editData?: PlatformSysPlatform | null
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'submit', data: any): void
  }

  const props = withDefaults(defineProps<Props>(), {
    visible: false
  })

  const emit = defineEmits<Emits>()

  const formRef = ref()
  const isEdit = ref(false)
  const loading = ref(false)

  const form = reactive<PlatformFormData>({
    id: 0,
    platformCode: '',
    platformName: '',
    platformDesc: '',
    platformIcon: '',
    sort: 0,
    isEnable: true,
    isDefault: false
  })

  /**
   * 表单验证规则
   */
  const rules = reactive<FormRules>({
    platformCode: [
      { required: true, message: '请输入平台编码', trigger: 'blur' },
      { min: 2, max: 50, message: '长度在 2 到 50 个字符', trigger: 'blur' },
      {
        pattern: /^[a-zA-Z][a-zA-Z0-9_-]*$/,
        message: '平台编码必须以字母开头，只能包含字母、数字、下划线和连字符',
        trigger: 'blur'
      }
    ],
    platformName: [
      { required: true, message: '请输入平台名称', trigger: 'blur' },
      { min: 2, max: 100, message: '长度在 2 到 100 个字符', trigger: 'blur' }
    ]
  })

  /**
   * 表单项配置
   */
  const formItems = computed<FormItem[]>(() => [
    {
      label: '平台编码',
      key: 'platformCode',
      type: 'input',
      props: {
        placeholder: '请输入平台编码，如：kube-nova',
        clearable: true,
        disabled: isEdit.value
      }
    },
    {
      label: '平台名称',
      key: 'platformName',
      type: 'input',
      props: {
        placeholder: '请输入平台名称',
        clearable: true
      }
    },
    {
      label: '平台图标',
      key: 'platformIcon'
    },
    {
      label: '平台描述',
      key: 'platformDesc',
      type: 'textarea',
      props: {
        placeholder: '请输入平台描述',
        rows: 3,
        maxlength: 500,
        showWordLimit: true
      }
    },
    {
      label: '排序',
      key: 'sort',
      type: 'number',
      props: {
        min: 0,
        controlsPosition: 'right',
        style: { width: '100%' }
      }
    },
    {
      label: '是否启用',
      key: 'isEnable',
      type: 'switch',
      span: 12
    },
    {
      label: '设为默认',
      key: 'isDefault',
      type: 'switch',
      span: 12
    }
  ])

  /**
   * 对话框标题
   */
  const dialogTitle = computed(() => {
    return isEdit.value ? '编辑平台' : '新增平台'
  })

  /**
   * 重置表单数据
   */
  const resetForm = (): void => {
    formRef.value?.reset()
    form.id = 0
    form.platformCode = ''
    form.platformName = ''
    form.platformDesc = ''
    form.platformIcon = ''
    form.sort = 0
    form.isEnable = true
    form.isDefault = false
  }

  /**
   * 加载表单数据（编辑模式）
   */
  const loadFormData = (): void => {
    if (!props.editData) return

    isEdit.value = true

    const row = props.editData

    form.id = row.id || 0
    form.platformCode = row.platformCode || ''
    form.platformName = row.platformName || ''
    form.platformDesc = row.platformDesc || ''
    form.platformIcon = row.platformIcon || ''
    form.sort = row.sort || 0
    form.isEnable = row.isEnable === 1
    form.isDefault = row.isDefault === 1
  }

  /**
   * 提交表单
   */
  const handleSubmit = async (): Promise<void> => {
    if (!formRef.value) return

    try {
      await formRef.value.validate()

      const submitData = {
        id: form.id,
        platformCode: form.platformCode,
        platformName: form.platformName,
        platformDesc: form.platformDesc,
        platformIcon: form.platformIcon,
        sort: form.sort,
        isEnable: form.isEnable ? 1 : 0,
        isDefault: form.isDefault ? 1 : 0
      }

      emit('submit', submitData)
    } catch (error) {
      console.error('表单验证失败:', error)
    }
  }

  /**
   * 取消操作
   */
  const handleCancel = (): void => {
    emit('update:visible', false)
  }

  /**
   * 对话框关闭后的回调
   */
  const handleClosed = (): void => {
    resetForm()
    isEdit.value = false
  }

  /**
   * 监听对话框显示状态
   */
  watch(
    () => props.visible,
    (newVal) => {
      if (newVal) {
        nextTick(() => {
          if (props.editData) {
            loadFormData()
          }
        })
      }
    }
  )
</script>

<style scoped lang="scss">
  .platform-dialog {
    :deep(.el-dialog__body) {
      max-height: 70vh;
      overflow-y: auto;
    }
  }
</style>
