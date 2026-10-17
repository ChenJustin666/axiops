<!-- modules/pv-dialog.vue -->
<template>
  <ElDialog
    v-model="visible"
    :title="dialogTitle"
    width="1200px"
    align-center
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @close="handleClose"
  >
    <!-- 状态提示 -->
    <ElAlert
      v-if="pvStatus && pvStatus.phase === 'Bound'"
      title="此 PV 已被绑定"
      type="warning"
      :closable="false"
      show-icon
      style="margin-bottom: 16px"
    >
      <template #default>
        <div>
          <p
            >PV 已绑定到 PVC：<strong>{{ pvStatus.claimRef }}</strong></p
          >
          <p style="margin-top: 8px; color: #e6a23c">
            ⚠️ 已绑定的 PV 只能修改标签、注解和回收策略
          </p>
        </div>
      </template>
    </ElAlert>

    <!-- 加载中 -->
    <div v-if="initialLoading" class="loading-container" v-loading="true">
      <div style="height: 400px"></div>
    </div>

    <!-- 主内容 -->
    <div v-else>
      <!-- 模式切换 -->
      <div class="mode-switcher">
        <ElRadioGroup v-model="editMode" size="small">
          <ElRadioButton value="form">
            <FormInputIcon :size="14" />
            表单模式
          </ElRadioButton>
          <ElRadioButton value="yaml">
            <FileCodeIcon :size="14" />
            YAML 模式
          </ElRadioButton>
        </ElRadioGroup>
      </div>

      <!-- 转换错误 -->
      <ElAlert
        v-if="convertError"
        :title="`转换错误: ${convertError}`"
        type="error"
        :closable="true"
        @close="convertError = ''"
        style="margin-bottom: 16px"
        show-icon
      />

      <!-- 表单模式 -->
      <div v-show="editMode === 'form'" class="form-container">
        <PVForm
          ref="formRef"
          v-model="formData"
          :disabled="loading || isBound"
          :is-edit="dialogType === 'edit'"
          :storage-classes="storageClasses"
        />
      </div>

      <!-- YAML 模式 -->
      <div v-show="editMode === 'yaml'" class="yaml-container">
        <YamlEditorPro
          v-model="yamlContent"
          :readonly="loading"
          :filename="`${formData.name || 'pv'}.yaml`"
          :show-toolbar="true"
          :show-status-bar="true"
          height="600px"
          @valid-change="handleYamlValidChange"
        />
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <div class="footer-info">
          <ElTag v-if="pvStatus" :type="getStatusType(pvStatus.phase)">
            {{ pvStatus.phase }}
          </ElTag>
        </div>
        <div class="footer-actions">
          <ElButton @click="handleClose" :disabled="loading">取消</ElButton>
          <ElButton
            type="primary"
            @click="handleSubmit"
            :loading="loading"
            :disabled="editMode === 'yaml' && !isYamlValid"
          >
            {{ submitText }}
          </ElButton>
        </div>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch, nextTick } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { FormInput as FormInputIcon, FileCode as FileCodeIcon } from 'lucide-vue-next'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import PVForm from './pv-form.vue'
  import {
    createPVApi,
    updatePVApi,
    getPVYamlApi,
    getStorageClassListApi
  } from '@/api/workload/cluster-resource'
  import { pvYamlToForm, pvFormToYaml, createEmptyPVForm, validatePVForm } from './type'
  import type { PVFormData } from './type'

  interface Props {
    modelValue: boolean
    dialogType: 'create' | 'edit'
    clusterUuid: string
    resourceName?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    resourceName: ''
  })

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    success: []
  }>()

  // 编辑模式下暂存原始 YAML（用于 merge 策略）
  const rawSourceYaml = ref('')

  // 状态
  const editMode = ref<'form' | 'yaml'>('form')
  const formData = ref<PVFormData>(createEmptyPVForm())
  const yamlContent = ref('')
  const loading = ref(false)
  const initialLoading = ref(false)
  const isYamlValid = ref(true)
  const convertError = ref('')
  const pvStatus = ref<{
    phase: string
    claimRef?: string
  } | null>(null)
  const storageClasses = ref<
    Array<{
      name: string
      provisioner: string
      isDefault: boolean
    }>
  >([])

  const formRef = ref()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const dialogTitle = computed(() => {
    const prefix = props.dialogType === 'create' ? '创建' : '编辑'
    if (props.dialogType === 'edit' && props.resourceName) {
      return `${prefix} PersistentVolume: ${props.resourceName}`
    }
    return `${prefix} PersistentVolume`
  })

  const submitText = computed(() => {
    return props.dialogType === 'create' ? '创建' : '保存'
  })

  const isBound = computed(() => {
    return pvStatus.value?.phase === 'Bound'
  })

  const getStatusType = (phase: string) => {
    const map: Record<string, string> = {
      Available: 'success',
      Bound: 'primary',
      Released: 'warning',
      Failed: 'danger',
      Pending: 'info'
    }
    return map[phase] || 'info'
  }

  // 加载 StorageClass 列表
  const loadStorageClasses = async () => {
    try {
      const response = await getStorageClassListApi({
        clusterUuid: props.clusterUuid
      })
      storageClasses.value = (response.items || []).map((sc: any) => ({
        name: sc.name,
        provisioner: sc.provisioner,
        isDefault: sc.isDefault || false
      }))
    } catch (error) {
      console.error('加载 StorageClass 列表失败:', error)
    }
  }

  // 表单 -> YAML（自动同步）
  const formToYaml = () => {
    try {
      convertError.value = ''
      const isEdit = props.dialogType === 'edit'
      yamlContent.value = pvFormToYaml(formData.value, isEdit)
    } catch (error: any) {
      convertError.value = error.message
      console.error('表单转 YAML 失败:', error)
    }
  }

  // YAML -> 表单
  const yamlToForm = () => {
    try {
      convertError.value = ''
      const parsed = pvYamlToForm(yamlContent.value)
      formData.value = { ...parsed }
    } catch (error: any) {
      convertError.value = error.message
      throw error
    }
  }

  // 监听表单变化，自动同步到 YAML
  watch(
    () => [
      formData.value.name,
      formData.value.capacity,
      formData.value.capacityUnit,
      formData.value.accessModes,
      formData.value.reclaimPolicy,
      formData.value.volumeMode,
      formData.value.storageClassName,
      formData.value.sourceType,
      formData.value.nfs,
      formData.value.hostPath,
      formData.value.csi,
      formData.value.local,
      formData.value.nodeAffinity,
      formData.value.mountOptions,
      formData.value.labels,
      formData.value.annotations
    ],
    () => {
      if (editMode.value === 'form' && !initialLoading.value) {
        formToYaml()
      }
    },
    { deep: true }
  )

  // 监听弹窗打开
  watch(
    () => props.modelValue,
    async (val) => {
      if (val) {
        console.log('🚀 弹窗打开，类型:', props.dialogType)

        // 重置状态
        convertError.value = ''
        pvStatus.value = null
        editMode.value = 'form'
        isYamlValid.value = true

        // 显示加载状态
        initialLoading.value = true

        try {
          // 加载 StorageClass 列表
          await loadStorageClasses()

          if (props.dialogType === 'edit' && props.resourceName) {
            // 编辑模式：加载现有 PV 数据
            await loadExistingPV()
          } else {
            // 创建模式：初始化空表单
            formData.value = createEmptyPVForm()
            await nextTick()
            formToYaml() // 初始化 YAML
          }
        } catch (error: any) {
          console.error('初始化失败:', error)
          handleClose()
        } finally {
          initialLoading.value = false
        }
      }
    }
  )

  // 加载现有 PV 数据（编辑模式）
  const loadExistingPV = async () => {
    console.log('📥 加载 PV 数据:', props.resourceName)

    // 1. 请求 YAML 数据
    const response = await getPVYamlApi({
      clusterUuid: props.clusterUuid,
      name: props.resourceName!
    })

    console.log('✅ YAML 数据获取成功')

    // 2. 保存原始 YAML（用于 merge 策略）
    rawSourceYaml.value = response
    yamlContent.value = response

    // 3. 转为表单数据
    console.log('🔄 转换 YAML 到表单数据')
    const parsed = pvYamlToForm(response)

    // 4. 更新表单数据
    formData.value = { ...parsed }

    // 5. 提取状态信息
    if (parsed._status) {
      pvStatus.value = {
        phase: parsed._status.phase || 'Available',
        claimRef: parsed._status.claimRef
          ? `${parsed._status.claimRef.namespace}/${parsed._status.claimRef.name}`
          : undefined
      }
    }

    console.log('✅ 表单数据加载完成:', {
      name: formData.value.name,
      sourceType: formData.value.sourceType,
      phase: pvStatus.value?.phase
    })

    // 如果 PV 已绑定，切换到 YAML 模式并提示
    if (isBound.value) {
      editMode.value = 'yaml'
      ElMessage.warning({
        message: '此 PV 已被绑定，只能在 YAML 模式下修改有限的字段',
        duration: 5000
      })
    }

    await nextTick()
  }

  // YAML 验证
  const handleYamlValidChange = (valid: boolean) => {
    isYamlValid.value = valid
  }

  // 关闭
  const handleClose = () => {
    if (loading.value) return
    visible.value = false
  }

  // 提交（统一用表单数据）
  const handleSubmit = async () => {
    // 防止重复提交
    if (loading.value) return

    // YAML 模式：先转回表单
    if (editMode.value === 'yaml') {
      if (!isYamlValid.value) {
        ElMessage.warning('YAML 格式错误')
        return
      }

      try {
        yamlToForm()
      } catch (error: any) {
        return
      }
    }

    // 表单模式：检查已绑定状态
    if (isBound.value && editMode.value === 'form') {
      return
    }

    // 验证表单数据（使用自定义验证）
    try {
      validatePVForm(formData.value)
    } catch (error: any) {
      ElMessage.warning(error.message || '请检查表单填写是否正确')
      return
    }

    // 确认提交
    const confirmed = await confirmSubmit()
    if (!confirmed) return

    // 表单数据转 YAML
    let finalYaml = ''
    try {
      const isEdit = props.dialogType === 'edit'
      finalYaml = pvFormToYaml(formData.value, isEdit)
    } catch (error: any) {
      ElMessage.error(`转换失败: ${error.message}`)
      return
    }

    // 提交请求
    loading.value = true
    try {
      const api = props.dialogType === 'create' ? createPVApi : updatePVApi
      await api({
        clusterUuid: props.clusterUuid,
        yamlStr: finalYaml
      })

      ElMessage.success(props.dialogType === 'create' ? 'PV 创建成功' : 'PV 更新成功')
      emit('success')
      visible.value = false
    } finally {
      loading.value = false
    }
  }

  // 确认提交
  const confirmSubmit = async (): Promise<boolean> => {
    if (props.dialogType === 'edit') {
      const message = isBound.value
        ? `
          <div class="confirm-content">
            <p class="confirm-title">此 PV 已被绑定，确定要修改吗？</p>
            <p class="confirm-warning">只能修改标签、注解和回收策略</p>
          </div>
        `
        : `
          <div class="confirm-content">
            <p class="confirm-title">确定要保存对 PV "<strong>${formData.value.name}</strong>" 的修改吗？</p>
          </div>
        `

      try {
        await ElMessageBox.confirm(message, '确认修改', {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning',
          dangerouslyUseHTMLString: true,
          customClass: 'pv-confirm-dialog'
        })
        return true
      } catch {
        return false
      }
    } else {
      const sourceTypeText =
        formData.value.sourceType === 'nfs'
          ? 'NFS'
          : formData.value.sourceType === 'hostPath'
            ? 'HostPath'
            : formData.value.sourceType === 'csi'
              ? 'CSI'
              : 'Local'

      try {
        await ElMessageBox.confirm(
          `
            <div class="confirm-content">
              <p class="confirm-title">确定要创建 PV "<strong>${formData.value.name}</strong>" 吗？</p>
              <div class="confirm-summary">
                <div class="summary-row">
                  <span class="summary-label">存储类型</span>
                  <span class="summary-value">${sourceTypeText}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-label">容量</span>
                  <span class="summary-value">${formData.value.capacity}${formData.value.capacityUnit}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-label">访问模式</span>
                  <span class="summary-value">${formData.value.accessModes.join(', ')}</span>
                </div>
                <div class="summary-row">
                  <span class="summary-label">回收策略</span>
                  <span class="summary-value">${formData.value.reclaimPolicy}</span>
                </div>
                ${formData.value.storageClassName ? `<div class="summary-row"><span class="summary-label">StorageClass</span><span class="summary-value">${formData.value.storageClassName}</span></div>` : ''}
              </div>
            </div>
          `,
          '确认创建',
          {
            confirmButtonText: '确定创建',
            cancelButtonText: '取消',
            type: 'info',
            dangerouslyUseHTMLString: true,
            customClass: 'pv-confirm-dialog'
          }
        )
        return true
      } catch {
        return false
      }
    }
  }
</script>

<style lang="scss" scoped>
  @use '@/views/cluster/resource/styles/resource-form.scss' as *;

  .loading-container {
    display: flex;
    align-items: center;
    justify-content: center;
  }

  .mode-switcher {
    @include dialog-toolbar;
    justify-content: flex-end;
    margin-bottom: 16px;

    :deep(.el-radio-group) {
      .el-radio-button__inner {
        display: flex;
        align-items: center;
        gap: 6px;
      }
    }
  }

  .form-container {
    @include form-container;
    max-height: 65vh;
  }

  .yaml-container {
    @include yaml-container;
  }

  .dialog-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .footer-info {
      flex: 1;
    }

    .footer-actions {
      display: flex;
      gap: 12px;
    }
  }
</style>

<style lang="scss">
  .pv-confirm-dialog {
    .el-message-box__content {
      padding: 16px 20px;
    }

    .confirm-content {
      .confirm-title {
        margin: 0 0 8px;
        font-size: 14px;
        color: #303133;
        line-height: 1.6;

        strong {
          color: #409eff;
        }
      }

      .confirm-warning {
        margin: 12px 0 0;
        padding: 8px 12px;
        background: #fef0f0;
        border-radius: 4px;
        font-size: 13px;
        color: #f56c6c;
      }

      .confirm-summary {
        margin-top: 16px;
        padding: 12px 16px;
        background: #f5f7fa;
        border-radius: 4px;
        border: 1px solid #e4e7ed;

        .summary-row {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 6px 0;
          border-bottom: 1px dashed #e4e7ed;

          &:last-child {
            border-bottom: none;
          }

          .summary-label {
            font-size: 13px;
            color: #909399;
          }

          .summary-value {
            font-size: 13px;
            color: #303133;
            font-weight: 500;
          }
        }
      }
    }
  }
</style>
