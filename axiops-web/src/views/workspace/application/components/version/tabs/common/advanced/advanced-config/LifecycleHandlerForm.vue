<template>
  <div class="lifecycle-handler-form">
    <!-- 处理器类型选择 -->
    <div class="handler-type">
      <span class="type-label">执行类型：</span>
      <ElRadioGroup v-model="localType" :disabled="!editing" size="small" @change="onTypeChange">
        <ElRadioButton value="exec">
          <span class="radio-content"><Terminal :size="14" /><span>Exec</span></span>
        </ElRadioButton>
        <ElRadioButton value="httpGet">
          <span class="radio-content"><Globe :size="14" /><span>HTTP</span></span>
        </ElRadioButton>
        <ElRadioButton value="tcpSocket">
          <span class="radio-content"><Network :size="14" /><span>TCP</span></span>
        </ElRadioButton>
        <ElRadioButton value="sleep">
          <span class="radio-content"><Clock :size="14" /><span>Sleep</span></span>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- Exec 配置 -->
    <div v-if="localType === 'exec'" class="handler-config">
      <div class="form-item-full">
        <label>命令（每行一个参数）</label>
        <ElInput
          v-model="execCommandStr"
          :disabled="!editing"
          type="textarea"
          :rows="3"
          placeholder="示例：&#10;/bin/sh&#10;-c&#10;echo done"
          @change="onFieldChange"
        />
      </div>
    </div>

    <!-- HTTP 配置 -->
    <div v-else-if="localType === 'httpGet'" class="handler-config">
      <div class="form-row">
        <div class="form-item">
          <label>路径</label>
          <ElInput
            v-model="httpConfig.path"
            :disabled="!editing"
            placeholder="/healthz"
            @change="onFieldChange"
          />
        </div>
        <div class="form-item" style="max-width: 120px">
          <label>端口</label>
          <ElInputNumber
            v-model="httpConfig.port"
            :disabled="!editing"
            :min="1"
            :max="65535"
            controls-position="right"
            @change="onFieldChange"
          />
        </div>
        <div class="form-item" style="max-width: 100px">
          <label>协议</label>
          <ElSelect v-model="httpConfig.scheme" :disabled="!editing" @change="onFieldChange">
            <ElOption label="HTTP" value="HTTP" />
            <ElOption label="HTTPS" value="HTTPS" />
          </ElSelect>
        </div>
        <div class="form-item">
          <label>Host（可选）</label>
          <ElInput
            v-model="httpConfig.host"
            :disabled="!editing"
            placeholder="默认 Pod IP"
            clearable
            @change="onFieldChange"
          />
        </div>
      </div>
    </div>

    <!-- TCP 配置 -->
    <div v-else-if="localType === 'tcpSocket'" class="handler-config">
      <div class="form-row">
        <div class="form-item" style="max-width: 150px">
          <label>端口</label>
          <ElInputNumber
            v-model="tcpConfig.port"
            :disabled="!editing"
            :min="1"
            :max="65535"
            controls-position="right"
            @change="onFieldChange"
          />
        </div>
        <div class="form-item">
          <label>Host（可选）</label>
          <ElInput
            v-model="tcpConfig.host"
            :disabled="!editing"
            placeholder="默认 Pod IP"
            clearable
            @change="onFieldChange"
          />
        </div>
      </div>
    </div>

    <!-- Sleep 配置 -->
    <div v-else-if="localType === 'sleep'" class="handler-config">
      <div class="form-row">
        <div class="form-item" style="max-width: 200px">
          <label>睡眠时间（秒）</label>
          <ElInputNumber
            v-model="sleepSeconds"
            :disabled="!editing"
            :min="0"
            :max="3600"
            controls-position="right"
            @change="onFieldChange"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, watch } from 'vue'
  import { Terminal, Globe, Network, Clock } from 'lucide-vue-next'
  import type { LifecycleHandler } from '@/api'

  interface Props {
    handler: LifecycleHandler
    editing: boolean
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ 'update:handler': [handler: LifecycleHandler] }>()

  // 同步控制 - 使用版本号避免循环更新
  let propsVersion = 0
  let localVersion = 0

  // 本地状态
  const localType = ref('exec')
  const execCommandStr = ref('')
  const httpConfig = reactive({ path: '/', port: 80, scheme: 'HTTP', host: '' })
  const tcpConfig = reactive({ port: 80, host: '' })
  const sleepSeconds = ref(5)

  // 从 props 同步
  const syncFromProps = () => {
    propsVersion++
    const currentPropsVersion = propsVersion

    localType.value = props.handler.type || 'exec'

    if (props.handler.exec) {
      execCommandStr.value = (props.handler.exec.command || []).join('\n')
    } else {
      execCommandStr.value = ''
    }

    if (props.handler.httpGet) {
      httpConfig.path = props.handler.httpGet.path || '/'
      httpConfig.port = props.handler.httpGet.port || 80
      httpConfig.scheme = props.handler.httpGet.scheme || 'HTTP'
      httpConfig.host = props.handler.httpGet.host || ''
    } else {
      httpConfig.path = '/'
      httpConfig.port = 80
      httpConfig.scheme = 'HTTP'
      httpConfig.host = ''
    }

    if (props.handler.tcpSocket) {
      tcpConfig.port = props.handler.tcpSocket.port || 80
      tcpConfig.host = props.handler.tcpSocket.host || ''
    } else {
      tcpConfig.port = 80
      tcpConfig.host = ''
    }

    if (props.handler.sleep) {
      sleepSeconds.value = props.handler.sleep.seconds || 5
    } else {
      sleepSeconds.value = 5
    }

    localVersion = currentPropsVersion
  }

  // 构建并发送更新
  const buildAndEmitHandler = () => {
    const handler: LifecycleHandler = { type: localType.value }

    if (localType.value === 'exec') {
      const commands = execCommandStr.value
        .split('\n')
        .map((s) => s.trim())
        .filter((s) => s)
      if (commands.length > 0) {
        handler.exec = { command: commands }
      }
    } else if (localType.value === 'httpGet') {
      handler.httpGet = {
        path: httpConfig.path || '/',
        port: httpConfig.port || 80,
        scheme: httpConfig.scheme || 'HTTP'
      }
      if (httpConfig.host) handler.httpGet.host = httpConfig.host
    } else if (localType.value === 'tcpSocket') {
      handler.tcpSocket = { port: tcpConfig.port || 80 }
      if (tcpConfig.host) handler.tcpSocket.host = tcpConfig.host
    } else if (localType.value === 'sleep') {
      handler.sleep = { seconds: sleepSeconds.value || 5 }
    }

    emit('update:handler', handler)
  }

  const onFieldChange = () => {
    if (localVersion === propsVersion) {
      buildAndEmitHandler()
    }
  }

  const onTypeChange = () => {
    // 类型切换时总是需要更新
    buildAndEmitHandler()
  }

  // 监听 props
  watch(() => props.handler, syncFromProps, { immediate: true, deep: true })
</script>

<style lang="scss" scoped>
  .lifecycle-handler-form {
    .handler-type {
      display: flex;
      align-items: center;
      gap: 12px;
      margin-bottom: 16px;
      .type-label {
        font-size: 13px;
        color: #606266;
      }

      :deep(.el-radio-group) {
        .el-radio-button__inner {
          .radio-content {
            display: inline-flex;
            align-items: center;
            gap: 4px;
          }
        }
      }
    }

    .handler-config {
      .form-row {
        display: flex;
        gap: 12px;
        flex-wrap: wrap;

        .form-item {
          flex: 1;
          min-width: 120px;
          max-width: 200px;
          label {
            display: block;
            font-size: 12px;
            color: #606266;
            margin-bottom: 6px;
          }
          :deep(.el-input),
          :deep(.el-select),
          :deep(.el-input-number) {
            width: 100%;
          }
        }
      }

      .form-item-full {
        label {
          display: block;
          font-size: 12px;
          color: #606266;
          margin-bottom: 6px;
        }
        :deep(.el-textarea) {
          width: 100%;
        }
      }
    }
  }
</style>
