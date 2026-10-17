<template>
  <ElDialog
    v-model="visible"
    :title="dialogTitle"
    :width="width"
    :top="top"
    class="pod-log-viewer"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div class="log-viewer-container">
      <div class="log-toolbar-main">
        <!-- 容器选择 -->
        <div class="control-item">
          <label class="control-label">
            <Package :size="13" />
            容器
          </label>
          <ElSelect
            v-model="selectedContainer"
            placeholder="选择容器"
            size="small"
            style="width: 200px"
            :loading="loadingContainers"
            @change="handleContainerChange"
          >
            <ElOption
              v-for="container in allContainers"
              :key="container.name"
              :label="container.name"
              :value="container.name"
            >
              <div class="container-option">
                <ElTag
                  size="small"
                  :type="
                    container.type === 'init'
                      ? 'info'
                      : container.type === 'ephemeral'
                        ? 'warning'
                        : 'success'
                  "
                >
                  {{ container.type }}
                </ElTag>
                <span>{{ container.name }}</span>
                <span class="container-state" :class="`state-${container.state}`">
                  {{ container.state }}
                </span>
              </div>
            </ElOption>
          </ElSelect>
        </div>

        <!-- 行数选择 -->
        <div class="control-item">
          <label class="control-label">
            <FileText :size="13" />
            行数
          </label>
          <ElSelect
            v-model="logOptions.tailLines"
            placeholder="行数"
            size="small"
            style="width: 100px"
            :disabled="isStreaming"
          >
            <ElOption label="100" :value="100" />
            <ElOption label="500" :value="500" />
            <ElOption label="1000" :value="1000" />
            <ElOption label="3000" :value="3000" />
            <ElOption label="5000" :value="5000" />
            <ElOption label="全部" :value="0" />
          </ElSelect>
        </div>

        <!-- 弹性空间 -->
        <div style="flex: 1"></div>

        <div class="main-actions">
          <ElButton
            v-if="!isStreaming && autoRefreshInterval === 0"
            type="primary"
            size="small"
            :loading="loadingLogs"
            @click="handleLoadLogs(false)"
          >
            <PlayCircle :size="13" style="margin-right: 4px" />
            {{ loadingLogs ? '加载中' : '加载日志' }}
          </ElButton>

          <ElButton
            v-if="!isStreaming && autoRefreshInterval > 0"
            type="danger"
            size="small"
            @click="handleStopAutoRefresh"
          >
            <PauseCircle :size="13" style="margin-right: 4px" />
            停止加载
          </ElButton>

          <!-- 实时日志 / 停止实时 -->
          <ElButton
            v-if="!isStreaming"
            type="success"
            size="small"
            :loading="connectingStream"
            @click="handleStartStream"
          >
            <Radio :size="13" style="margin-right: 4px" />
            {{ connectingStream ? '连接中' : '实时日志' }}
          </ElButton>

          <ElButton v-if="isStreaming" type="warning" size="small" @click="handleStopStream">
            <PauseCircle :size="13" style="margin-right: 4px" />
            停止实时
          </ElButton>

          <!-- 下载按钮 -->
          <ElButton
            size="small"
            :icon="Download"
            :loading="downloadingLog"
            :disabled="!logLines.length"
            @click="handleDownload"
          >
            下载
          </ElButton>

          <!-- 高级选项按钮 -->
          <ElButton
            size="small"
            :type="showAdvanced ? 'primary' : 'default'"
            @click="showAdvanced = !showAdvanced"
          >
            <Settings :size="13" style="margin-right: 4px" />
            高级
            <component
              :is="showAdvanced ? ChevronUp : ChevronDown"
              :size="13"
              style="margin-left: 4px"
            />
          </ElButton>
        </div>
      </div>

      <transition name="slide-down">
        <div v-if="showAdvanced" class="log-toolbar-advanced">
          <div class="advanced-content">
            <!-- 时间范围 -->
            <div class="control-item">
              <label class="control-label"> 时间 </label>
              <ElSelect
                v-model="logOptions.sinceSeconds"
                placeholder="时间范围"
                size="small"
                style="width: 120px"
                :disabled="isStreaming"
              >
                <ElOption label="1 分钟" :value="60" />
                <ElOption label="5 分钟" :value="300" />
                <ElOption label="15 分钟" :value="900" />
                <ElOption label="30 分钟" :value="1800" />
                <ElOption label="1 小时" :value="3600" />
                <ElOption label="3 小时" :value="10800" />
                <ElOption label="不限" :value="0" />
              </ElSelect>
            </div>

            <!-- 自动刷新间隔 -->
            <div class="control-item" v-if="!isStreaming">
              <label class="control-label"> 刷新 </label>
              <ElSelect
                v-model="autoRefreshInterval"
                placeholder="刷新间隔"
                size="small"
                style="width: 110px"
              >
                <ElOption label="关闭" :value="0" />
                <ElOption label="5 秒" :value="5000" />
                <ElOption label="10 秒" :value="10000" />
                <ElOption label="30 秒" :value="30000" />
                <ElOption label="1 分钟" :value="60000" />
              </ElSelect>
            </div>

            <!-- 选项开关 -->
            <ElCheckbox v-model="logOptions.timestamps" border size="small"> 时间戳 </ElCheckbox>

            <ElCheckbox v-model="logOptions.previous" border size="small" :disabled="isStreaming">
              上次实例
            </ElCheckbox>

            <ElCheckbox v-model="autoScroll" border size="small"> 自动滚动 </ElCheckbox>

            <!-- 清空按钮 -->
            <ElButton :icon="Trash2" size="small" @click="handleClear"> 清空 </ElButton>
          </div>
        </div>
      </transition>

      <!-- 搜索栏 -->
      <div class="log-search-bar">
        <ElInput
          v-model="searchText"
          placeholder="搜索日志... (支持正则表达式)"
          clearable
          size="small"
          style="flex: 1"
          @input="handleSearch"
        >
          <template #prefix>
            <Search :size="13" />
          </template>
        </ElInput>
        <div class="search-stats" v-if="searchText && filteredLogsCount > 0">
          <span>{{ currentMatchIndex + 1 }} / {{ filteredLogsCount }}</span>
          <ElButton
            text
            size="small"
            :icon="ChevronUp"
            :disabled="!filteredLogsCount"
            @click="handlePrevMatch"
          />
          <ElButton
            text
            size="small"
            :icon="ChevronDown"
            :disabled="!filteredLogsCount"
            @click="handleNextMatch"
          />
        </div>
      </div>

      <!-- 状态栏 -->
      <div class="log-status-bar">
        <div class="status-left">
          <div class="status-item">
            <div class="status-dot" :class="{ active: isStreaming, loading: connectingStream }" />
            <span>{{ statusText }}</span>
          </div>
          <div class="status-item">
            <FileText :size="12" />
            <span>{{ logLines.length }} 行</span>
          </div>
          <div class="status-item" v-if="logSize">
            <HardDrive :size="12" />
            <span>{{ formatSize(logSize) }}</span>
          </div>
          <div class="status-item" v-if="autoRefreshInterval > 0 && !isStreaming">
            <RefreshCw :size="12" :class="{ spinning: loadingLogs }" />
            <span>{{ Math.ceil(autoRefreshCountdown / 1000) }}s</span>
          </div>
        </div>
        <div class="status-right">
          <ElTooltip content="换行" placement="top">
            <ElButton
              text
              size="small"
              :type="wrapLines ? 'primary' : 'default'"
              @click="wrapLines = !wrapLines"
            >
              <WrapText :size="14" />
            </ElButton>
          </ElTooltip>
        </div>
      </div>

      <div
        ref="logScrollContainerRef"
        class="log-content-wrapper"
        :class="{ 'wrap-mode': wrapLines }"
        :style="{ height: logHeight }"
        @scroll="handleScroll"
      >
        <div
          class="log-content"
          v-loading="loadingLogs"
          element-loading-text="加载日志中..."
          element-loading-background="rgba(30, 30, 30, 0.8)"
        >
          <table v-if="logLines.length" class="log-table">
            <tbody>
              <tr
                v-for="(log, index) in displayLogs"
                :key="index"
                class="log-line"
                :class="{
                  highlight: log.highlight,
                  'current-match': currentMatchIndex === log.matchIndex
                }"
                @click="handleLineClick(index)"
              >
                <td class="line-number">{{ log.lineNumber }}</td>
                <td v-if="logOptions.timestamps && log.timestamp" class="line-timestamp">{{
                  log.timestamp
                }}</td>
                <td class="line-content" v-html="log.html"></td>
              </tr>
            </tbody>
          </table>

          <div v-if="!logLines.length && !loadingLogs" class="log-empty">
            <FileText :size="40" />
            <p>暂无日志数据</p>
            <p class="empty-tip">请选择容器并点击"加载日志"或"实时日志"</p>
          </div>
        </div>
      </div>
    </div>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch, nextTick, onBeforeUnmount } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    Package,
    FileText,
    Clock,
    RotateCcw,
    ScrollText,
    PlayCircle,
    Radio,
    PauseCircle,
    Download,
    Trash2,
    Search,
    ChevronUp,
    ChevronDown,
    HardDrive,
    WrapText,
    Settings,
    RefreshCw
  } from 'lucide-vue-next'
  import {
    getPodContainerListInWorkloadApi,
    getPodLogsApi,
    getPodLogsStreamUrl,
    createWebSocket,
    WSMessageType,
    type ContainerInfoList,
    type ContainerInfo,
    type PodLogsGetReq
  } from '@/api'

  interface Props {
    modelValue: boolean
    workloadId: number
    podName: string
    width?: string
    height?: string
    top?: string
  }

  interface Emits {
    (e: 'update:modelValue', value: boolean): void
  }

  const props = withDefaults(defineProps<Props>(), {
    width: '85%',
    height: '500px',
    top: '5vh'
  })

  const emit = defineEmits<Emits>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const dialogTitle = computed(() => `Pod 日志 - ${props.podName}`)

  const logHeight = computed(() => props.height)

  const showAdvanced = ref(false)

  const loadingContainers = ref(false)
  const containerData = ref<ContainerInfoList | null>(null)
  const selectedContainer = ref('')

  interface ExtendedContainer extends ContainerInfo {
    type: 'init' | 'container' | 'ephemeral'
  }

  const allContainers = computed<ExtendedContainer[]>(() => {
    if (!containerData.value) return []
    const containers: ExtendedContainer[] = []

    containerData.value.containers?.forEach((c) => {
      containers.push({ ...c, type: 'container' })
    })

    containerData.value.initContainers?.forEach((c) => {
      containers.push({ ...c, type: 'init' })
    })

    containerData.value.ephemeralContainers?.forEach((c) => {
      containers.push({ ...c, type: 'ephemeral' })
    })

    return containers
  })

  const logOptions = ref({
    tailLines: 1000,
    timestamps: false,
    previous: false,
    sinceSeconds: 0
  })

  const autoRefreshInterval = ref(0)
  const autoRefreshTimer = ref<number | null>(null)
  const autoRefreshCountdown = ref(0)
  const countdownTimer = ref<number | null>(null)

  interface LogLineData {
    content: string
    timestamp?: string
    stream?: string
  }

  const logLines = ref<LogLineData[]>([])
  const loadingLogs = ref(false)
  const logSize = ref(0)

  const isStreaming = ref(false)
  const connectingStream = ref(false)
  let ws: WebSocket | null = null
  let wsCloseRequested = ref(false)

  const searchText = ref('')
  const currentMatchIndex = ref(-1)

  const autoScroll = ref(true)
  const wrapLines = ref(false)
  const downloadingLog = ref(false)

  const logScrollContainerRef = ref<HTMLElement | null>(null)
  const userScrolling = ref(false)

  interface LogLine {
    lineNumber: number
    content: string
    timestamp?: string
    html: string
    highlight: boolean
    matchIndex?: number
  }

  const displayLogs = computed<LogLine[]>(() => {
    return logLines.value.map((line, index) => {
      let html = escapeHtml(line.content)
      let highlight = false
      let matchIndex: number | undefined

      if (searchText.value) {
        try {
          const regex = new RegExp(searchText.value, 'gi')
          const matches = line.content.match(regex)
          if (matches) {
            highlight = true
            html = line.content.replace(regex, (match) => `<mark>${escapeHtml(match)}</mark>`)
            matchIndex = filteredIndices.value.indexOf(index)
          }
        } catch (e) {
          if (line.content.toLowerCase().includes(searchText.value.toLowerCase())) {
            highlight = true
            const regex = new RegExp(escapeHtml(searchText.value), 'gi')
            html = escapeHtml(line.content).replace(regex, (match) => `<mark>${match}</mark>`)
            matchIndex = filteredIndices.value.indexOf(index)
          }
        }
      }

      return {
        lineNumber: index + 1,
        content: line.content,
        timestamp: line.timestamp,
        html,
        highlight,
        matchIndex
      }
    })
  })

  const filteredIndices = computed(() => {
    if (!searchText.value) return []
    return displayLogs.value
      .map((log, index) => (log.highlight ? index : -1))
      .filter((index) => index !== -1)
  })

  const filteredLogsCount = computed(() => filteredIndices.value.length)

  const statusText = computed(() => {
    if (connectingStream.value) return '连接中...'
    if (isStreaming.value) return '实时日志'
    if (loadingLogs.value) return '加载中...'
    if (autoRefreshInterval.value > 0) return '自动刷新'
    return '就绪'
  })

  function escapeHtml(text: string): string {
    const div = document.createElement('div')
    div.textContent = text
    return div.innerHTML
  }

  function formatSize(bytes: number): string {
    if (bytes === 0) return '0 B'
    const k = 1024
    const sizes = ['B', 'KB', 'MB', 'GB']
    const i = Math.floor(Math.log(bytes) / Math.log(k))
    return Math.round((bytes / Math.pow(k, i)) * 100) / 100 + ' ' + sizes[i]
  }

  async function loadContainers() {
    if (!props.workloadId || !props.podName) return

    loadingContainers.value = true
    try {
      const response = await getPodContainerListInWorkloadApi(props.workloadId, {
        podName: props.podName
      })
      containerData.value = response

      if (allContainers.value.length > 0) {
        selectedContainer.value = allContainers.value[0].name
      }
    } catch (error) {
      console.error('加载容器列表失败:', error)
    } finally {
      loadingContainers.value = false
    }
  }

  function handleContainerChange() {
    if (isStreaming.value) {
      handleStopStream()
    }
    stopAutoRefresh()
    logLines.value = []
  }

  async function handleLoadLogs(silent = false) {
    if (!visible.value) {
      return
    }

    if (!selectedContainer.value) {
      if (!silent) {
      }
      return
    }

    loadingLogs.value = true
    try {
      const params: PodLogsGetReq = {
        workloadId: props.workloadId,
        podName: props.podName,
        container: selectedContainer.value,
        timestamps: logOptions.value.timestamps,
        previous: logOptions.value.previous,
        tailLines: logOptions.value.tailLines || undefined,
        sinceSeconds: logOptions.value.sinceSeconds || undefined
      }

      const response = await getPodLogsApi(params)
      const logs = response.logs || ''

      logLines.value = logs
        .split('\n')
        .filter((line) => line.trim())
        .map((line) => {
          if (logOptions.value.timestamps) {
            const timestampMatch = line.match(
              /^(\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d+Z)\s+(.*)$/
            )
            if (timestampMatch) {
              return {
                timestamp: timestampMatch[1],
                content: timestampMatch[2],
                stream: 'stdout'
              }
            }
          }
          return {
            content: line,
            stream: 'stdout'
          }
        })

      logSize.value = new Blob([logs]).size

      if (!silent) {
        ElMessage.success(`加载成功，共 ${logLines.value.length} 行`)
      } else {
      }

      nextTick(() => {
        scrollToBottom()
      })
    } catch (error) {
      console.error('加载日志失败:', error)
      if (!silent) {
      }
    } finally {
      loadingLogs.value = false
    }
  }

  function startAutoRefresh() {
    if (autoRefreshInterval.value <= 0 || isStreaming.value) return

    stopAutoRefresh()

    autoRefreshCountdown.value = autoRefreshInterval.value

    countdownTimer.value = window.setInterval(() => {
      autoRefreshCountdown.value -= 1000
      if (autoRefreshCountdown.value <= 0) {
        autoRefreshCountdown.value = autoRefreshInterval.value
        handleLoadLogs(true)
      }
    }, 1000)
  }

  function stopAutoRefresh() {
    if (countdownTimer.value) {
      clearInterval(countdownTimer.value)
      countdownTimer.value = null
    }
    if (autoRefreshTimer.value) {
      clearInterval(autoRefreshTimer.value)
      autoRefreshTimer.value = null
    }
    autoRefreshCountdown.value = 0
  }

  function handleStopAutoRefresh() {
    autoRefreshInterval.value = 0
    stopAutoRefresh()
    ElMessage.info('自动刷新已停止')
  }

  watch(autoRefreshInterval, (newVal) => {
    if (newVal > 0 && visible.value && !isStreaming.value && selectedContainer.value) {
      startAutoRefresh()
    } else {
      stopAutoRefresh()
    }
  })

  watch(visible, (newVal) => {
    if (!newVal) {
      stopAutoRefresh()
      if (isStreaming.value) {
        handleStopStream()
      }
    }
  })

  function handleStartStream() {
    if (!selectedContainer.value) {
      return
    }

    if (isStreaming.value) {
      return
    }

    stopAutoRefresh()

    connectingStream.value = true
    logLines.value = []
    wsCloseRequested.value = false

    try {
      const params: any = {
        workloadId: props.workloadId,
        podName: props.podName,
        container: selectedContainer.value,
        timestamps: logOptions.value.timestamps
      }

      if (logOptions.value.tailLines && logOptions.value.tailLines > 0) {
        params.tailLines = logOptions.value.tailLines
      }

      if (logOptions.value.sinceSeconds && logOptions.value.sinceSeconds > 0) {
        params.sinceSeconds = logOptions.value.sinceSeconds
      }

      const url = getPodLogsStreamUrl(params)

      ws = createWebSocket(url, handleStreamMessage, handleStreamError, handleStreamClose)

      ws.addEventListener('open', () => {
        connectingStream.value = false
        isStreaming.value = true
        ElMessage.success('实时日志已连接')
      })
    } catch (error) {
      console.error('连接失败:', error)
      connectingStream.value = false
    }
  }

  function handleStreamMessage(data: any) {
    if (!data || !data.type) return

    switch (data.type) {
      case WSMessageType.LOG_INIT:
        break

      case WSMessageType.LOG_DATA:
        const logData = data.data
        if (logData && logData.log) {
          logLines.value.push({
            content: logData.log,
            timestamp: logData.timestamp,
            stream: logData.stream || 'stdout'
          })
          logSize.value += new Blob([logData.log]).size

          if (autoScroll.value && !userScrolling.value) {
            nextTick(() => scrollToBottom())
          }
        }
        break

      case WSMessageType.LOG_END:
        ElMessage.info('日志流已结束')
        handleStopStream()
        break

      case WSMessageType.LOG_ERROR:
      case WSMessageType.ERROR:
        console.error('[实时日志] 错误:', data.data)
        const errorMsg = data.data?.message || '日志流发生错误'

        if (!wsCloseRequested.value) {
          if (data.data?.code === 'HEARTBEAT_TIMEOUT') {
          }
        }

        handleStopStream()
        break
    }
  }

  function handleStreamError(error: Event) {
    console.error('[实时日志] WebSocket 错误:', error)
    if (!wsCloseRequested.value) {
    }
    handleStopStream()
  }

  function handleStreamClose(event: CloseEvent) {
    if (wsCloseRequested.value) {
    } else if (event.code === 1000) {
    } else if (event.code === 1006) {
      if (!wsCloseRequested.value) {
      }
    } else if (isStreaming.value && !wsCloseRequested.value) {
      ElMessage.info('实时日志连接已关闭')
    }

    isStreaming.value = false
    connectingStream.value = false
  }

  function handleStopStream() {
    wsCloseRequested.value = true

    if (ws) {
      if (ws.readyState === WebSocket.OPEN) {
        ws.close(1000, 'Client stopped streaming')
      } else if (ws.readyState === WebSocket.CONNECTING) {
        setTimeout(() => {
          if (ws && ws.readyState === WebSocket.OPEN) {
            ws.close(1000, 'Client stopped streaming')
          }
        }, 500)
      }
      ws = null
    }

    isStreaming.value = false
    connectingStream.value = false
  }

  function handleDownload() {
    if (!logLines.value.length) {
      return
    }

    downloadingLog.value = true
    try {
      const content = logLines.value
        .map((line) => {
          if (line.timestamp) {
            return `${line.timestamp} ${line.content}`
          }
          return line.content
        })
        .join('\n')

      const blob = new Blob([content], { type: 'text/plain;charset=utf-8' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${props.podName}-${selectedContainer.value}-${Date.now()}.log`
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success('日志下载成功')
    } catch (error) {
      console.error('下载失败:', error)
    } finally {
      downloadingLog.value = false
    }
  }

  function handleClear() {
    logLines.value = []
    logSize.value = 0
    searchText.value = ''
    ElMessage.success('日志已清空')
  }

  function handleSearch() {
    currentMatchIndex.value = -1
    if (filteredIndices.value.length > 0) {
      handleNextMatch()
    }
  }

  function handlePrevMatch() {
    if (!filteredIndices.value.length) return

    if (currentMatchIndex.value <= 0) {
      currentMatchIndex.value = filteredIndices.value.length - 1
    } else {
      currentMatchIndex.value--
    }

    scrollToMatch()
  }

  function handleNextMatch() {
    if (!filteredIndices.value.length) return

    if (currentMatchIndex.value >= filteredIndices.value.length - 1) {
      currentMatchIndex.value = 0
    } else {
      currentMatchIndex.value++
    }

    scrollToMatch()
  }

  function scrollToMatch() {
    if (currentMatchIndex.value === -1) return

    const lineIndex = filteredIndices.value[currentMatchIndex.value]
    const lineElement = document.querySelectorAll('.log-line')[lineIndex] as HTMLElement

    if (lineElement && logScrollContainerRef.value) {
      lineElement.scrollIntoView({ behavior: 'smooth', block: 'center' })
    }
  }

  function scrollToBottom() {
    if (logScrollContainerRef.value) {
      const container = logScrollContainerRef.value
      container.scrollTop = container.scrollHeight
    }
  }

  function handleScroll() {
    if (!logScrollContainerRef.value) return

    const container = logScrollContainerRef.value
    const { scrollTop, scrollHeight, clientHeight } = container

    const isAtBottom = scrollHeight - scrollTop - clientHeight < 10

    if (!isAtBottom) {
      userScrolling.value = true
    } else {
      userScrolling.value = false
    }
  }

  function handleLineClick(index: number) {}

  function handleClosed() {
    handleStopStream()
    stopAutoRefresh()

    logLines.value = []
    logSize.value = 0
    searchText.value = ''
    selectedContainer.value = ''
    containerData.value = null
    showAdvanced.value = false
    autoRefreshInterval.value = 0
  }

  watch(
    () => props.modelValue,
    (newVal) => {
      if (newVal) {
        loadContainers().then(() => {
          if (selectedContainer.value) {
            setTimeout(() => {
              handleStartStream()
            }, 500)
          }
        })
      }
    },
    { immediate: true }
  )

  onBeforeUnmount(() => {
    handleStopStream()
    stopAutoRefresh()
  })

  const handleBeforeUnload = () => {
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.close(1000, 'Page unloading')
    }
  }

  if (typeof window !== 'undefined') {
    window.addEventListener('beforeunload', handleBeforeUnload)
  }

  onBeforeUnmount(() => {
    if (typeof window !== 'undefined') {
      window.removeEventListener('beforeunload', handleBeforeUnload)
    }
  })
</script>

<style lang="scss" scoped>
  .pod-log-viewer {
    .log-viewer-container {
      display: flex;
      flex-direction: column;
      gap: 8px;
      height: 100%;
    }

    .log-toolbar-main {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-wrap: wrap;
      padding: 10px;
      background: #f9fafb;
      border-radius: 6px;

      .control-item {
        display: flex;
        align-items: center;
        gap: 6px;

        .control-label {
          display: flex;
          align-items: center;
          gap: 3px;
          font-size: 12px;
          font-weight: 500;
          color: #606266;
          white-space: nowrap;
        }
      }

      .main-actions {
        display: flex;
        align-items: center;
        gap: 6px;
      }

      .container-option {
        display: flex;
        align-items: center;
        gap: 8px;

        .container-state {
          font-size: 11px;
          padding: 1px 5px;
          border-radius: 3px;

          &.state-running {
            color: #67c23a;
            background: rgba(103, 194, 58, 0.1);
          }

          &.state-waiting {
            color: #e6a23c;
            background: rgba(230, 162, 60, 0.1);
          }

          &.state-terminated {
            color: #f56c6c;
            background: rgba(245, 108, 108, 0.1);
          }
        }
      }
    }

    .log-toolbar-advanced {
      padding: 10px;
      background: #f5f7fa;
      border-radius: 6px;
      overflow: hidden;

      .advanced-content {
        display: flex;
        align-items: center;
        gap: 8px;
        flex-wrap: wrap;

        .control-item {
          display: flex;
          align-items: center;
          gap: 6px;

          .control-label {
            display: flex;
            align-items: center;
            gap: 3px;
            font-size: 12px;
            font-weight: 500;
            color: #606266;
            white-space: nowrap;
          }
        }
      }
    }

    .slide-down-enter-active,
    .slide-down-leave-active {
      transition: all 0.3s ease;
      max-height: 150px;
    }

    .slide-down-enter-from,
    .slide-down-leave-to {
      max-height: 0;
      opacity: 0;
      padding-top: 0;
      padding-bottom: 0;
    }

    .log-search-bar {
      display: flex;
      align-items: center;
      gap: 8px;

      .search-stats {
        display: flex;
        align-items: center;
        gap: 6px;
        font-size: 12px;
        color: #606266;
        white-space: nowrap;
      }
    }

    .log-status-bar {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 5px 10px;
      background: #f5f7fa;
      border-radius: 4px;
      font-size: 12px;
      color: #606266;

      .status-left,
      .status-right {
        display: flex;
        align-items: center;
        gap: 10px;
      }

      .status-item {
        display: flex;
        align-items: center;
        gap: 4px;

        .status-dot {
          width: 7px;
          height: 7px;
          border-radius: 50%;
          background: #909399;

          &.active {
            background: #67c23a;
            animation: pulse 2s ease-in-out infinite;
          }

          &.loading {
            background: #409eff;
            animation: pulse 1s ease-in-out infinite;
          }
        }

        .spinning {
          animation: spin 1s linear infinite;
        }
      }
    }

    .log-content-wrapper {
      border: 1px solid #dcdfe6;
      border-radius: 6px;
      background: #1e1e1e;
      overflow: auto !important;

      &::-webkit-scrollbar {
        width: 12px;
        height: 12px;
      }

      &::-webkit-scrollbar-track {
        background: #2d2d2d;
        border-radius: 6px;
      }

      &::-webkit-scrollbar-thumb {
        background: #5a5a5a;
        border-radius: 6px;
        border: 2px solid #2d2d2d;

        &:hover {
          background: #6a6a6a;
        }
      }

      &::-webkit-scrollbar-corner {
        background: #2d2d2d;
      }

      .log-content {
        display: inline-block;
        min-width: 100%;
        padding: 8px;
        font-family: var(--ax-font-mono);
        font-size: 12px;
        line-height: 1.6;
        color: #d4d4d4;
      }

      &.wrap-mode {
        overflow-x: hidden !important; /* 换行模式下隐藏水平滚动 */

        .log-content {
          display: block;
          width: 100%;
        }

        .log-table {
          table-layout: fixed;
          width: 100%;

          .line-number {
            width: 50px;
            min-width: 50px;
            max-width: 50px;
          }

          .line-timestamp {
            width: 210px;
            min-width: 210px;
            max-width: 210px;
          }

          .line-content {
            width: auto;
            white-space: pre-wrap !important;
            word-break: break-all !important;
            overflow-wrap: break-word !important;
          }
        }
      }

      .log-table {
        border-collapse: collapse;
        border-spacing: 0;
        white-space: nowrap;

        .log-line {
          cursor: pointer;
          transition: background-color 0.15s;

          &:hover {
            background: rgba(255, 255, 255, 0.05);
          }

          &.highlight {
            background: rgba(255, 235, 59, 0.15);

            :deep(mark) {
              background: #ffeb3b;
              color: #000;
              padding: 1px 3px;
              border-radius: 2px;
              font-weight: 500;
            }
          }

          &.current-match {
            background: rgba(33, 150, 243, 0.25);

            .line-number {
              border-left: 3px solid #2196f3;
              padding-left: 9px;
            }
          }

          td {
            padding: 2px 0;
            vertical-align: top;
          }

          .line-number {
            min-width: 50px;
            padding-left: 12px;
            padding-right: 12px;
            text-align: right;
            color: #858585;
            user-select: none;
            font-size: 11px;
          }

          .line-timestamp {
            min-width: 210px;
            padding-right: 16px;
            color: #4ec9b0;
            user-select: none;
            font-size: 11px;
          }

          .line-content {
            padding-right: 20px;
            white-space: pre;
          }
        }
      }

      .log-empty {
        display: flex;
        flex-direction: column;
        align-items: center;
        justify-content: center;
        height: 400px;
        color: #909399;

        p {
          margin: 12px 0 6px;
          font-size: 14px;
        }

        .empty-tip {
          font-size: 12px;
          color: #c0c4cc;
        }
      }
    }

    :deep(.el-dialog__header) {
      border-bottom: 1px solid #e4e7ed;
      padding: 12px 16px;
    }

    :deep(.el-dialog__body) {
      padding: 6px 16px 16px 16px !important;
      overflow: hidden;
    }

    :deep(.el-scrollbar__view) {
      padding: 0;
    }
  }

  @keyframes pulse {
    0%,
    100% {
      opacity: 1;
    }
    50% {
      opacity: 0.5;
    }
  }

  @keyframes spin {
    from {
      transform: rotate(0deg);
    }
    to {
      transform: rotate(360deg);
    }
  }
</style>
