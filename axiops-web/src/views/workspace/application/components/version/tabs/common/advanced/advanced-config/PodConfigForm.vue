<template>
  <div class="pod-config-form">
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
      <!-- 基础设置 -->
      <div v-show="activeTab === 'basic'" class="content-panel">
        <div class="form-row">
          <div class="form-item">
            <label>优雅终止时间(秒)</label>
            <ElInputNumber
              v-model="localConfig.terminationGracePeriodSeconds"
              :disabled="!editing"
              :min="0"
              :max="3600"
              controls-position="right"
              placeholder="30"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>重启策略</label>
            <ElSelect
              v-model="localConfig.restartPolicy"
              :disabled="!editing"
              placeholder="Always"
              @change="onFieldChange"
            >
              <ElOption label="Always（总是重启）" value="Always" />
              <ElOption label="OnFailure（失败时重启）" value="OnFailure" />
              <ElOption label="Never（从不重启）" value="Never" />
            </ElSelect>
          </div>
          <div class="form-item">
            <label>调度器名称</label>
            <ElInput
              v-model="localConfig.schedulerName"
              :disabled="!editing"
              placeholder="default-scheduler"
              clearable
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>活跃截止时间(秒)</label>
            <ElInputNumber
              v-model="localConfig.activeDeadlineSeconds"
              :disabled="!editing"
              :min="0"
              controls-position="right"
              placeholder="不限制"
              @change="onFieldChange"
            />
          </div>
        </div>
      </div>

      <!-- DNS 配置 -->
      <div v-show="activeTab === 'dns'" class="content-panel">
        <div class="form-row">
          <div class="form-item" style="flex: 2; max-width: 320px">
            <label>DNS 策略</label>
            <ElSelect
              v-model="localConfig.dnsPolicy"
              :disabled="!editing"
              placeholder="ClusterFirst"
              @change="onFieldChange"
            >
              <ElOption label="ClusterFirst（集群优先）" value="ClusterFirst" />
              <ElOption label="ClusterFirstWithHostNet" value="ClusterFirstWithHostNet" />
              <ElOption label="Default（默认）" value="Default" />
              <ElOption label="None（自定义）" value="None" />
            </ElSelect>
          </div>
        </div>
        <div class="form-row" style="margin-top: 16px">
          <div class="form-item" style="flex: 1; max-width: none">
            <label>DNS 服务器</label>
            <ElSelect
              v-model="dnsNameservers"
              :disabled="!editing"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="输入 IP 后回车添加"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item" style="flex: 1; max-width: none">
            <label>搜索域</label>
            <ElSelect
              v-model="dnsSearches"
              :disabled="!editing"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="输入域名后回车添加"
              @change="onFieldChange"
            />
          </div>
        </div>
      </div>

      <!-- 网络配置 -->
      <div v-show="activeTab === 'network'" class="content-panel">
        <div class="switch-group">
          <div class="switch-item">
            <ElSwitch
              v-model="localConfig.hostNetwork"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>主机网络</span>
          </div>
          <div class="switch-item">
            <ElSwitch v-model="localConfig.hostPID" :disabled="!editing" @change="onFieldChange" />
            <span>主机 PID</span>
          </div>
          <div class="switch-item">
            <ElSwitch v-model="localConfig.hostIPC" :disabled="!editing" @change="onFieldChange" />
            <span>主机 IPC</span>
          </div>
          <div class="switch-item">
            <ElSwitch
              v-model="localConfig.shareProcessNamespace"
              :disabled="!editing"
              @change="onFieldChange"
            />
            <span>共享进程空间</span>
          </div>
        </div>
        <div class="form-row" style="margin-top: 16px">
          <div class="form-item">
            <label>主机名</label>
            <ElInput
              v-model="localConfig.hostname"
              :disabled="!editing"
              placeholder="Pod 主机名"
              clearable
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>子域名</label>
            <ElInput
              v-model="localConfig.subdomain"
              :disabled="!editing"
              placeholder="子域名"
              clearable
              @change="onFieldChange"
            />
          </div>
        </div>
      </div>

      <!-- 静态 Host 配置 -->
      <div v-show="activeTab === 'hosts'" class="content-panel">
        <div class="section-header">
          <span class="section-title">静态 Host 配置</span>
          <ElButton v-if="editing" type="primary" size="small" @click="addHostAlias">
            <Plus :size="14" />
            添加
          </ElButton>
        </div>
        <div v-if="hostAliases.length === 0" class="empty-tip">
          <span>未配置静态 Host</span>
          <template v-if="editing">
            <ElButton type="primary" link size="small" @click="addHostAlias">点击添加</ElButton>
          </template>
          <template v-else>
            <span class="edit-hint">（编辑模式下可添加）</span>
          </template>
        </div>
        <div v-else class="host-aliases-table">
          <div class="table-header">
            <span class="col-ip">IP 地址</span>
            <span class="col-hosts">主机名（多个用空格或回车分隔）</span>
            <span class="col-action">操作</span>
          </div>
          <div v-for="(alias, index) in hostAliases" :key="index" class="table-row">
            <div class="col-ip">
              <ElInput
                v-model="alias.ip"
                :disabled="!editing"
                placeholder="192.168.1.1"
                @change="onFieldChange"
              />
            </div>
            <div class="col-hosts">
              <ElSelect
                v-model="alias.hostnames"
                :disabled="!editing"
                multiple
                filterable
                allow-create
                default-first-option
                placeholder="输入主机名后回车"
                @change="onFieldChange"
              />
            </div>
            <div class="col-action">
              <ElButton
                v-if="editing"
                type="danger"
                link
                size="small"
                @click="removeHostAlias(index)"
              >
                <Trash2 :size="14" />
              </ElButton>
              <span v-else class="action-placeholder">-</span>
            </div>
          </div>
        </div>
      </div>

      <!-- 服务账户 -->
      <div v-show="activeTab === 'service'" class="content-panel">
        <div class="form-row">
          <div class="form-item">
            <label>服务账户名称</label>
            <ElInput
              v-model="localConfig.serviceAccountName"
              :disabled="!editing"
              placeholder="default"
              clearable
              @change="onFieldChange"
            />
          </div>
          <div class="form-item switch-inline">
            <label>自动挂载 Token</label>
            <ElSwitch
              v-model="localConfig.automountServiceAccountToken"
              :disabled="!editing"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item" style="flex: 2; max-width: none">
            <label>镜像拉取密钥</label>
            <ElSelect
              v-model="imagePullSecrets"
              :disabled="!editing"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="选择或输入 Secret 名称"
              @change="onFieldChange"
            />
          </div>
        </div>
      </div>

      <!-- 优先级配置 -->
      <div v-show="activeTab === 'priority'" class="content-panel">
        <div class="form-row">
          <div class="form-item">
            <label>优先级类名</label>
            <ElInput
              v-model="localConfig.priorityClassName"
              :disabled="!editing"
              placeholder="优先级类名"
              clearable
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>优先级数值</label>
            <ElInputNumber
              v-model="localConfig.priority"
              :disabled="!editing"
              controls-position="right"
              placeholder="数值"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item">
            <label>抢占策略</label>
            <ElSelect
              v-model="localConfig.preemptionPolicy"
              :disabled="!editing"
              placeholder="选择策略"
              clearable
              @change="onFieldChange"
            >
              <ElOption label="PreemptLowerPriority" value="PreemptLowerPriority" />
              <ElOption label="Never（从不抢占）" value="Never" />
            </ElSelect>
          </div>
          <div class="form-item">
            <label>运行时类名</label>
            <ElInput
              v-model="localConfig.runtimeClassName"
              :disabled="!editing"
              placeholder="运行时类名"
              clearable
              @change="onFieldChange"
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
          <div class="form-item">
            <label>文件系统组 ID</label>
            <ElInputNumber
              v-model="securityContext.fsGroup"
              :disabled="!editing"
              :min="0"
              controls-position="right"
              placeholder="fsGroup"
              @change="onFieldChange"
            />
          </div>
          <div class="form-item switch-inline">
            <label>必须非 Root</label>
            <ElSwitch
              v-model="securityContext.runAsNonRoot"
              :disabled="!editing"
              @change="onFieldChange"
            />
          </div>
        </div>

        <div class="form-row" style="margin-top: 16px">
          <div class="form-item">
            <label>fsGroup 变更策略</label>
            <ElSelect
              v-model="securityContext.fsGroupChangePolicy"
              :disabled="!editing"
              placeholder="选择策略"
              clearable
              @change="onFieldChange"
            >
              <ElOption label="Always（总是修改）" value="Always" />
              <ElOption label="OnRootMismatch（不匹配时修改）" value="OnRootMismatch" />
            </ElSelect>
          </div>
          <div class="form-item" style="flex: 2; max-width: none">
            <label>补充组 (Supplemental Groups)</label>
            <ElSelect
              v-model="supplementalGroups"
              :disabled="!editing"
              multiple
              filterable
              allow-create
              default-first-option
              placeholder="输入组 ID 后回车"
              @change="onFieldChange"
            />
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
                placeholder="SELinux User"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Role</label>
              <ElInput
                v-model="seLinuxOptions.role"
                :disabled="!editing"
                placeholder="SELinux Role"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Type</label>
              <ElInput
                v-model="seLinuxOptions.type"
                :disabled="!editing"
                placeholder="SELinux Type"
                @change="onFieldChange"
              />
            </div>
            <div class="form-item">
              <label>Level</label>
              <ElInput
                v-model="seLinuxOptions.level"
                :disabled="!editing"
                placeholder="SELinux Level"
                @change="onFieldChange"
              />
            </div>
          </div>
        </div>

        <!-- Sysctls -->
        <div class="sub-section">
          <div class="sub-title-row">
            <span class="sub-title">Sysctls 配置</span>
            <ElButton v-if="editing" type="primary" link size="small" @click="addSysctl">
              <Plus :size="12" />
              添加
            </ElButton>
          </div>
          <div v-if="sysctls.length === 0" class="empty-tip small">
            <span>未配置 Sysctls</span>
            <template v-if="editing">
              <ElButton type="primary" link size="small" @click="addSysctl">点击添加</ElButton>
            </template>
            <template v-else>
              <span class="edit-hint">（编辑模式下可添加）</span>
            </template>
          </div>
          <div v-else class="sysctl-list">
            <div v-for="(item, index) in sysctls" :key="index" class="sysctl-item">
              <ElInput
                v-model="item.name"
                :disabled="!editing"
                placeholder="参数名"
                style="width: 240px"
                @change="onFieldChange"
              />
              <ElInput
                v-model="item.value"
                :disabled="!editing"
                placeholder="参数值"
                style="flex: 1"
                @change="onFieldChange"
              />
              <ElButton v-if="editing" type="danger" link size="small" @click="removeSysctl(index)">
                <Trash2 :size="14" />
              </ElButton>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, markRaw } from 'vue'
  import {
    Settings,
    Globe,
    Network,
    FileText,
    User,
    Layers,
    Shield,
    Plus,
    Trash2
  } from 'lucide-vue-next'
  import type {
    PodAdvancedConfig,
    HostAlias,
    Sysctl,
    SELinuxOptions,
    PodSecurityContext
  } from '@/api'

  interface Props {
    config: PodAdvancedConfig
    editing: boolean
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{
    'update:config': [config: PodAdvancedConfig]
  }>()

  // ==================== 标签配置 ====================
  const activeTab = ref('basic')

  const tabs = computed(() => [
    { key: 'basic', label: '基础设置', icon: markRaw(Settings), hasConfig: false },
    { key: 'dns', label: 'DNS 设置', icon: markRaw(Globe), hasConfig: hasDnsConfig.value },
    {
      key: 'network',
      label: '网络配置',
      icon: markRaw(Network),
      hasConfig: hasNetworkConfig.value
    },
    {
      key: 'hosts',
      label: '静态 Host',
      icon: markRaw(FileText),
      hasConfig: hostAliases.value.length > 0
    },
    { key: 'service', label: '服务账户', icon: markRaw(User), hasConfig: hasServiceConfig.value },
    {
      key: 'priority',
      label: '优先级配置',
      icon: markRaw(Layers),
      hasConfig: hasPriorityConfig.value
    },
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
    terminationGracePeriodSeconds: 30 as number | undefined,
    restartPolicy: 'Always',
    schedulerName: '',
    activeDeadlineSeconds: undefined as number | undefined,
    dnsPolicy: 'ClusterFirst',
    hostNetwork: false,
    hostPID: false,
    hostIPC: false,
    shareProcessNamespace: false,
    hostname: '',
    subdomain: '',
    serviceAccountName: '',
    automountServiceAccountToken: true,
    priorityClassName: '',
    priority: undefined as number | undefined,
    preemptionPolicy: '',
    runtimeClassName: ''
  })

  // DNS 配置
  const dnsNameservers = ref<string[]>([])
  const dnsSearches = ref<string[]>([])
  const hasDnsConfig = computed(
    () =>
      dnsNameservers.value.length > 0 ||
      dnsSearches.value.length > 0 ||
      localConfig.dnsPolicy !== 'ClusterFirst'
  )

  // 网络配置标记
  const hasNetworkConfig = computed(
    () =>
      localConfig.hostNetwork ||
      localConfig.hostPID ||
      localConfig.hostIPC ||
      localConfig.shareProcessNamespace ||
      !!localConfig.hostname ||
      !!localConfig.subdomain
  )

  // 服务账户配置标记
  const hasServiceConfig = computed(
    () =>
      !!localConfig.serviceAccountName ||
      !localConfig.automountServiceAccountToken ||
      imagePullSecrets.value.length > 0
  )

  // 优先级配置标记
  const hasPriorityConfig = computed(
    () =>
      !!localConfig.priorityClassName ||
      localConfig.priority !== undefined ||
      !!localConfig.preemptionPolicy ||
      !!localConfig.runtimeClassName
  )

  // Host Aliases
  const hostAliases = ref<HostAlias[]>([])

  // 镜像拉取密钥
  const imagePullSecrets = ref<string[]>([])

  // 安全上下文
  const securityContext = reactive({
    runAsUser: undefined as number | undefined,
    runAsGroup: undefined as number | undefined,
    runAsNonRoot: false,
    fsGroup: undefined as number | undefined,
    fsGroupChangePolicy: ''
  })
  const supplementalGroups = ref<string[]>([])
  const seccompType = ref('')
  const seccompLocalhostProfile = ref('')
  const seLinuxOptions = reactive({ user: '', role: '', type: '', level: '' })
  const sysctls = ref<Sysctl[]>([])

  // 判断是否有安全配置
  const hasSecurityConfig = computed(() => {
    return (
      securityContext.runAsUser !== undefined ||
      securityContext.runAsGroup !== undefined ||
      securityContext.fsGroup !== undefined ||
      securityContext.runAsNonRoot ||
      seccompType.value ||
      Object.values(seLinuxOptions).some((v) => v) ||
      sysctls.value.length > 0
    )
  })

  // ==================== 从 props 同步到本地 ====================
  const syncFromProps = () => {
    propsVersion++
    const currentPropsVersion = propsVersion

    // 基础配置
    localConfig.terminationGracePeriodSeconds = props.config.terminationGracePeriodSeconds ?? 30
    localConfig.restartPolicy = props.config.restartPolicy || 'Always'
    localConfig.schedulerName = props.config.schedulerName || ''
    localConfig.activeDeadlineSeconds = props.config.activeDeadlineSeconds
    localConfig.dnsPolicy = props.config.dnsPolicy || 'ClusterFirst'
    localConfig.hostNetwork = props.config.hostNetwork ?? false
    localConfig.hostPID = props.config.hostPID ?? false
    localConfig.hostIPC = props.config.hostIPC ?? false
    localConfig.shareProcessNamespace = props.config.shareProcessNamespace ?? false
    localConfig.hostname = props.config.hostname || ''
    localConfig.subdomain = props.config.subdomain || ''
    localConfig.serviceAccountName = props.config.serviceAccountName || ''
    localConfig.automountServiceAccountToken = props.config.automountServiceAccountToken ?? true
    localConfig.priorityClassName = props.config.priorityClassName || ''
    localConfig.priority = props.config.priority
    localConfig.preemptionPolicy = props.config.preemptionPolicy || ''
    localConfig.runtimeClassName = props.config.runtimeClassName || ''

    // DNS
    dnsNameservers.value = props.config.dnsConfig?.nameservers
      ? [...props.config.dnsConfig.nameservers]
      : []
    dnsSearches.value = props.config.dnsConfig?.searches ? [...props.config.dnsConfig.searches] : []

    // Host Aliases
    hostAliases.value = props.config.hostAliases
      ? props.config.hostAliases.map((a) => ({ ip: a.ip, hostnames: [...a.hostnames] }))
      : []

    // 镜像拉取密钥
    imagePullSecrets.value = props.config.imagePullSecrets ? [...props.config.imagePullSecrets] : []

    // 安全上下文
    const sc = props.config.securityContext || {}
    securityContext.runAsUser = sc.runAsUser
    securityContext.runAsGroup = sc.runAsGroup
    securityContext.runAsNonRoot = sc.runAsNonRoot ?? false
    securityContext.fsGroup = sc.fsGroup
    securityContext.fsGroupChangePolicy = sc.fsGroupChangePolicy || ''
    supplementalGroups.value = sc.supplementalGroups
      ? sc.supplementalGroups.map((g) => String(g))
      : []
    seccompType.value = sc.seccompProfile?.type || ''
    seccompLocalhostProfile.value = sc.seccompProfile?.localhostProfile || ''
    seLinuxOptions.user = sc.seLinuxOptions?.user || ''
    seLinuxOptions.role = sc.seLinuxOptions?.role || ''
    seLinuxOptions.type = sc.seLinuxOptions?.type || ''
    seLinuxOptions.level = sc.seLinuxOptions?.level || ''
    sysctls.value = sc.sysctls ? sc.sysctls.map((s) => ({ name: s.name, value: s.value })) : []

    localVersion = currentPropsVersion
  }

  // ==================== 构建配置并发送更新 ====================
  const buildAndEmitConfig = () => {
    const newConfig: PodAdvancedConfig = {
      terminationGracePeriodSeconds: localConfig.terminationGracePeriodSeconds,
      restartPolicy: localConfig.restartPolicy,
      schedulerName: localConfig.schedulerName || undefined,
      activeDeadlineSeconds: localConfig.activeDeadlineSeconds,
      dnsPolicy: localConfig.dnsPolicy,
      hostNetwork: localConfig.hostNetwork,
      hostPID: localConfig.hostPID,
      hostIPC: localConfig.hostIPC,
      shareProcessNamespace: localConfig.shareProcessNamespace,
      hostname: localConfig.hostname || undefined,
      subdomain: localConfig.subdomain || undefined,
      serviceAccountName: localConfig.serviceAccountName || undefined,
      automountServiceAccountToken: localConfig.automountServiceAccountToken,
      priorityClassName: localConfig.priorityClassName || undefined,
      priority: localConfig.priority,
      preemptionPolicy: localConfig.preemptionPolicy || undefined,
      runtimeClassName: localConfig.runtimeClassName || undefined
    }

    // DNS
    if (dnsNameservers.value.length || dnsSearches.value.length) {
      newConfig.dnsConfig = {}
      if (dnsNameservers.value.length) newConfig.dnsConfig.nameservers = [...dnsNameservers.value]
      if (dnsSearches.value.length) newConfig.dnsConfig.searches = [...dnsSearches.value]
    }

    // Host Aliases
    if (hostAliases.value.length > 0) {
      newConfig.hostAliases = hostAliases.value.map((a) => ({
        ip: a.ip,
        hostnames: [...a.hostnames]
      }))
    }

    // 镜像拉取密钥
    if (imagePullSecrets.value.length > 0) {
      newConfig.imagePullSecrets = [...imagePullSecrets.value]
    }

    // 安全上下文
    const sc: PodSecurityContext = {}
    if (securityContext.runAsUser !== undefined && securityContext.runAsUser !== null)
      sc.runAsUser = securityContext.runAsUser
    if (securityContext.runAsGroup !== undefined && securityContext.runAsGroup !== null)
      sc.runAsGroup = securityContext.runAsGroup
    if (securityContext.runAsNonRoot) sc.runAsNonRoot = true
    if (securityContext.fsGroup !== undefined && securityContext.fsGroup !== null)
      sc.fsGroup = securityContext.fsGroup
    if (securityContext.fsGroupChangePolicy)
      sc.fsGroupChangePolicy = securityContext.fsGroupChangePolicy
    if (supplementalGroups.value.length > 0) {
      sc.supplementalGroups = supplementalGroups.value
        .map((g) => parseInt(g, 10))
        .filter((g) => !isNaN(g))
    }
    if (seccompType.value) {
      sc.seccompProfile = { type: seccompType.value }
      if (seccompType.value === 'Localhost' && seccompLocalhostProfile.value) {
        sc.seccompProfile.localhostProfile = seccompLocalhostProfile.value
      }
    }
    const selinux: SELinuxOptions = {}
    if (seLinuxOptions.user) selinux.user = seLinuxOptions.user
    if (seLinuxOptions.role) selinux.role = seLinuxOptions.role
    if (seLinuxOptions.type) selinux.type = seLinuxOptions.type
    if (seLinuxOptions.level) selinux.level = seLinuxOptions.level
    if (Object.keys(selinux).length > 0) sc.seLinuxOptions = selinux
    if (sysctls.value.length > 0) {
      sc.sysctls = sysctls.value.map((s) => ({ name: s.name, value: s.value }))
    }
    if (Object.keys(sc).length > 0) newConfig.securityContext = sc

    emit('update:config', newConfig)
  }

  // ==================== 字段变更处理 ====================
  const onFieldChange = () => {
    if (localVersion === propsVersion) {
      buildAndEmitConfig()
    }
  }

  // ==================== 操作方法 ====================
  const addHostAlias = () => {
    hostAliases.value.push({ ip: '', hostnames: [] })
  }

  const removeHostAlias = (index: number) => {
    hostAliases.value.splice(index, 1)
    onFieldChange()
  }

  const addSysctl = () => {
    sysctls.value.push({ name: '', value: '' })
  }

  const removeSysctl = (index: number) => {
    sysctls.value.splice(index, 1)
    onFieldChange()
  }

  // ==================== 监听 props 变化 ====================
  watch(() => props.config, syncFromProps, { immediate: true, deep: true })
</script>

<style lang="scss" scoped>
  .pod-config-form {
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

    .section-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 16px;

      .section-title {
        font-size: 14px;
        font-weight: 500;
        color: #303133;
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

        &.switch-inline {
          display: flex;
          align-items: center;
          gap: 8px;
          min-width: 160px;
          max-width: 200px;

          label {
            margin-bottom: 0;
            white-space: nowrap;
          }
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

    .host-aliases-table {
      .table-header {
        display: flex;
        gap: 12px;
        padding: 10px 0;
        border-bottom: 1px solid #ebeef5;
        font-size: 12px;
        color: #909399;
      }

      .table-row {
        display: flex;
        gap: 12px;
        padding: 10px 0;
        align-items: center;

        &:not(:last-child) {
          border-bottom: 1px solid #f5f7fa;
        }
      }

      .col-ip {
        width: 180px;
        flex-shrink: 0;
      }

      .col-hosts {
        flex: 1;
        min-width: 200px;
      }

      .col-action {
        width: 50px;
        flex-shrink: 0;
        text-align: center;

        .action-placeholder {
          color: #c0c4cc;
        }
      }
    }

    .empty-tip {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 24px;
      color: #909399;
      font-size: 13px;
      background: #fafbfc;
      border-radius: 6px;
      border: 1px dashed #e4e7ed;

      .edit-hint {
        color: #c0c4cc;
        font-size: 12px;
      }

      &.small {
        padding: 12px;
        font-size: 12px;
      }
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

      .sub-title-row {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 12px;

        .sub-title {
          margin-bottom: 0;
        }
      }
    }

    .sysctl-list {
      display: flex;
      flex-direction: column;
      gap: 10px;

      .sysctl-item {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 10px 12px;
        background: #fafbfc;
        border-radius: 6px;
        border: 1px solid #ebeef5;
      }
    }
  }
</style>
