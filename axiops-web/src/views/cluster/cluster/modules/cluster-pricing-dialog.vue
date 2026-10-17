<template>
  <ElDialog
    v-model="dialogVisible"
    title="集群定价配置"
    width="600px"
    :close-on-click-modal="false"
    :close-on-press-escape="!loading"
    :show-close="!loading"
    class="pricing-dialog"
    @close="handleClose"
  >
    <!-- 集群信息头部 -->
    <div class="cluster-header">
      <div class="cluster-avatar">
        <img v-if="cluster?.avatar" :src="cluster.avatar" alt="集群头像" />
        <Server v-else />
      </div>
      <div class="cluster-info">
        <h3 class="cluster-name">{{ cluster?.name }}</h3>
        <p class="cluster-uuid">{{ cluster?.uuid }}</p>
      </div>
    </div>

    <!-- 当前绑定状态 -->
    <div v-if="currentBinding" class="current-binding">
      <div class="binding-header">
        <CheckCircle class="binding-icon success" />
        <span>已绑定价格配置</span>
      </div>
      <div class="binding-detail">
        <span class="config-name">{{ currentConfigName }}</span>
        <ElTag type="success" size="small">生效中</ElTag>
      </div>
    </div>

    <div v-else class="no-binding">
      <AlertCircle class="binding-icon warning" />
      <span>该集群尚未绑定价格配置，请选择一个配置进行绑定</span>
    </div>

    <!-- 价格配置选择 -->
    <div class="config-section">
      <div class="section-header">
        <h4 class="section-title">
          <DollarSign />
          选择价格配置
        </h4>
        <span class="config-count">共 {{ priceConfigs.length }} 个配置</span>
      </div>

      <!-- 搜索框 -->
      <div class="search-box">
        <ElInput
          v-model="searchKeyword"
          placeholder="搜索配置名称或描述..."
          clearable
          :prefix-icon="Search"
          @clear="searchKeyword = ''"
        />
      </div>

      <!-- 配置列表 -->
      <div v-loading="configLoading" class="config-list">
        <template v-if="filteredConfigs.length > 0">
          <div
            v-for="config in filteredConfigs"
            :key="config.id"
            class="config-item"
            :class="{
              selected: selectedConfigId === config.id,
              current: currentBinding?.priceConfigId === config.id
            }"
            @click="selectConfig(config.id)"
          >
            <div class="config-radio">
              <div class="radio-inner" :class="{ checked: selectedConfigId === config.id }"></div>
            </div>
            <div class="config-content">
              <div class="config-header">
                <span class="config-name">{{ config.configName }}</span>
                <div class="config-tags">
                  <ElTag v-if="config.isSystem === 1" type="info" size="small">系统内置</ElTag>
                  <ElTag
                    v-if="currentBinding?.priceConfigId === config.id"
                    type="success"
                    size="small"
                    >当前</ElTag
                  >
                </div>
              </div>
              <p class="config-desc">{{ config.description || '暂无描述' }}</p>
              <div class="config-prices">
                <span class="price-item">
                  <Cpu class="price-icon" />
                  CPU: ¥{{ config.cpuPrice }}/核/时
                </span>
                <span class="price-item">
                  <MemoryStick class="price-icon" />
                  内存: ¥{{ config.memoryPrice }}/GiB/时
                </span>
                <span class="price-item">
                  <HardDrive class="price-icon" />
                  存储: ¥{{ config.storagePrice }}/GiB/时
                </span>
                <span class="price-item">
                  <Layers class="price-icon" />
                  GPU: ¥{{ config.gpuPrice }}/卡/时
                </span>
                <span class="price-item">
                  <Box class="price-icon" />
                  Pod: ¥{{ config.podPrice }}/个/时
                </span>
                <span class="price-item">
                  <Settings class="price-icon" />
                  管理费: ¥{{ config.managementFee }}/时
                </span>
              </div>
            </div>
          </div>
        </template>

        <!-- 搜索无结果 -->
        <div v-else-if="!configLoading && searchKeyword" class="empty-result">
          <SearchX class="empty-icon" />
          <p>未找到匹配 "{{ searchKeyword }}" 的配置</p>
          <ElButton type="primary" link @click="searchKeyword = ''">清除搜索</ElButton>
        </div>

        <!-- 无数据 -->
        <ElEmpty
          v-else-if="!configLoading && priceConfigs.length === 0"
          description="暂无可用的价格配置"
        />
      </div>
    </div>

    <!-- 计费开始时间 -->
    <div v-if="!currentBinding && selectedConfigId" class="billing-time-section">
      <h4 class="section-title">
        <Calendar />
        计费开始时间
      </h4>
      <ElDatePicker
        v-model="billingStartTime"
        type="datetime"
        placeholder="选择计费开始时间"
        :disabled-date="disabledDate"
        style="width: 100%"
        value-format="x"
      />
      <p class="time-hint">留空则默认从当前时间开始计费</p>
    </div>

    <!-- 操作选项 -->
    <div
      v-if="currentBinding && selectedConfigId && selectedConfigId !== currentBinding.priceConfigId"
      class="options-section"
    >
      <ElCheckbox v-model="isGenerateBilling"> 变更配置后立即生成截止账单 </ElCheckbox>
      <p class="option-hint">勾选后将自动生成从上次计费时间到现在的账单</p>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleClose" :disabled="loading">取消</ElButton>
        <ElButton
          v-if="currentBinding"
          type="danger"
          :loading="unbindLoading"
          :disabled="loading"
          @click="handleUnbind"
        >
          <Unlink v-if="!unbindLoading" />
          解除绑定
        </ElButton>
        <ElButton
          type="primary"
          :loading="bindLoading"
          :disabled="
            !selectedConfigId ||
            loading ||
            (currentBinding && selectedConfigId === currentBinding.priceConfigId)
          "
          @click="handleBind"
        >
          <Link v-if="!bindLoading" />
          {{ currentBinding ? '更换配置' : '确认绑定' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessageBox } from 'element-plus'
  import {
    Server,
    DollarSign,
    Cpu,
    MemoryStick,
    HardDrive,
    CheckCircle,
    AlertCircle,
    Calendar,
    Link,
    Unlink,
    Search,
    SearchX,
    Box,
    Layers,
    Settings
  } from 'lucide-vue-next'
  import type { Cluster } from '@/api/manager/cluster'
  import {
    searchBillingPriceConfigApi,
    getBillingConfigBindingApi,
    addBillingConfigBindingApi,
    updateBillingConfigBindingApi,
    deleteBillingConfigBindingApi,
    type BillingPriceConfig,
    type BillingConfigBinding
  } from '@/api/manager/billing'

  interface Props {
    modelValue: boolean
    cluster: Cluster | null
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
    (e: 'bindSuccess'): void
  }

  const props = defineProps<Props>()
  const emit = defineEmits<Emits>()

  // 弹窗显示状态
  const dialogVisible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  // 加载状态
  const configLoading = ref(false)
  const bindLoading = ref(false)
  const unbindLoading = ref(false)
  const loading = computed(() => configLoading.value || bindLoading.value || unbindLoading.value)

  // 数据
  const priceConfigs = ref<BillingPriceConfig[]>([])
  const currentBinding = ref<BillingConfigBinding | null>(null)
  const selectedConfigId = ref<number | null>(null)
  const billingStartTime = ref<number | null>(null)
  const isGenerateBilling = ref(true)

  // 搜索关键词
  const searchKeyword = ref('')

  // 过滤后的配置列表
  const filteredConfigs = computed(() => {
    if (!searchKeyword.value.trim()) {
      return priceConfigs.value
    }
    const keyword = searchKeyword.value.toLowerCase().trim()
    return priceConfigs.value.filter(
      (config) =>
        config.configName.toLowerCase().includes(keyword) ||
        (config.description && config.description.toLowerCase().includes(keyword))
    )
  })

  // 当前配置名称
  const currentConfigName = computed(() => {
    if (!currentBinding.value) return ''
    const config = priceConfigs.value.find((c) => c.id === currentBinding.value?.priceConfigId)
    return config?.configName || '未知配置'
  })

  // 禁用未来的日期
  const disabledDate = (date: Date) => {
    return date.getTime() > Date.now()
  }

  // 加载价格配置列表
  const loadPriceConfigs = async () => {
    configLoading.value = true
    try {
      const res = await searchBillingPriceConfigApi()
      priceConfigs.value = res.data || []
    } finally {
      configLoading.value = false
    }
  }

  // 判断绑定数据是否有效
  const isValidBinding = (data: BillingConfigBinding | null | undefined): boolean => {
    // id 为 0 或不存在都表示没有绑定
    return !!(data && data.id && data.id > 0)
  }

  // 加载当前绑定
  const loadCurrentBinding = async () => {
    if (!props.cluster?.uuid) return

    try {
      const res = await getBillingConfigBindingApi({
        bindingType: 'cluster',
        bindingClusterUuid: props.cluster.uuid
      })

      // 判断返回数据是否有效（id > 0 才表示有绑定）
      if (isValidBinding(res.data)) {
        currentBinding.value = res.data
        selectedConfigId.value = res.data!.priceConfigId
      } else {
        // id 为 0 或空数据，表示没有绑定
        currentBinding.value = null
        selectedConfigId.value = null
      }
    } catch {
      // 404 或其他错误表示没有绑定
      currentBinding.value = null
      selectedConfigId.value = null
    }
  }

  // 选择配置
  const selectConfig = (configId: number) => {
    if (loading.value) return
    selectedConfigId.value = configId
  }

  // 绑定/更新配置
  const handleBind = async () => {
    if (!selectedConfigId.value || !props.cluster?.uuid) return

    bindLoading.value = true
    try {
      if (currentBinding.value) {
        // 更新绑定
        await updateBillingConfigBindingApi({
          id: currentBinding.value.id,
          priceConfigId: selectedConfigId.value,
          isGenerateBilling: isGenerateBilling.value
        })
      } else {
        // 新增绑定
        await addBillingConfigBindingApi({
          bindingType: 'cluster',
          bindingClusterUuid: props.cluster.uuid,
          priceConfigId: selectedConfigId.value,
          billingStartTime: billingStartTime.value
            ? Math.floor(billingStartTime.value / 1000)
            : undefined
        })
      }

      // 关闭弹窗
      dialogVisible.value = false
      // 通知父组件
      emit('bindSuccess')
    } finally {
      bindLoading.value = false
    }
  }

  // 解除绑定
  const handleUnbind = async () => {
    if (!currentBinding.value) return

    try {
      await ElMessageBox.confirm(
        '解除绑定后，该集群将不再计费。确定要解除绑定吗？',
        '解除绑定确认',
        {
          confirmButtonText: '确定解除',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      unbindLoading.value = true
      await deleteBillingConfigBindingApi(currentBinding.value.id)
      // 关闭弹窗
      dialogVisible.value = false
      // 通知父组件
      emit('bindSuccess')
    } catch (error: any) {
      // 用户取消操作，不做处理
      if (error === 'cancel') return
      // 其他错误交给全局处理器
    } finally {
      unbindLoading.value = false
    }
  }

  // 关闭弹窗
  const handleClose = () => {
    if (loading.value) return
    dialogVisible.value = false
  }

  // 重置状态
  const resetState = () => {
    priceConfigs.value = []
    currentBinding.value = null
    selectedConfigId.value = null
    billingStartTime.value = null
    isGenerateBilling.value = true
    searchKeyword.value = ''
  }

  // 监听弹窗打开
  watch(dialogVisible, async (val) => {
    if (val && props.cluster) {
      resetState()
      await Promise.all([loadPriceConfigs(), loadCurrentBinding()])
    }
  })
</script>

<style lang="scss" scoped>
  .pricing-dialog {
    :deep(.el-dialog__body) {
      padding: 20px 24px;
      max-height: 70vh;
      overflow-y: auto;
    }
  }

  .cluster-header {
    display: flex;
    align-items: center;
    gap: 16px;
    padding: 16px;
    background: linear-gradient(135deg, #f8fafc 0%, #f1f5f9 100%);
    border-radius: 12px;
    margin-bottom: 20px;

    .cluster-avatar {
      width: 56px;
      height: 56px;
      border-radius: 12px;
      background: white;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
      display: flex;
      align-items: center;
      justify-content: center;
      overflow: hidden;
      flex-shrink: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }

      svg {
        width: 28px;
        height: 28px;
        color: #667eea;
      }
    }

    .cluster-info {
      flex: 1;
      min-width: 0;

      .cluster-name {
        font-size: 18px;
        font-weight: 600;
        color: #1e293b;
        margin: 0 0 4px 0;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .cluster-uuid {
        font-size: 12px;
        color: #94a3b8;
        margin: 0;
        font-family: var(--ax-font-mono);
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
    }
  }

  .current-binding,
  .no-binding {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 14px 16px;
    border-radius: 10px;
    margin-bottom: 20px;
  }

  .current-binding {
    background: #f0fdf4;
    border: 1px solid #bbf7d0;
    flex-direction: column;
    align-items: flex-start;

    .binding-header {
      display: flex;
      align-items: center;
      gap: 8px;
      color: #166534;
      font-weight: 500;
    }

    .binding-detail {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-top: 8px;
      padding-left: 28px;

      .config-name {
        font-size: 15px;
        font-weight: 600;
        color: #15803d;
      }
    }
  }

  .no-binding {
    background: #fffbeb;
    border: 1px solid #fde68a;
    color: #92400e;
    font-size: 14px;
  }

  .binding-icon {
    width: 20px;
    height: 20px;
    flex-shrink: 0;

    &.success {
      color: #22c55e;
    }

    &.warning {
      color: #f59e0b;
    }
  }

  .config-section {
    margin-bottom: 20px;

    .section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;

      .config-count {
        font-size: 12px;
        color: #94a3b8;
      }
    }

    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
      font-weight: 600;
      color: #475569;
      margin: 0;

      svg {
        width: 16px;
        height: 16px;
        color: #667eea;
      }
    }

    .search-box {
      margin-bottom: 12px;

      :deep(.el-input__wrapper) {
        border-radius: 8px;
        box-shadow: 0 0 0 1px #e2e8f0 inset;

        &:hover {
          box-shadow: 0 0 0 1px #cbd5e1 inset;
        }

        &.is-focus {
          box-shadow: 0 0 0 1px #667eea inset;
        }
      }
    }

    .config-list {
      max-height: 320px;
      overflow-y: auto;
      border: 1px solid #e2e8f0;
      border-radius: 10px;
      background: #fafafa;

      &::-webkit-scrollbar {
        width: 6px;
      }

      &::-webkit-scrollbar-track {
        background: transparent;
      }

      &::-webkit-scrollbar-thumb {
        background: #cbd5e1;
        border-radius: 3px;

        &:hover {
          background: #94a3b8;
        }
      }
    }

    .config-item {
      display: flex;
      gap: 14px;
      padding: 16px;
      border-bottom: 1px solid #e2e8f0;
      cursor: pointer;
      transition: all 0.2s;
      background: white;

      &:first-child {
        border-radius: 10px 10px 0 0;
      }

      &:last-child {
        border-bottom: none;
        border-radius: 0 0 10px 10px;
      }

      &:only-child {
        border-radius: 10px;
      }

      &:hover:not(.selected) {
        background: #f8fafc;
      }

      &.selected {
        background: linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%);

        .radio-inner {
          border-color: #667eea;

          &.checked::after {
            opacity: 1;
            transform: scale(1);
          }
        }
      }

      &.current {
        position: relative;

        &::after {
          content: '';
          position: absolute;
          top: 0;
          right: 0;
          bottom: 0;
          left: 0;
          border: 2px solid #22c55e;
          border-radius: inherit;
          pointer-events: none;
        }
      }

      .config-radio {
        padding-top: 2px;
        flex-shrink: 0;

        .radio-inner {
          width: 18px;
          height: 18px;
          border: 2px solid #cbd5e1;
          border-radius: 50%;
          position: relative;
          transition: all 0.2s;

          &::after {
            content: '';
            position: absolute;
            top: 50%;
            left: 50%;
            width: 8px;
            height: 8px;
            background: #667eea;
            border-radius: 50%;
            transform: translate(-50%, -50%) scale(0);
            opacity: 0;
            transition: all 0.2s;
          }

          &.checked::after {
            opacity: 1;
            transform: translate(-50%, -50%) scale(1);
          }
        }
      }

      .config-content {
        flex: 1;
        min-width: 0;

        .config-header {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 6px;
          flex-wrap: wrap;

          .config-name {
            font-size: 15px;
            font-weight: 600;
            color: #1e293b;
          }

          .config-tags {
            display: flex;
            gap: 6px;
          }
        }

        .config-desc {
          font-size: 13px;
          color: #64748b;
          margin: 0 0 10px 0;
          line-height: 1.4;
          display: -webkit-box;
          -webkit-line-clamp: 2;
          -webkit-box-orient: vertical;
          overflow: hidden;
        }

        .config-prices {
          display: flex;
          flex-wrap: wrap;
          gap: 8px;

          .price-item {
            display: flex;
            align-items: center;
            gap: 4px;
            font-size: 11px;
            color: #64748b;
            background: rgba(255, 255, 255, 0.8);
            padding: 3px 8px;
            border-radius: 4px;
            border: 1px solid #e2e8f0;

            .price-icon {
              width: 11px;
              height: 11px;
              color: #94a3b8;
            }
          }
        }
      }
    }

    .empty-result {
      padding: 40px 20px;
      text-align: center;
      background: white;
      border-radius: 10px;

      .empty-icon {
        width: 48px;
        height: 48px;
        color: #cbd5e1;
        margin-bottom: 12px;
      }

      p {
        font-size: 14px;
        color: #64748b;
        margin: 0 0 12px 0;
      }
    }
  }

  .billing-time-section {
    margin-bottom: 20px;

    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 14px;
      font-weight: 600;
      color: #475569;
      margin: 0 0 12px 0;

      svg {
        width: 16px;
        height: 16px;
        color: #667eea;
      }
    }

    .time-hint {
      font-size: 12px;
      color: #94a3b8;
      margin: 8px 0 0 0;
    }
  }

  .options-section {
    padding: 14px 16px;
    background: #f8fafc;
    border-radius: 10px;
    margin-bottom: 20px;

    .option-hint {
      font-size: 12px;
      color: #94a3b8;
      margin: 8px 0 0 24px;
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;

    .el-button {
      display: flex;
      align-items: center;
      gap: 6px;

      svg {
        width: 14px;
        height: 14px;
      }
    }
  }
</style>
