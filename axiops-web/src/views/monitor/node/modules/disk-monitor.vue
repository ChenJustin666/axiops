<template>
  <div class="disk-monitor" v-loading="loading">
    <Transition name="fade">
      <div v-if="hasData" class="monitor-card">
        <!-- 卡片头部 -->
        <div class="card-header">
          <div class="header-info">
            <div class="icon-badge disk">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  fill="currentColor"
                  d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9M7,18A1,1 0 0,0 6,19A1,1 0 0,0 7,20A1,1 0 0,0 8,19A1,1 0 0,0 7,18Z"
                />
              </svg>
            </div>
            <div>
              <h3>磁盘监控</h3>
              <p>文件系统与磁盘 I/O</p>
            </div>
          </div>
          <div class="status-tag" :class="diskHealthStatus.class">{{ diskHealthStatus.text }}</div>
        </div>

        <!-- 文件系统区域 -->
        <div v-if="filesystems?.length" class="filesystems-section">
          <div class="section-title">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path
                fill="currentColor"
                d="M10,4H4C2.89,4 2,4.89 2,6V18A2,2 0 0,0 4,20H20A2,2 0 0,0 22,18V8C22,6.89 21.1,6 20,6H12L10,4Z"
              />
            </svg>
            <span>文件系统</span>
            <span class="count-badge">{{ filesystems.length }}</span>
          </div>

          <div class="filesystem-grid">
            <div
              v-for="fs in filesystems"
              :key="fs.mountpoint"
              class="filesystem-item"
              :class="{
                warning: fs.current.usagePercent >= 80,
                danger: fs.current.usagePercent >= 90
              }"
            >
              <div class="fs-header">
                <div class="fs-info">
                  <span class="mountpoint">{{ fs.mountpoint }}</span>
                  <span class="fs-type">{{ fs.fsType }}</span>
                </div>
                <div class="fs-usage" :class="getUsageLevelClass(fs.current.usagePercent)">
                  {{ fs.current.usagePercent.toFixed(1) }}%
                </div>
              </div>

              <div class="fs-progress">
                <div
                  class="progress-fill"
                  :style="{ width: `${fs.current.usagePercent}%` }"
                  :class="getUsageLevelClass(fs.current.usagePercent)"
                ></div>
              </div>

              <div class="fs-details">
                <div class="detail-row">
                  <span class="label">已用</span>
                  <span class="value">{{ formatBytes(fs.current.usedBytes) }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">总计</span>
                  <span class="value">{{ formatBytes(fs.current.totalBytes) }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">可用</span>
                  <span class="value">{{ formatBytes(fs.current.availableBytes) }}</span>
                </div>
                <div class="detail-row">
                  <span class="label">Inode</span>
                  <span class="value">{{ fs.current.inodePercent.toFixed(1) }}%</span>
                </div>
              </div>

              <div class="fs-device">
                <svg viewBox="0 0 24 24" width="12" height="12">
                  <path
                    fill="currentColor"
                    d="M2,17H22V21H2V17M6.25,7H9V6H6V3H18V6H15V7H17.75C18.44,7 19,7.56 19,8.25V14.75A1.25,1.25 0 0,1 17.75,16H6.25A1.25,1.25 0 0,1 5,14.75V8.25C5,7.56 5.56,7 6.25,7Z"
                  />
                </svg>
                <span>{{ fs.device }}</span>
              </div>
            </div>
          </div>
        </div>

        <!-- 磁盘设备区域 -->
        <div v-if="devices?.length" class="devices-section">
          <div class="section-title">
            <svg viewBox="0 0 24 24" width="16" height="16">
              <path
                fill="currentColor"
                d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4Z"
              />
            </svg>
            <span>磁盘设备 I/O</span>
            <span class="count-badge">{{ devices.length }}</span>
          </div>

          <!-- 设备选择标签 -->
          <div class="device-tabs">
            <div
              v-for="device in devices"
              :key="device.device"
              class="device-tab"
              :class="{ active: activeDevice === device.device }"
              @click="handleDeviceChange(device.device)"
            >
              <svg viewBox="0 0 24 24" width="14" height="14">
                <path
                  fill="currentColor"
                  d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4Z"
                />
              </svg>
              <span>{{ device.device }}</span>
            </div>
          </div>

          <!-- 当前设备内容 -->
          <div v-if="currentDevice" class="device-content">
            <!-- 设备统计指标 -->
            <div class="metrics-grid">
              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M13,9V3.5L18.5,9M6,2C4.89,2 4,2.89 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2H6Z"
                    />
                  </svg>
                  <span>平均读取</span>
                </div>
                <div class="box-value">
                  <span class="value">{{
                    formatBytes(currentDevice.summary.avgReadBytesPerSec)
                  }}</span>
                  <span class="unit">/s</span>
                </div>
                <div class="box-sub"
                  >峰值: {{ formatBytes(currentDevice.summary.maxReadBytesPerSec) }}/s</div
                >
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M14,2H6A2,2 0 0,0 4,4V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V8L14,2M12,18L8,14H11V10H13V14H16L12,18M13,9V3.5L18.5,9H13Z"
                    />
                  </svg>
                  <span>平均写入</span>
                </div>
                <div class="box-value">
                  <span class="value">{{
                    formatBytes(currentDevice.summary.avgWriteBytesPerSec)
                  }}</span>
                  <span class="unit">/s</span>
                </div>
                <div class="box-sub"
                  >峰值: {{ formatBytes(currentDevice.summary.maxWriteBytesPerSec) }}/s</div
                >
              </div>

              <div
                class="metric-box"
                :class="{ warning: currentDevice.summary.avgIOUtilization > 70 }"
              >
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20M12,6A6,6 0 0,0 6,12A6,6 0 0,0 12,18A6,6 0 0,0 18,12A6,6 0 0,0 12,6Z"
                    />
                  </svg>
                  <span>I/O 利用率</span>
                </div>
                <div class="box-value">
                  <span class="value">{{ currentDevice.summary.avgIOUtilization.toFixed(1) }}</span>
                  <span class="unit">%</span>
                </div>
                <div class="box-sub"
                  >峰值: {{ currentDevice.summary.maxIOUtilization.toFixed(1) }}%</div
                >
                <div class="box-progress">
                  <div
                    class="progress-fill"
                    :style="{ width: `${currentDevice.summary.avgIOUtilization}%` }"
                    :class="getIOUtilClass(currentDevice.summary.avgIOUtilization)"
                  ></div>
                </div>
              </div>

              <div class="metric-box">
                <div class="box-header">
                  <svg viewBox="0 0 24 24" width="16" height="16">
                    <path
                      fill="currentColor"
                      d="M12,20A8,8 0 0,1 4,12A8,8 0 0,1 12,4A8,8 0 0,1 20,12A8,8 0 0,1 12,20M12,2A10,10 0 0,0 2,12A10,10 0 0,0 12,22A10,10 0 0,0 22,12A10,10 0 0,0 12,2M12.5,7V12.25L17,14.92L16.25,16.15L11,13V7H12.5Z"
                    />
                  </svg>
                  <span>当前状态</span>
                </div>
                <div class="box-detail">
                  <div class="detail-line">
                    <span class="label">Read:</span>
                    <span class="value">{{ currentDevice.current.readIOPS.toFixed(0) }} IOPS</span>
                  </div>
                  <div class="detail-line">
                    <span class="label">Write:</span>
                    <span class="value">{{ currentDevice.current.writeIOPS.toFixed(0) }} IOPS</span>
                  </div>
                  <div class="detail-line">
                    <span class="label">Await:</span>
                    <span class="value">{{ currentDevice.current.avgAwaitMs.toFixed(2) }} ms</span>
                  </div>
                </div>
              </div>
            </div>

            <!-- 设备图表区域 -->
            <div class="chart-area">
              <div class="chart-header">
                <span class="chart-title">读写速率与 I/O 利用率趋势</span>
                <div class="chart-legend">
                  <span class="legend-item"><i class="dot read"></i>读取速率</span>
                  <span class="legend-item"><i class="dot write"></i>写入速率</span>
                  <span class="legend-item"><i class="dot io"></i>I/O 利用率</span>
                </div>
              </div>
              <div ref="deviceChartRef" class="chart-container"></div>
            </div>
          </div>
        </div>
      </div>

      <!-- 空状态 -->
      <div v-else class="empty-state">
        <div class="empty-icon">
          <svg viewBox="0 0 24 24" width="64" height="64">
            <path
              fill="currentColor"
              d="M6,2H18A2,2 0 0,1 20,4V20A2,2 0 0,1 18,22H6A2,2 0 0,1 4,20V4A2,2 0 0,1 6,2M12,4A6,6 0 0,0 6,10C6,13.31 8.69,16 12,16A6,6 0 0,0 18,10C18,6.69 15.31,4 12,4M12,9A1,1 0 0,1 13,10A1,1 0 0,1 12,11A1,1 0 0,1 11,10A1,1 0 0,1 12,9M7,18A1,1 0 0,0 6,19A1,1 0 0,0 7,20A1,1 0 0,0 8,19A1,1 0 0,0 7,18Z"
            />
          </svg>
        </div>
        <h3>暂无磁盘监控数据</h3>
        <p>等待数据加载中...</p>
      </div>
    </Transition>
  </div>
</template>

<script setup lang="ts">
  import { ref, watch, onMounted, onUnmounted, nextTick, computed } from 'vue'
  import * as echarts from 'echarts'
  import type { NodeDiskMetrics } from '@/api/console/node-monitor'
  import { formatBytes, smartFormatTime } from '@/utils/format'

  interface Props {
    diskData?: NodeDiskMetrics
    loading?: boolean
    filesystems?: any[]
    devices?: any[]
  }

  const props = withDefaults(defineProps<Props>(), {
    diskData: undefined,
    loading: false,
    filesystems: () => [],
    devices: () => []
  })

  // 当前选中的设备
  const activeDevice = ref<string>('')
  // 图表容器引用
  const deviceChartRef = ref<HTMLElement>()
  // 图表实例
  let deviceChart: echarts.ECharts | null = null
  // 标记图表是否已初始化
  let chartInitialized = false

  // 是否有数据
  const hasData = computed(() => {
    return (props.filesystems?.length || 0) > 0 || (props.devices?.length || 0) > 0
  })

  // 当前选中的设备数据
  const currentDevice = computed(() => {
    if (!props.devices?.length || !activeDevice.value) return null
    return props.devices.find((d) => d.device === activeDevice.value)
  })

  // 磁盘健康状态
  const diskHealthStatus = computed(() => {
    if (!hasData.value) return { class: 'unknown', text: '未知' }

    // 检查文件系统使用率
    const hasHighUsage = props.filesystems?.some((fs) => fs.current.usagePercent >= 90)
    const hasWarningUsage = props.filesystems?.some((fs) => fs.current.usagePercent >= 80)

    // 检查 I/O 利用率
    const hasHighIO = props.devices?.some((d) => d.summary.avgIOUtilization > 80)
    const hasWarningIO = props.devices?.some((d) => d.summary.avgIOUtilization > 60)

    if (hasHighUsage || hasHighIO) return { class: 'danger', text: '异常' }
    if (hasWarningUsage || hasWarningIO) return { class: 'warning', text: '需关注' }
    return { class: 'success', text: '正常' }
  })

  // 获取使用率级别样式
  const getUsageLevelClass = (percent: number): string => {
    if (percent < 60) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
  }

  // 获取 I/O 利用率级别样式
  const getIOUtilClass = (percent: number): string => {
    if (percent < 50) return 'level-normal'
    if (percent < 80) return 'level-warning'
    return 'level-danger'
  }

  // 转换时间戳为毫秒
  const convertTimestamp = (timestamp: any): number => {
    if (typeof timestamp === 'string') {
      return new Date(timestamp).getTime()
    } else if (typeof timestamp === 'number') {
      // 秒级时间戳转换为毫秒
      if (timestamp < 10000000000) {
        return timestamp * 1000
      }
      return timestamp
    }
    return Date.now()
  }

  // 格式化磁盘速率
  const formatDiskRate = (value: number | undefined | null): string => {
    if (value === undefined || value === null || isNaN(value)) {
      return '0 B/s'
    }
    const result = formatBytes(value)
    if (!result || result === 'undefined' || result === 'NaN') {
      return '0 B/s'
    }
    return result + '/s'
  }

  // 初始化图表
  const initChart = () => {
    if (!deviceChartRef.value || chartInitialized) return

    deviceChart = echarts.init(deviceChartRef.value)
    chartInitialized = true

    const option: echarts.EChartsOption = {
      tooltip: {
        trigger: 'axis',
        axisPointer: { type: 'cross' },
        backgroundColor: 'rgba(255,255,255,0.98)',
        borderColor: '#eaecef',
        borderWidth: 1,
        padding: [12, 16],
        textStyle: { color: '#1f2937', fontSize: 12 },
        formatter: (params: any) => {
          if (!Array.isArray(params) || params.length === 0) return ''

          const timestamp = params[0].value[0]
          let result = `<div style="font-weight: 600; margin-bottom: 8px;">${smartFormatTime(timestamp)}</div>`

          params.forEach((item: any) => {
            const rawValue = item.value[1]
            let displayValue: string
            if (item.seriesName === 'I/O 利用率') {
              displayValue = `${(rawValue || 0).toFixed(2)}%`
            } else {
              displayValue = formatDiskRate(rawValue)
            }
            result += `<div style="margin: 4px 0; display: flex; align-items: center;">
              <span style="display: inline-block; width: 8px; height: 8px; border-radius: 50%; background: ${item.color}; margin-right: 8px;"></span>
              <span style="color: #6b7280;">${item.seriesName}: </span>
              <span style="font-weight: 600; margin-left: 4px;">${displayValue}</span>
            </div>`
          })
          return result
        }
      },
      legend: {
        show: false
      },
      grid: {
        left: '3%',
        right: '4%',
        bottom: '8%',
        top: '12%',
        containLabel: true
      },
      xAxis: {
        type: 'time',
        boundaryGap: false,
        axisLine: { lineStyle: { color: '#e5e7eb' } },
        axisLabel: {
          color: '#6b7280',
          fontSize: 11,
          formatter: (value: number) => smartFormatTime(value, 'HH:mm:ss')
        },
        splitLine: { show: true, lineStyle: { color: '#f3f4f6', type: 'dashed' } }
      },
      yAxis: [
        {
          type: 'value',
          name: '速率',
          nameLocation: 'middle',
          nameGap: 50,
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          position: 'left',
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: {
            formatter: (value: number) => formatDiskRate(value),
            color: '#6b7280',
            fontSize: 11
          },
          splitLine: { lineStyle: { color: '#f3f4f6', type: 'dashed' } }
        },
        {
          type: 'value',
          name: '利用率',
          nameLocation: 'middle',
          nameGap: 40,
          nameTextStyle: { color: '#6b7280', fontSize: 11 },
          position: 'right',
          axisLine: { show: false },
          axisTick: { show: false },
          axisLabel: {
            formatter: '{value}%',
            color: '#6b7280',
            fontSize: 11
          },
          min: 0,
          max: 100,
          splitLine: { show: false }
        }
      ],
      series: []
    }

    deviceChart.setOption(option)
  }

  // 更新图表数据
  const updateChart = () => {
    if (!deviceChart || !currentDevice.value) return

    const trendData =
      currentDevice.value.trend?.map((item: any) => ({
        ...item,
        timestampMs: convertTimestamp(item.timestamp)
      })) || []

    deviceChart.setOption({
      series: [
        {
          name: '读取速率',
          type: 'line',
          smooth: true,
          data: trendData.map((item: any) => [item.timestampMs, item.readBytesPerSec || 0]),
          symbol: 'none',
          lineStyle: { width: 2, color: '#10b981' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(16, 185, 129, 0.2)' },
              { offset: 1, color: 'rgba(16, 185, 129, 0.02)' }
            ])
          }
        },
        {
          name: '写入速率',
          type: 'line',
          smooth: true,
          data: trendData.map((item: any) => [item.timestampMs, item.writeBytesPerSec || 0]),
          symbol: 'none',
          lineStyle: { width: 2, color: '#f59e0b' },
          areaStyle: {
            color: new echarts.graphic.LinearGradient(0, 0, 0, 1, [
              { offset: 0, color: 'rgba(245, 158, 11, 0.2)' },
              { offset: 1, color: 'rgba(245, 158, 11, 0.02)' }
            ])
          }
        },
        {
          name: 'I/O 利用率',
          type: 'line',
          smooth: true,
          yAxisIndex: 1,
          data: trendData.map((item: any) => [item.timestampMs, item.ioUtilizationPercent || 0]),
          symbol: 'none',
          lineStyle: { width: 2, color: '#ef4444' }
        }
      ]
    })
  }

  // 处理设备切换
  const handleDeviceChange = (deviceName: string) => {
    if (activeDevice.value === deviceName) return
    activeDevice.value = deviceName

    nextTick(() => {
      if (!chartInitialized) {
        initChart()
      }
      updateChart()
    })
  }

  // 监听设备数据变化
  watch(
    () => props.devices,
    (newDevices) => {
      if (newDevices && newDevices.length > 0) {
        // 设置默认选中的设备
        if (!activeDevice.value || !newDevices.find((d) => d.device === activeDevice.value)) {
          activeDevice.value = newDevices[0].device
        }

        nextTick(() => {
          if (!chartInitialized && deviceChartRef.value) {
            initChart()
          }
          updateChart()
        })
      }
    },
    { immediate: true, deep: true }
  )

  // 窗口大小变化时调整图表
  const handleResize = () => {
    deviceChart?.resize()
  }

  onMounted(() => {
    window.addEventListener('resize', handleResize)
  })

  onUnmounted(() => {
    window.removeEventListener('resize', handleResize)
    if (deviceChart) {
      deviceChart.dispose()
      deviceChart = null
    }
    chartInitialized = false
  })
</script>

<style lang="scss" scoped>
  // 颜色变量
  $primary: #3b82f6;
  $success: #10b981;
  $warning: #f59e0b;
  $danger: #ef4444;
  $gray-50: #f9fafb;
  $gray-100: #f3f4f6;
  $gray-200: #e5e7eb;
  $gray-300: #d1d5db;
  $gray-400: #9ca3af;
  $gray-500: #6b7280;
  $gray-600: #4b5563;
  $gray-700: #374151;
  $gray-800: #1f2937;
  $radius: 14px;
  $radius-sm: 10px;
  $radius-xs: 6px;

  .disk-monitor {
    .monitor-card {
      background: #fff;
      border-radius: $radius;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
      overflow: hidden;
    }

    // 卡片头部
    .card-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 20px 24px;
      border-bottom: 1px solid $gray-200;

      .header-info {
        display: flex;
        align-items: center;
        gap: 14px;

        .icon-badge {
          width: 46px;
          height: 46px;
          border-radius: $radius-sm;
          display: flex;
          align-items: center;
          justify-content: center;
          color: #fff;

          &.disk {
            background: linear-gradient(135deg, $success, #059669);
          }
        }

        h3 {
          margin: 0;
          font-size: 16px;
          font-weight: 600;
          color: $gray-800;
        }

        p {
          margin: 2px 0 0;
          font-size: 12px;
          color: $gray-500;
        }
      }

      .status-tag {
        padding: 6px 14px;
        border-radius: 16px;
        font-size: 12px;
        font-weight: 600;

        &.success {
          background: rgba($success, 0.1);
          color: #059669;
        }

        &.warning {
          background: rgba($warning, 0.1);
          color: #d97706;
        }

        &.danger {
          background: rgba($danger, 0.1);
          color: #dc2626;
        }

        &.unknown {
          background: $gray-100;
          color: $gray-500;
        }
      }
    }

    // 区域标题
    .section-title {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 13px;
      font-weight: 600;
      color: $gray-600;
      margin-bottom: 16px;
      padding-bottom: 12px;
      border-bottom: 1px solid $gray-200;

      svg {
        color: $gray-400;
      }

      .count-badge {
        padding: 2px 8px;
        background: $gray-100;
        border-radius: 10px;
        font-size: 11px;
        font-weight: 500;
        color: $gray-500;
      }
    }

    // 文件系统区域
    .filesystems-section {
      padding: 20px 24px;

      .filesystem-grid {
        display: grid;
        grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
        gap: 14px;
      }

      .filesystem-item {
        padding: 16px;
        background: $gray-50;
        border-radius: $radius-sm;
        border: 1px solid $gray-200;
        transition: all 0.2s ease;

        &:hover {
          border-color: $gray-300;
        }

        &.warning {
          border-left: 3px solid $warning;
        }

        &.danger {
          border-left: 3px solid $danger;
        }

        .fs-header {
          display: flex;
          justify-content: space-between;
          align-items: flex-start;
          margin-bottom: 12px;

          .fs-info {
            display: flex;
            flex-direction: column;
            gap: 4px;

            .mountpoint {
              font-size: 14px;
              font-weight: 600;
              color: $gray-800;
            }

            .fs-type {
              display: inline-block;
              padding: 2px 6px;
              background: $gray-200;
              border-radius: 4px;
              font-size: 10px;
              color: $gray-600;
              width: fit-content;
            }
          }

          .fs-usage {
            font-size: 20px;
            font-weight: 700;

            &.level-normal {
              color: $success;
            }

            &.level-warning {
              color: $warning;
            }

            &.level-danger {
              color: $danger;
            }
          }
        }

        .fs-progress {
          height: 6px;
          background: $gray-200;
          border-radius: 3px;
          overflow: hidden;
          margin-bottom: 14px;

          .progress-fill {
            height: 100%;
            transition: width 0.4s ease;

            &.level-normal {
              background: linear-gradient(90deg, $success, #34d399);
            }

            &.level-warning {
              background: linear-gradient(90deg, $warning, #fbbf24);
            }

            &.level-danger {
              background: linear-gradient(90deg, $danger, #f87171);
            }
          }
        }

        .fs-details {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 8px;
          margin-bottom: 12px;

          .detail-row {
            display: flex;
            justify-content: space-between;
            padding: 6px 10px;
            background: #fff;
            border-radius: $radius-xs;

            .label {
              font-size: 11px;
              color: $gray-500;
            }

            .value {
              font-size: 12px;
              font-weight: 600;
              color: $gray-700;
            }
          }
        }

        .fs-device {
          display: flex;
          align-items: center;
          gap: 6px;
          padding-top: 10px;
          border-top: 1px dashed $gray-200;
          font-size: 11px;
          color: $gray-400;

          svg {
            flex-shrink: 0;
          }

          span {
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }
    }

    // 磁盘设备区域
    .devices-section {
      padding: 20px 24px;
      border-top: 1px solid $gray-200;

      .device-tabs {
        display: flex;
        gap: 8px;
        margin-bottom: 20px;
        flex-wrap: wrap;

        .device-tab {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 8px 14px;
          background: $gray-50;
          border: 1px solid $gray-200;
          border-radius: $radius-xs;
          font-size: 12px;
          font-weight: 500;
          color: $gray-600;
          cursor: pointer;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
          }

          &.active {
            background: rgba($success, 0.08);
            border-color: rgba($success, 0.3);
            color: #059669;

            svg {
              color: $success;
            }
          }

          svg {
            color: $gray-400;
          }
        }
      }

      .device-content {
        .metrics-grid {
          display: grid;
          grid-template-columns: repeat(4, 1fr);
          gap: 14px;
          margin-bottom: 20px;

          @media (max-width: 1200px) {
            grid-template-columns: repeat(2, 1fr);
          }

          @media (max-width: 640px) {
            grid-template-columns: 1fr;
          }
        }

        .metric-box {
          padding: 16px;
          background: $gray-50;
          border-radius: $radius-sm;
          border: 1px solid $gray-200;
          transition: all 0.2s ease;

          &:hover {
            border-color: $gray-300;
          }

          &.warning {
            background: rgba($warning, 0.06);
            border-color: rgba($warning, 0.25);
          }

          .box-header {
            display: flex;
            align-items: center;
            gap: 6px;
            margin-bottom: 10px;
            color: $gray-500;

            span {
              font-size: 12px;
              font-weight: 500;
            }
          }

          .box-value {
            display: flex;
            align-items: baseline;
            gap: 4px;

            .value {
              font-size: 22px;
              font-weight: 700;
              color: $gray-800;
            }

            .unit {
              font-size: 12px;
              color: $gray-400;
            }
          }

          .box-sub {
            margin-top: 6px;
            font-size: 11px;
            color: $gray-400;
          }

          .box-detail {
            .detail-line {
              display: flex;
              justify-content: space-between;
              padding: 4px 0;
              font-size: 12px;

              .label {
                color: $gray-500;
              }

              .value {
                font-weight: 600;
                color: $gray-700;
              }
            }
          }

          .box-progress {
            margin-top: 10px;
            height: 4px;
            background: $gray-200;
            border-radius: 2px;
            overflow: hidden;

            .progress-fill {
              height: 100%;
              transition: width 0.4s ease;

              &.level-normal {
                background: $success;
              }

              &.level-warning {
                background: $warning;
              }

              &.level-danger {
                background: $danger;
              }
            }
          }
        }

        // 图表区域
        .chart-area {
          padding: 18px;
          background: $gray-50;
          border-radius: $radius-sm;

          .chart-header {
            display: flex;
            justify-content: space-between;
            align-items: center;
            margin-bottom: 14px;

            .chart-title {
              font-size: 13px;
              font-weight: 600;
              color: $gray-600;
            }

            .chart-legend {
              display: flex;
              gap: 14px;

              .legend-item {
                display: flex;
                align-items: center;
                gap: 5px;
                font-size: 11px;
                color: $gray-500;

                .dot {
                  width: 8px;
                  height: 3px;
                  border-radius: 2px;

                  &.read {
                    background: $success;
                  }

                  &.write {
                    background: $warning;
                  }

                  &.io {
                    background: $danger;
                  }
                }
              }
            }
          }

          .chart-container {
            height: 280px;
            width: 100%;
          }
        }
      }
    }

    // 空状态
    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      min-height: 400px;
      background: #fff;
      border-radius: $radius;
      padding: 60px 40px;

      .empty-icon {
        color: $gray-300;
        margin-bottom: 20px;
      }

      h3 {
        margin: 0 0 10px;
        font-size: 16px;
        color: $gray-600;
        font-weight: 500;
      }

      p {
        margin: 0;
        font-size: 13px;
        color: $gray-400;
      }
    }
  }

  // 动画
  .fade-enter-active,
  .fade-leave-active {
    transition: opacity 0.3s ease;
  }

  .fade-enter-from,
  .fade-leave-to {
    opacity: 0;
  }

  // 响应式适配
  @media (max-width: 768px) {
    .disk-monitor {
      .card-header {
        flex-direction: column;
        align-items: flex-start;
        gap: 12px;
      }

      .filesystems-section {
        padding: 16px;

        .filesystem-grid {
          grid-template-columns: 1fr;
        }
      }

      .devices-section {
        padding: 16px;
      }
    }
  }
</style>
