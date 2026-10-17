<template>
  <div class="monitor-toolbar">
    <div class="toolbar-left">
      <!-- 时间范围选择 -->
      <ElSelect
        v-model="selectedRange"
        size="default"
        style="width: 140px"
        @change="handleRangeChange"
      >
        <ElOption label="最近 5 分钟" value="5m" />
        <ElOption label="最近 15 分钟" value="15m" />
        <ElOption label="最近 30 分钟" value="30m" />
        <ElOption label="最近 1 小时" value="1h" />
        <ElOption label="最近 3 小时" value="3h" />
        <ElOption label="最近 6 小时" value="6h" />
        <ElOption label="最近 12 小时" value="12h" />
        <ElOption label="最近 24 小时" value="24h" />
        <ElOption label="自定义" value="custom" />
      </ElSelect>

      <!-- 自定义时间范围 -->
      <ElDatePicker
        v-if="selectedRange === 'custom'"
        v-model="customTimeRange"
        type="datetimerange"
        range-separator="至"
        start-placeholder="开始时间"
        end-placeholder="结束时间"
        size="default"
        style="width: 380px"
        @change="handleCustomRangeChange"
      />
    </div>

    <div class="toolbar-right">
      <!-- 自动刷新 -->
      <ElCheckbox v-model="autoRefresh" @change="handleAutoRefreshChange">自动刷新</ElCheckbox>

      <!-- 刷新间隔 -->
      <ElSelect
        v-model="refreshInterval"
        size="default"
        style="width: 100px"
        :disabled="!autoRefresh"
        @change="handleRefreshIntervalChange"
      >
        <ElOption label="10秒" :value="10" />
        <ElOption label="30秒" :value="30" />
        <ElOption label="1分钟" :value="60" />
        <ElOption label="5分钟" :value="300" />
      </ElSelect>

      <!-- 刷新按钮 -->
      <ElButton :loading="loading" :icon="RefreshRight" @click="handleRefresh">刷新</ElButton>

      <!-- 导出按钮 -->
      <ElButton :icon="Download" @click="handleExport">导出</ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted, onUnmounted } from 'vue'
  import { RefreshRight, Download } from '@element-plus/icons-vue'

  interface Props {
    loading?: boolean
  }

  defineProps<Props>()

  const emit = defineEmits<{
    refresh: [start: string, end: string, silent?: boolean]
    export: []
  }>()

  // ==================== 状态 ====================
  const selectedRange = ref('15m')
  const customTimeRange = ref<[Date, Date]>()
  const autoRefresh = ref(false)
  const refreshInterval = ref(30)
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  // ==================== 方法 ====================

  /**
   * 计算时间范围
   */
  const calculateTimeRange = (range: string): { start: string; end: string } => {
    const now = new Date()
    const end = now.toISOString()
    let start: Date

    switch (range) {
      case '5m':
        start = new Date(now.getTime() - 5 * 60 * 1000)
        break
      case '15m':
        start = new Date(now.getTime() - 15 * 60 * 1000)
        break
      case '30m':
        start = new Date(now.getTime() - 30 * 60 * 1000)
        break
      case '1h':
        start = new Date(now.getTime() - 60 * 60 * 1000)
        break
      case '3h':
        start = new Date(now.getTime() - 3 * 60 * 60 * 1000)
        break
      case '6h':
        start = new Date(now.getTime() - 6 * 60 * 60 * 1000)
        break
      case '12h':
        start = new Date(now.getTime() - 12 * 60 * 60 * 1000)
        break
      case '24h':
        start = new Date(now.getTime() - 24 * 60 * 60 * 1000)
        break
      default:
        start = new Date(now.getTime() - 15 * 60 * 1000)
    }

    return {
      start: start.toISOString(),
      end
    }
  }

  /**
   * 触发刷新
   */
  const triggerRefresh = (silent: boolean = false) => {
    if (selectedRange.value === 'custom' && customTimeRange.value) {
      const [start, end] = customTimeRange.value
      emit('refresh', start.toISOString(), end.toISOString(), silent)
    } else {
      const { start, end } = calculateTimeRange(selectedRange.value)
      emit('refresh', start, end, silent)
    }
  }

  /**
   * 处理时间范围变化
   */
  const handleRangeChange = () => {
    if (selectedRange.value !== 'custom') {
      triggerRefresh()
    }
  }

  /**
   * 处理自定义时间范围变化
   */
  const handleCustomRangeChange = () => {
    if (customTimeRange.value) {
      triggerRefresh()
    }
  }

  /**
   * 处理手动刷新
   */
  const handleRefresh = () => {
    triggerRefresh(false)
  }

  /**
   * 处理导出
   */
  const handleExport = () => {
    emit('export')
  }

  /**
   * 处理自动刷新开关
   */
  const handleAutoRefreshChange = (enabled: boolean) => {
    if (enabled) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
  }

  /**
   * 处理刷新间隔变化
   */
  const handleRefreshIntervalChange = () => {
    if (autoRefresh.value) {
      stopAutoRefresh()
      startAutoRefresh()
    }
  }

  /**
   * 启动自动刷新
   */
  const startAutoRefresh = () => {
    stopAutoRefresh()
    refreshTimer = setInterval(() => {
      triggerRefresh(true)
    }, refreshInterval.value * 1000)
  }

  /**
   * 停止自动刷新
   */
  const stopAutoRefresh = () => {
    if (refreshTimer) {
      clearInterval(refreshTimer)
      refreshTimer = null
    }
  }

  // ==================== 生命周期 ====================
  onMounted(() => {
    // 初始加载
    triggerRefresh()
  })

  onUnmounted(() => {
    stopAutoRefresh()
  })

  // ==================== 暴露方法 ====================
  defineExpose({
    triggerRefresh
  })
</script>

<style lang="scss" scoped>
  .monitor-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    background: #ffffff;
    border-radius: 12px;
    box-shadow: 0 2px 12px rgba(0, 0, 0, 0.04);
    margin-bottom: 20px;
    border: 1px solid #ebeef5;

    .toolbar-left {
      display: flex;
      align-items: center;
      gap: 12px;
    }

    .toolbar-right {
      display: flex;
      align-items: center;
      gap: 12px;
    }
  }

  @media (max-width: 768px) {
    .monitor-toolbar {
      flex-direction: column;
      gap: 12px;

      .toolbar-left,
      .toolbar-right {
        width: 100%;
        justify-content: space-between;
      }
    }
  }
</style>
