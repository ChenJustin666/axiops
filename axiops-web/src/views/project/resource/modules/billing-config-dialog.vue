<template>
  <ElDialog
    v-model="dialogVisible"
    title="费用配置"
    width="600px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div v-loading="loading" class="billing-config-content">
      <!-- 资源池信息 -->
      <div class="resource-info" v-if="resource">
        <div class="info-item">
          <span class="label">集群名称：</span>
          <span class="value">{{ resource.clusterName }}</span>
        </div>
        <div class="info-item">
          <span class="label">集群UUID：</span>
          <span class="value">{{ resource.clusterUuid }}</span>
        </div>
      </div>

      <ElDivider />

      <!-- 配置表单 -->
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
        label-position="right"
      >
        <ElFormItem label="价格配置" prop="priceConfigId">
          <ElSelect
            v-model="formData.priceConfigId"
            placeholder="请选择价格配置"
            style="width: 100%"
            :loading="priceConfigLoading"
          >
            <ElOption
              v-for="config in priceConfigList"
              :key="config.id"
              :label="config.configName"
              :value="config.id"
            >
              <div class="price-config-option">
                <span class="name">{{ config.configName }}</span>
                <span class="desc" v-if="config.description">{{ config.description }}</span>
              </div>
            </ElOption>
          </ElSelect>
        </ElFormItem>

        <ElFormItem label="计费开始时间" prop="billingStartTime">
          <ElDatePicker
            v-model="formData.billingStartTime"
            type="datetime"
            placeholder="选择计费开始时间"
            format="YYYY-MM-DD HH:mm:ss"
            value-format="x"
            style="width: 100%"
            :disabled="!!bindingData"
          />
          <div class="form-tip" v-if="bindingData">
            <ElIcon><InfoFilled /></ElIcon>
            已有绑定配置，计费开始时间不可修改
          </div>
        </ElFormItem>

        <ElFormItem label="立即生成账单">
          <ElSwitch v-model="formData.isGenerateBilling" />
          <span class="switch-tip">保存后立即生成一次账单</span>
        </ElFormItem>
      </ElForm>

      <!-- 当前选中配置的价格详情 -->
      <div class="price-detail" v-if="selectedPriceConfig">
        <div class="detail-title">
          <DollarSign :size="16" />
          <span>价格详情</span>
        </div>
        <div class="detail-grid">
          <div class="detail-item">
            <span class="label">CPU单价</span>
            <span class="value">{{ formatPrice(selectedPriceConfig.cpuPrice) }} 元/核/小时</span>
          </div>
          <div class="detail-item">
            <span class="label">内存单价</span>
            <span class="value"
              >{{ formatPrice(selectedPriceConfig.memoryPrice) }} 元/GiB/小时</span
            >
          </div>
          <div class="detail-item">
            <span class="label">存储单价</span>
            <span class="value"
              >{{ formatPrice(selectedPriceConfig.storagePrice) }} 元/GiB/小时</span
            >
          </div>
          <div class="detail-item">
            <span class="label">GPU单价</span>
            <span class="value">{{ formatPrice(selectedPriceConfig.gpuPrice) }} 元/卡/小时</span>
          </div>
          <div class="detail-item">
            <span class="label">Pod单价</span>
            <span class="value">{{ formatPrice(selectedPriceConfig.podPrice) }} 元/个/小时</span>
          </div>
          <div class="detail-item">
            <span class="label">管理费</span>
            <span class="value">{{ formatPrice(selectedPriceConfig.managementFee) }} 元/小时</span>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="submitLoading">
          {{ bindingData ? '更新配置' : '保存配置' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { InfoFilled } from '@element-plus/icons-vue'
  import { DollarSign } from 'lucide-vue-next'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    searchBillingPriceConfigApi,
    getBillingConfigBindingApi,
    addBillingConfigBindingApi,
    updateBillingConfigBindingApi,
    type BillingPriceConfig,
    type BillingConfigBinding
  } from '@/api/manager/billing'
  import type { ProjectCluster } from '@/api'

  interface Props {
    visible: boolean
    resource: ProjectCluster | null
  }

  interface Emits {
    (e: 'update:visible', value: boolean): void
    (e: 'success'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (val) => emit('update:visible', val)
  })

  // 表单相关
  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const submitLoading = ref(false)
  const priceConfigLoading = ref(false)

  // 数据
  const priceConfigList = ref<BillingPriceConfig[]>([])
  const bindingData = ref<BillingConfigBinding | null>(null)

  // 表单数据
  const formData = ref({
    priceConfigId: undefined as number | undefined,
    billingStartTime: undefined as number | undefined,
    isGenerateBilling: false
  })

  // 表单验证规则
  const formRules: FormRules = {
    priceConfigId: [{ required: true, message: '请选择价格配置', trigger: 'change' }],
    billingStartTime: [{ required: true, message: '请选择计费开始时间', trigger: 'change' }]
  }

  // 当前选中的价格配置
  const selectedPriceConfig = computed(() => {
    if (!formData.value.priceConfigId) return null
    return priceConfigList.value.find((c) => c.id === formData.value.priceConfigId)
  })

  // 格式化价格
  const formatPrice = (price: number): string => {
    return price.toFixed(4)
  }

  // 判断绑定数据是否有效
  const isValidBinding = (data: BillingConfigBinding | null | undefined): boolean => {
    // id 为 0 或不存在都表示没有绑定
    return !!(data && data.id && data.id > 0)
  }

  // 加载价格配置列表
  const loadPriceConfigList = async () => {
    priceConfigLoading.value = true
    try {
      const res = await searchBillingPriceConfigApi({})
      priceConfigList.value = res.data || []
    } catch (error) {
      priceConfigList.value = []
    } finally {
      priceConfigLoading.value = false
    }
  }

  // 加载当前绑定配置
  const loadBindingData = async () => {
    if (!props.resource) return

    loading.value = true
    try {
      const res = await getBillingConfigBindingApi({
        bindingType: 'project_cluster',
        bindingClusterUuid: props.resource.clusterUuid,
        bindingProjectId: props.resource.projectId
      })

      // 判断返回数据是否有效（id > 0 才表示有绑定）
      if (isValidBinding(res.data)) {
        bindingData.value = res.data
        // 填充表单
        formData.value.priceConfigId = res.data!.priceConfigId
        formData.value.billingStartTime = res.data!.billingStartTime * 1000 // 转毫秒
      } else {
        // id 为 0 或空数据，表示没有绑定
        bindingData.value = null
      }
    } catch (error: any) {
      // 如果是404，说明没有绑定，这是正常情况
      bindingData.value = null
    } finally {
      loading.value = false
    }
  }

  // 重置表单
  const resetForm = () => {
    formData.value = {
      priceConfigId: undefined,
      billingStartTime: Date.now(),
      isGenerateBilling: false
    }
    bindingData.value = null
    formRef.value?.resetFields()
  }

  // 关闭弹窗
  const handleClose = () => {
    dialogVisible.value = false
    resetForm()
  }

  // 提交表单
  const handleSubmit = async () => {
    if (!formRef.value || !props.resource) return

    try {
      await formRef.value.validate()
    } catch {
      return
    }

    submitLoading.value = true
    try {
      if (bindingData.value) {
        // 更新
        await updateBillingConfigBindingApi({
          id: bindingData.value.id,
          priceConfigId: formData.value.priceConfigId!,
          isGenerateBilling: formData.value.isGenerateBilling
        })
      } else {
        // 新增
        await addBillingConfigBindingApi({
          bindingType: 'project_cluster',
          bindingClusterUuid: props.resource.clusterUuid,
          bindingProjectId: props.resource.projectId,
          priceConfigId: formData.value.priceConfigId!,
          billingStartTime: Math.floor(formData.value.billingStartTime! / 1000), // 转秒
          isGenerateBilling: formData.value.isGenerateBilling
        })
      }

      ElMessage.success(bindingData.value ? '更新成功' : '保存成功')
      emit('success')
      handleClose()
    } catch (error) {
    } finally {
      submitLoading.value = false
    }
  }

  // 监听弹窗打开
  watch(
    () => props.visible,
    async (val) => {
      if (val && props.resource) {
        resetForm()
        await loadPriceConfigList()
        await loadBindingData()
      }
    }
  )
</script>

<style lang="scss" scoped>
  .billing-config-content {
    .resource-info {
      display: flex;
      flex-wrap: wrap;
      gap: 16px;
      padding: 12px 16px;
      background: #f9fafb;
      border-radius: 8px;

      .info-item {
        display: flex;
        align-items: center;
        gap: 8px;

        .label {
          color: #6b7280;
          font-size: 13px;
        }

        .value {
          color: #1f2937;
          font-weight: 500;
          font-size: 13px;
        }
      }
    }

    .form-tip {
      display: flex;
      align-items: center;
      gap: 4px;
      margin-top: 4px;
      font-size: 12px;
      color: #9ca3af;

      .el-icon {
        color: #f59e0b;
      }
    }

    .switch-tip {
      margin-left: 12px;
      font-size: 12px;
      color: #9ca3af;
    }

    .price-detail {
      margin-top: 20px;
      padding: 16px;
      background: linear-gradient(135deg, #f0f9ff, #e0f2fe);
      border-radius: 8px;
      border: 1px solid #bae6fd;

      .detail-title {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 14px;
        font-weight: 600;
        color: #0369a1;
        margin-bottom: 12px;
      }

      .detail-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 10px;

        .detail-item {
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 8px 12px;
          background: rgba(255, 255, 255, 0.8);
          border-radius: 6px;

          .label {
            font-size: 12px;
            color: #64748b;
          }

          .value {
            font-size: 12px;
            color: #0f172a;
            font-weight: 500;
          }
        }
      }
    }
  }

  .price-config-option {
    display: flex;
    flex-direction: column;
    gap: 2px;
    padding: 4px 0;

    .name {
      font-size: 13px;
      color: #1f2937;
    }

    .desc {
      font-size: 11px;
      color: #9ca3af;
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }
</style>
