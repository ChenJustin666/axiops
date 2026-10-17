<template>
  <div class="container-config-form">
    <!-- 标签导航栏 -->
    <div class="tab-nav-wrapper">
      <div class="tab-nav">
        <div
          v-for="tab in tabs"
          :key="tab.key"
          class="tab-item"
          :class="{ active: activeTab === tab.key, 'has-config': tab.hasConfig }"
          @click="activeTab = tab.key"
        >
          <component :is="tab.icon" :size="14" />
          <span>{{ tab.label }}</span>
          <span v-if="tab.hasConfig" class="config-dot"></span>
        </div>
      </div>
    </div>

    <!-- 内容区域 -->
    <div class="tab-content">
      <!-- 基础配置 -->
      <div v-show="activeTab === 'basic'" class="content-panel">
        <div class="form-row">
          <div class="form-item">
            <label>镜像拉取策略</label>
            <ElSelect
              v-model="localConfig.imagePullPolicy"
              :disabled="!editing"
              placeholder="IfNotPresent"
              @change="onFieldChange"
            >
              <ElOption label="Always（总是拉取）" value="Always" />
              <ElOption label="IfNotPresent（不存在时拉取）" value="IfNotPresent" />
              <ElOption label="Never（从不拉取）" value="Never" />
            </ElSelect>
          </div>
          <div class="form-item" style="flex: 2; max-width: none">
            <label>工作目录</label>
            <ElInput
              v-model="localConfig.workingDir"
              :disabled="!editing"
              placeholder="/app"
              clearable
              @change="onFieldChange"
            />
          </div>
        </div>

        <div class="switch-group" style="margin-top: 16px">
          <div class="switch-item">
            <ElSwitch v-model="localConfig.tty" :disabled="!editing" @change="onFieldChange" />
            <span>分配 TTY</span>
          </div>
          <div class="switch-item">
            <ElSwitch v-model="localConfig.stdin" :disabled="!editing" @change="onFieldChange" />
            <span>开启 Stdin</span>
          </div>
          <div class="switch-item">
            <ElSwitch
              v-model="localConfig.stdinOnce"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>Stdin Once</span>
          </div>
        </div>
      </div>

      <!-- 启动命令 -->
      <div v-show="activeTab === 'command'" class="content-panel">
        <div class="form-row-vertical">
          <div class="form-item-full">
            <label>Command（容器启动命令，每行一个参数）</label>
            <ElInput
              v-model="commandStr"
              :disabled="!editing"
              type="textarea"
              :rows="4"
              placeholder="例如：&#10;/bin/sh&#10;-c"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item-full">
            <label>Args（命令参数，每行一个参数）</label>
            <ElInput
              v-model="argsStr"
              :disabled="!editing"
              type="textarea"
              :rows="4"
              placeholder="例如：&#10;echo hello"
              @change="onFieldChange"
            />
          </div>
        </div>
      </div>

      <!-- 生命周期钩子 -->
      <div v-show="activeTab === 'lifecycle'" class="content-panel">
        <div class="lifecycle-section">
          <div class="lifecycle-header">
            <span class="lifecycle-title">PostStart（启动后执行）</span>
            <ElSwitch
              v-model="postStartEnabled"
              :disabled="!editing"
              size="small"
              @change="onFieldChange"
            />
          </div>
          <div v-if="postStartEnabled" class="lifecycle-body">
            <LifecycleHandlerForm
              :handler="postStartHandler"
              :editing="editing"
              @update:handler="handlePostStartUpdate"
            />
          </div>
        </div>

        <ElDivider />

        <div class="lifecycle-section">
          <div class="lifecycle-header">
            <span class="lifecycle-title">PreStop（停止前执行）</span>
            <ElSwitch
              v-model="preStopEnabled"
              :disabled="!editing"
              size="small"
              @change="onFieldChange"
            />
          </div>
          <div v-if="preStopEnabled" class="lifecycle-body">
            <LifecycleHandlerForm
              :handler="preStopHandler"
              :editing="editing"
              @update:handler="handlePreStopUpdate"
            />
          </div>
        </div>
      </div>

      <!-- 安全上下文 -->
      <div v-show="activeTab === 'security'" class="content-panel">
        <div class="form-row">
          <div class="form-item">
            <label>运行用户 UID</label>
            <ElInputNumber
              v-model="securityContext.runAsUser"
              :disabled="!editing"
              :min="0"
              controls-position="right"
              placeholder="UID"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>运行组 GID</label>
            <ElInputNumber
              v-model="securityContext.runAsGroup"
              :disabled="!editing"
              :min="0"
              controls-position="right"
              placeholder="GID"
              @change="onFieldChange"
            />
          </div>
        </div>

        <div class="switch-group" style="margin-top: 16px">
          <div class="switch-item">
            <ElSwitch
              v-model="securityContext.runAsNonRoot"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>必须非 Root</span>
          </div>
          <div class="switch-item">
            <ElSwitch
              v-model="securityContext.readOnlyRootFilesystem"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>只读根文件系统</span>
          </div>
          <div class="switch-item">
            <ElSwitch
              v-model="securityContext.privileged"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>特权模式</span>
          </div>
          <div class="switch-item">
            <ElSwitch
              v-model="securityContext.allowPrivilegeEscalation"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>允许提权</span>
          </div>
        </div>

        <!-- Capabilities -->
        <div class="sub-section">
          <span class="sub-title">Capabilities 能力配置</span>
          <div class="form-row">
            <div class="form-item" style="max-width: none; flex: 1">
              <label>添加能力 (Add)</label>
              <ElSelect
                v-model="capabilitiesAdd"
                :disabled="!editing"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="选择或输入"
                @change="onFieldChange"
              >
                <ElOption v-for="cap in commonCapabilities" :key="cap" :label="cap" :value="cap" />
              </ElSelect>
            </div>
            <div class="form-item" style="max-width: none; flex: 1">
              <label>移除能力 (Drop)</label>
              <ElSelect
                v-model="capabilitiesDrop"
                :disabled="!editing"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="选择或输入"
                @change="onFieldChange"
              >
                <ElOption label="ALL（移除全部）" value="ALL" />
                <ElOption v-for="cap in commonCapabilities" :key="cap" :label="cap" :value="cap" />
              </ElSelect>
            </div>
          </div>
        </div>

        <!-- Seccomp -->
        <div class="sub-section">
          <span class="sub-title">Seccomp 配置</span>
          <div class="form-row">
            <div class="form-item">
              <label>类型</label>
              <ElSelect
                v-model="seccompType"
                :disabled="!editing"
                placeholder="未设置"
                clearable
                @change="onFieldChange"
              >
                <ElOption label="RuntimeDefault" value="RuntimeDefault" />
                <ElOption label="Unconfined" value="Unconfined" />
                <ElOption label="Localhost" value="Localhost" />
              </ElSelect>
            </div>
            <div v-if="seccompType === 'Localhost'" class="form-item" style="flex: 2">
              <label>本地配置文件路径</label>
              <ElInput
                v-model="seccompLocalhostProfile"
                :disabled="!editing"
                placeholder="profiles/xxx.json"
                @change="onFieldChange"
              />
            </div>
          </div>
        </div>

        <!-- SELinux -->
        <div class="sub-section">
          <span class="sub-title">SELinux 选项</span>
          <div class="form-row">
            <div class="form-item">
              <label>User</label>
              <ElInput
                v-model="seLinuxOptions.user"
                :disabled="!editing"
                placeholder="User"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Role</label>
              <ElInput
                v-model="seLinuxOptions.role"
                :disabled="!editing"
                placeholder="Role"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Type</label>
              <ElInput
                v-model="seLinuxOptions.type"
                :disabled="!editing"
                placeholder="Type"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Level</label>
              <ElInput
                v-model="seLinuxOptions.level"
                :disabled="!editing"
                placeholder="Level"
                @change="onFieldChange"
              />
            </div>
          </div>
        </div>

        <!-- AppArmor -->
        <div class="sub-section">
          <span class="sub-title">AppArmor 配置</span>
          <div class="form-row">
            <div class="form-item">
              <label>类型</label>
              <ElSelect
                v-model="appArmorType"
                :disabled="!editing"
                placeholder="未设置"
                clearable
                @change="onFieldChange"
              >
                <ElOption label="RuntimeDefault" value="RuntimeDefault" />
                <ElOption label="Unconfined" value="Unconfined" />
                <ElOption label="Localhost" value="Localhost" />
              </ElSelect>
            </div>
            <div v-if="appArmorType === 'Localhost'" class="form-item" style="flex: 2">
              <label>本地配置文件名</label>
              <ElInput
                v-model="appArmorLocalhostProfile"
                :disabled="!editing"
                placeholder="profile_name"
                @change="onFieldChange"
              />
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, markRaw } from 'vue'
  import { Settings, Terminal, LifeBuoy, Shield } from 'lucide-vue-next'
  import type {
    ContainerAdvancedConfig,
    LifecycleHandler,
    ContainerSecurityContext,
    SELinuxOptions
  } from '@/api'
  import LifecycleHandlerForm from './LifecycleHandlerForm.vue'

  interface Props {
    config: ContainerAdvancedConfig
    editing: boolean
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ 'update:config': [config: ContainerAdvancedConfig] }>()

  // ==================== 标签配置 ====================
  const activeTab = ref('basic')

  const tabs = computed(() => [
    { key: 'basic', label: '基础配置', icon: markRaw(Settings), hasConfig: false },
    { key: 'command', label: '启动命令', icon: markRaw(Terminal), hasConfig: hasCommand.value },
    { key: 'lifecycle', label: '生命周期', icon: markRaw(LifeBuoy), hasConfig: hasLifecycle.value },
    {
      key: 'security',
      label: '安全上下文',
      icon: markRaw(Shield),
      hasConfig: hasSecurityConfig.value
    }
  ])

  // ==================== 同步控制 ====================
  let propsVersion = 0
  let localVersion = 0

  // ==================== 本地配置 ====================
  const localConfig = reactive({
    imagePullPolicy: 'IfNotPresent',
    workingDir: '',
    tty: false,
    stdin: false,
    stdinOnce: false
  })

  // 命令
  const commandStr = ref('')
  const argsStr = ref('')
  const hasCommand = computed(() => commandStr.value.trim() !== '' || argsStr.value.trim() !== '')

  // 生命周期
  const postStartEnabled = ref(false)
  const preStopEnabled = ref(false)
  const postStartHandler = ref<LifecycleHandler>({ type: 'exec' })
  const preStopHandler = ref<LifecycleHandler>({ type: 'exec' })
  const hasLifecycle = computed(() => postStartEnabled.value || preStopEnabled.value)

  // 安全上下文
  const securityContext = reactive({
    runAsUser: undefined as number | undefined,
    runAsGroup: undefined as number | undefined,
    runAsNonRoot: false,
    readOnlyRootFilesystem: false,
    privileged: false,
    allowPrivilegeEscalation: false
  })
  const capabilitiesAdd = ref<string[]>([])
  const capabilitiesDrop = ref<string[]>([])
  const seccompType = ref('')
  const seccompLocalhostProfile = ref('')
  const seLinuxOptions = reactive({ user: '', role: '', type: '', level: '' })
  const appArmorType = ref('')
  const appArmorLocalhostProfile = ref('')

  const hasSecurityConfig = computed(() => {
    return (
      securityContext.runAsUser !== undefined ||
      securityContext.runAsGroup !== undefined ||
      securityContext.runAsNonRoot ||
      securityContext.readOnlyRootFilesystem ||
      securityContext.privileged ||
      capabilitiesAdd.value.length > 0 ||
      capabilitiesDrop.value.length > 0 ||
      seccompType.value ||
      Object.values(seLinuxOptions).some((v) => v) ||
      appArmorType.value
    )
  })

  const commonCapabilities = [
    'NET_ADMIN',
    'NET_RAW',
    'NET_BIND_SERVICE',
    'SYS_ADMIN',
    'SYS_PTRACE',
    'SYS_TIME',
    'CHOWN',
    'DAC_OVERRIDE',
    'FOWNER',
    'KILL',
    'SETGID',
    'SETUID',
    'MKNOD'
  ]

  // ==================== 从 props 同步到本地 ====================
  const syncFromProps = () => {
    propsVersion++
    const currentPropsVersion = propsVersion

    localConfig.imagePullPolicy = props.config.imagePullPolicy || 'IfNotPresent'
    localConfig.workingDir = props.config.workingDir || ''
    localConfig.tty = props.config.tty ?? false
    localConfig.stdin = props.config.stdin ?? false
    localConfig.stdinOnce = props.config.stdinOnce ?? false

    commandStr.value = (props.config.command || []).join('\n')
    argsStr.value = (props.config.args || []).join('\n')

    if (props.config.lifecycle?.postStart) {
      postStartEnabled.value = true
      postStartHandler.value = JSON.parse(JSON.stringify(props.config.lifecycle.postStart))
    } else {
      postStartEnabled.value = false
      postStartHandler.value = { type: 'exec' }
    }

    if (props.config.lifecycle?.preStop) {
      preStopEnabled.value = true
      preStopHandler.value = JSON.parse(JSON.stringify(props.config.lifecycle.preStop))
    } else {
      preStopEnabled.value = false
      preStopHandler.value = { type: 'exec' }
    }

    const sc = props.config.securityContext || {}
    securityContext.runAsUser = sc.runAsUser
    securityContext.runAsGroup = sc.runAsGroup
    securityContext.runAsNonRoot = sc.runAsNonRoot ?? false
    securityContext.readOnlyRootFilesystem = sc.readOnlyRootFilesystem ?? false
    securityContext.privileged = sc.privileged ?? false
    securityContext.allowPrivilegeEscalation = sc.allowPrivilegeEscalation ?? false

    capabilitiesAdd.value = sc.capabilities?.add ? [...sc.capabilities.add] : []
    capabilitiesDrop.value = sc.capabilities?.drop ? [...sc.capabilities.drop] : []
    seccompType.value = sc.seccompProfile?.type || ''
    seccompLocalhostProfile.value = sc.seccompProfile?.localhostProfile || ''
    seLinuxOptions.user = sc.seLinuxOptions?.user || ''
    seLinuxOptions.role = sc.seLinuxOptions?.role || ''
    seLinuxOptions.type = sc.seLinuxOptions?.type || ''
    seLinuxOptions.level = sc.seLinuxOptions?.level || ''
    appArmorType.value = sc.appArmorProfile?.type || ''
    appArmorLocalhostProfile.value = sc.appArmorProfile?.localhostProfile || ''

    localVersion = currentPropsVersion
  }

  // ==================== 构建配置并发送更新 ====================
  const buildAndEmitConfig = () => {
    const newConfig: ContainerAdvancedConfig = {
      containerName: props.config.containerName,
      containerType: props.config.containerType,
      imagePullPolicy: localConfig.imagePullPolicy,
      workingDir: localConfig.workingDir || undefined,
      tty: localConfig.tty,
      stdin: localConfig.stdin,
      stdinOnce: localConfig.stdinOnce
    }

    const cmd = commandStr.value
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s)
    const args = argsStr.value
      .split('\n')
      .map((s) => s.trim())
      .filter((s) => s)
    if (cmd.length > 0) newConfig.command = cmd
    if (args.length > 0) newConfig.args = args

    if (postStartEnabled.value || preStopEnabled.value) {
      newConfig.lifecycle = {}
      if (postStartEnabled.value)
        newConfig.lifecycle.postStart = JSON.parse(JSON.stringify(postStartHandler.value))
      if (preStopEnabled.value)
        newConfig.lifecycle.preStop = JSON.parse(JSON.stringify(preStopHandler.value))
    }

    const sc: ContainerSecurityContext = {}
    if (securityContext.runAsUser !== undefined && securityContext.runAsUser !== null)
      sc.runAsUser = securityContext.runAsUser
    if (securityContext.runAsGroup !== undefined && securityContext.runAsGroup !== null)
      sc.runAsGroup = securityContext.runAsGroup
    if (securityContext.runAsNonRoot) sc.runAsNonRoot = true
    if (securityContext.readOnlyRootFilesystem) sc.readOnlyRootFilesystem = true
    if (securityContext.privileged) sc.privileged = true
    if (securityContext.allowPrivilegeEscalation !== undefined)
      sc.allowPrivilegeEscalation = securityContext.allowPrivilegeEscalation

    if (capabilitiesAdd.value.length > 0 || capabilitiesDrop.value.length > 0) {
      sc.capabilities = {}
      if (capabilitiesAdd.value.length > 0) sc.capabilities.add = [...capabilitiesAdd.value]
      if (capabilitiesDrop.value.length > 0) sc.capabilities.drop = [...capabilitiesDrop.value]
    }

    if (seccompType.value) {
      sc.seccompProfile = { type: seccompType.value }
      if (seccompType.value === 'Localhost' && seccompLocalhostProfile.value)
        sc.seccompProfile.localhostProfile = seccompLocalhostProfile.value
    }

    const selinux: SELinuxOptions = {}
    if (seLinuxOptions.user) selinux.user = seLinuxOptions.user
    if (seLinuxOptions.role) selinux.role = seLinuxOptions.role
    if (seLinuxOptions.type) selinux.type = seLinuxOptions.type
    if (seLinuxOptions.level) selinux.level = seLinuxOptions.level
    if (Object.keys(selinux).length > 0) sc.seLinuxOptions = selinux

    if (appArmorType.value) {
      sc.appArmorProfile = { type: appArmorType.value }
      if (appArmorType.value === 'Localhost' && appArmorLocalhostProfile.value)
        sc.appArmorProfile.localhostProfile = appArmorLocalhostProfile.value
    }

    if (Object.keys(sc).length > 0) newConfig.securityContext = sc

    emit('update:config', newConfig)
  }

  const onFieldChange = () => {
    if (localVersion === propsVersion) {
      buildAndEmitConfig()
    }
  }

  const handlePostStartUpdate = (handler: LifecycleHandler) => {
    postStartHandler.value = handler
    onFieldChange()
  }

  const handlePreStopUpdate = (handler: LifecycleHandler) => {
    preStopHandler.value = handler
    onFieldChange()
  }

  watch(() => props.config, syncFromProps, { immediate: true, deep: true })
</script>

<style lang="scss" scoped>
  .container-config-form {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #fff;

    .tab-nav-wrapper {
      padding: 16px 20px 0;
      background: #fff;
      flex-shrink: 0;
    }

    .tab-nav {
      display: flex;
      gap: 4px;
      overflow-x: auto;
      padding-bottom: 0;
      border-bottom: 1px solid #e4e7ed;

      &::-webkit-scrollbar {
        height: 0;
      }

      .tab-item {
        display: inline-flex;
        align-items: center;
        gap: 6px;
        padding: 10px 20px;
        font-size: 13px;
        color: #606266;
        cursor: pointer;
        transition: all 0.2s;
        white-space: nowrap;
        position: relative;
        border-bottom: 2px solid transparent;
        margin-bottom: -1px;

        &:hover {
          color: #409eff;
        }

        &.active {
          color: #409eff;
          font-weight: 500;
          border-bottom-color: #409eff;
        }

        &.has-config .config-dot {
          width: 6px;
          height: 6px;
          background: #67c23a;
          border-radius: 50%;
          margin-left: 2px;
        }
      }
    }

    .tab-content {
      flex: 1;
      overflow-y: auto;
      background: #fff;

      .content-panel {
        padding: 20px;
        min-height: 800px;
      }
    }

    .form-row {
      display: flex;
      gap: 16px;
      flex-wrap: wrap;

      .form-item {
        flex: 1;
        min-width: 180px;
        max-width: 280px;

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

    .form-row-vertical {
      display: flex;
      flex-direction: column;
      gap: 16px;

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

    .switch-group {
      display: flex;
      flex-wrap: wrap;
      gap: 24px;

      .switch-item {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 13px;
        color: #606266;
      }
    }

    .lifecycle-section {
      .lifecycle-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 12px;

        .lifecycle-title {
          font-size: 14px;
          font-weight: 500;
          color: #303133;
        }
      }

      .lifecycle-body {
        padding: 16px;
        background: #fafbfc;
        border-radius: 6px;
        border: 1px solid #ebeef5;
      }
    }

    :deep(.el-divider) {
      margin: 20px 0;
    }

    .sub-section {
      margin-top: 24px;
      padding-top: 20px;
      border-top: 1px solid #ebeef5;

      .sub-title {
        display: block;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        margin-bottom: 12px;
      }
    }
  }
</style>
