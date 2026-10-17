<template>
  <ElDialog
    v-model="visible"
    :title="dialogType === 'add' ? '新增收费配置' : '编辑收费配置'"
    width="650px"
    align-center
    @close="handleClose"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
  >
    <ElForm ref="formRef" :model="formData" :rules="rules" label-width="120px">
      <ElRow :gutter="20">
        <ElCol :span="24">
          <ElFormItem label="配置名称" prop="configName">
            <ElInput
              v-model="formData.configName"
              placeholder="请输入配置名称"
              maxlength="100"
              :disabled="loading"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="20">
        <ElCol :span="24">
          <ElFormItem label="配置描述" prop="description">
            <ElInput
              v-model="formData.description"
              type="textarea"
              placeholder="请输入配置描述"
              maxlength="500"
              :rows="3"
              :disabled="loading"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElDivider content-position="left">资源单价配置</ElDivider>

      <ElRow :gutter="20">
        <ElCol :span="12">
          <ElFormItem prop="cpuPrice">
            <template #label>
              <span>CPU 单价</span>
              <ElTooltip content="元/核/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.cpuPrice"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem prop="memoryPrice">
            <template #label>
              <span>内存单价</span>
              <ElTooltip content="元/GiB/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.memoryPrice"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="20">
        <ElCol :span="12">
          <ElFormItem prop="storagePrice">
            <template #label>
              <span>存储单价</span>
              <ElTooltip content="元/GiB/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.storagePrice"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem prop="gpuPrice">
            <template #label>
              <span>GPU 单价</span>
              <ElTooltip content="元/卡/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.gpuPrice"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="20">
        <ElCol :span="12">
          <ElFormItem prop="podPrice">
            <template #label>
              <span>Pod 单价</span>
              <ElTooltip content="元/个/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.podPrice"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem prop="managementFee">
            <template #label>
              <span>管理费</span>
              <ElTooltip content="元/小时" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElInputNumber
              v-model="formData.managementFee"
              :precision="4"
              :step="0.0001"
              :min="0"
              :controls="false"
              placeholder="请输入"
              :disabled="loading"
              style="width: 100%"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>

      <ElRow :gutter="20" v-if="dialogType === 'edit'">
        <ElCol :span="12">
          <ElFormItem prop="isGenerateBilling">
            <template #label>
              <span>重新生成账单</span>
              <ElTooltip content="修改配置后是否重新生成关联的账单" placement="top">
                <ElIcon class="label-tip-icon"><QuestionFilled /></ElIcon>
              </ElTooltip>
            </template>
            <ElSwitch
              v-model="formData.isGenerateBilling"
              :disabled="loading"
              inline-prompt
              active-text="是"
              inactive-text="否"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>
    </ElForm>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleClose" :disabled="loading">取消</ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="loading">
          {{
            loading
              ? dialogType === 'add'
                ? '新增中...'
                : '更新中...'
              : dialogType === 'add'
                ? '新增'
                : '更新'
          }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, nextTick } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { QuestionFilled } from '@element-plus/icons-vue'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    addBillingPriceConfigApi,
    updateBillingPriceConfigApi,
    type BillingPriceConfig,
    type BillingPriceConfigAddRequest,
    type BillingPriceConfigUpdateRequest
  } from '@/api/manager/billing'

  interface Props {
    modelValue: boolean
    dialogType: 'add' | 'edit'
    configData?: BillingPriceConfig
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'success'): void
  }

  const props = withDefaults(defineProps<Props>(), {
    modelValue: false,
    dialogType: 'add',
    configData: undefined
  })

  const emit = defineEmits<Emits>()

  // 对话框显示控制
  const visible = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })

  // 表单实例
  const formRef = ref<FormInstance>()
  const loading = ref(false)

  // 表单数据
  const formData = reactive({
    id: 0,
    configName: '',
    description: '',
    cpuPrice: 0,
    memoryPrice: 0,
    storagePrice: 0,
    gpuPrice: 0,
    podPrice: 0,
    managementFee: 0,
    isGenerateBilling: false
  })

  // 表单验证规则
  const rules: FormRules = {
    configName: [
      { required: true, message: '请输入配置名称', trigger: 'blur' },
      { min: 1, max: 100, message: '长度在 1 到 100 个字符', trigger: 'blur' }
    ],
    cpuPrice: [{ required: true, message: '请输入 CPU 单价', trigger: 'blur' }],
    memoryPrice: [{ required: true, message: '请输入内存单价', trigger: 'blur' }],
    storagePrice: [{ required: true, message: '请输入存储单价', trigger: 'blur' }],
    gpuPrice: [{ required: true, message: '请输入 GPU 单价', trigger: 'blur' }],
    podPrice: [{ required: true, message: '请输入 Pod 单价', trigger: 'blur' }],
    managementFee: [{ required: true, message: '请输入管理费', trigger: 'blur' }]
  }

  // 初始化表单数据
  const initFormData = () => {
    if (props.dialogType === 'edit' && props.configData) {
      const config = props.configData
      Object.assign(formData, {
        id: config.id,
        configName: config.configName,
        description: config.description || '',
        cpuPrice: config.cpuPrice,
        memoryPrice: config.memoryPrice,
        storagePrice: config.storagePrice,
        gpuPrice: config.gpuPrice,
        podPrice: config.podPrice,
        managementFee: config.managementFee,
        isGenerateBilling: false
      })
    } else {
      // 新增时重置表单
      Object.assign(formData, {
        id: 0,
        configName: '',
        description: '',
        cpuPrice: 0,
        memoryPrice: 0,
        storagePrice: 0,
        gpuPrice: 0,
        podPrice: 0,
        managementFee: 0,
        isGenerateBilling: false
      })
    }
  }

  // 监听对话框状态变化
  watch(
    () => props.modelValue,
    async (newVal) => {
      if (newVal) {
        // 初始化表单
        initFormData()
        // 清除验证
        nextTick(() => {
          formRef.value?.clearValidate()
        })
      }
    }
  )

  // 关闭对话框
  const handleClose = () => {
    if (loading.value) {
      return
    }
    visible.value = false
    formRef.value?.resetFields()
  }

  // 提交表单
  const handleSubmit = async () => {
    if (!formRef.value) return

    try {
      await formRef.value.validate()

      // 编辑时进行二次确认
      if (props.dialogType === 'edit') {
        await ElMessageBox.confirm(
          `确定要保存对收费配置「${formData.configName}」的修改吗？`,
          '确认修改',
          {
            confirmButtonText: '确定',
            cancelButtonText: '取消',
            type: 'warning'
          }
        )
      }

      loading.value = true

      if (props.dialogType === 'add') {
        // 新增配置
        const addData: BillingPriceConfigAddRequest = {
          configName: formData.configName,
          description: formData.description || undefined,
          cpuPrice: formData.cpuPrice,
          memoryPrice: formData.memoryPrice,
          storagePrice: formData.storagePrice,
          gpuPrice: formData.gpuPrice,
          podPrice: formData.podPrice,
          managementFee: formData.managementFee
        }
        await addBillingPriceConfigApi(addData)
        ElMessage.success('新增成功')
      } else {
        // 更新配置
        const updateData: BillingPriceConfigUpdateRequest = {
          id: formData.id,
          configName: formData.configName,
          description: formData.description || undefined,
          cpuPrice: formData.cpuPrice,
          memoryPrice: formData.memoryPrice,
          storagePrice: formData.storagePrice,
          gpuPrice: formData.gpuPrice,
          podPrice: formData.podPrice,
          managementFee: formData.managementFee,
          isGenerateBilling: formData.isGenerateBilling
        }
        await updateBillingPriceConfigApi(updateData)
        ElMessage.success('更新成功')
      }

      emit('success')
      visible.value = false
    } catch (error: any) {
      if (error === 'cancel') {
        // 用户取消了确认
        return
      } else if (error === false) {
        // 表单验证失败
      } else {
        console.error('提交失败:', error)
      }
    } finally {
      loading.value = false
    }
  }
</script>

<style lang="scss" scoped>
  .dialog-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  .label-tip-icon {
    margin-left: 4px;
    color: var(--el-text-color-secondary);
    cursor: help;
  }
</style>
