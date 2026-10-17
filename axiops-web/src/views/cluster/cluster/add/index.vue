<script setup lang="ts">
  import { ref, reactive, computed, onMounted, onActivated } from 'vue'
  import { useRouter, onBeforeRouteLeave } from 'vue-router'
  import { ElMessage, ElNotification } from 'element-plus'
  import {
    ArrowLeft,
    ArrowRight,
    Server,
    Shield,
    Info,
    CheckCircle,
    CloudUpload,
    AlertCircle,
    Sparkles,
    Globe,
    MapPin,
    Building2,
    Settings,
    Layers,
    Network,
    DollarSign,
    Cpu,
    MemoryStick,
    HardDrive,
    Box,
    Search,
    Monitor,
    Activity,
    Lock,
    Unlock,
    ShieldCheck,
    ShieldOff,
    Tag
  } from 'lucide-vue-next'
  import { addClusterApi, testClusterConnectivityApi } from '@/api/manager/cluster'
  import type { AddClusterRequest, TestClusterConnectivityRequest } from '@/api/manager/cluster'
  import { searchBillingPriceConfigApi, type BillingPriceConfig } from '@/api/manager/billing'
  import {
    environmentOptions,
    clusterTypeOptions,
    providerOptions,
    authTypeOptions
  } from '../constants'

  const router = useRouter()

  // 步骤控制
  const currentStep = ref(0)
  const clusterSource = ref<'existing' | 'new' | ''>('')

  // 表单引用
  const basicFormRef = ref()
  const authFormRef = ref()
  const prometheusFormRef = ref()

  // 加载状态
  const submitLoading = ref(false)
  const testLoading = ref(false)
  const testPrometheusLoading = ref(false)
  const backLoading = ref(false)
  const nextLoading = ref(false)
  const prevLoading = ref(false)
  const priceConfigLoading = ref(false)

  // 费用配置相关
  const priceConfigs = ref<BillingPriceConfig[]>([])
  const selectedPriceConfigId = ref<number | null>(null)
  const billingStartTime = ref<number | null>(null)
  const priceSearchKeyword = ref('')

  // Prometheus 配置相关
  const enablePrometheus = ref(false)
  const skipPrometheusWarningConfirmed = ref(false)

  // 管控配置相关
  const enableAccessControl = ref(false)
  const accessControlConfirmed = ref(false)

  // 步骤配置
  const steps = computed(() => {
    const baseSteps = [{ title: '选择集群类型', description: '选择添加现有集群或创建新集群' }]

    if (clusterSource.value === 'existing') {
      return [
        ...baseSteps,
        { title: '基础信息', description: '配置集群基本信息' },
        { title: '认证配置', description: '设置集群访问认证' },
        { title: '费用配置', description: '绑定计费价格配置' },
        { title: '监控配置', description: '配置 Prometheus 监控（可选）' },
        { title: '管控配置', description: '配置集群资源管控策略' },
        { title: '完成', description: '确认并创建集群' }
      ]
    }

    return baseSteps
  })

  // 表单数据
  const formData = reactive<AddClusterRequest>({
    name: '',
    avatar: '',
    description: '',
    clusterType: 'standard',
    environment: 'development',
    region: '',
    zone: '',
    datacenter: '',
    provider: 'self-hosted',
    isManaged: 0,
    nodeLb: '',
    masterLb: '',
    ingressDomain: '',
    authType: 'kubeconfig',
    apiServerHost: '',
    kubeFile: '',
    token: '',
    caCert: '',
    caFile: '',
    clientCert: '',
    certFile: '',
    clientKey: '',
    keyFile: '',
    insecureSkipVerify: 0,
    // 管控配置
    allowCreate: false,
    // 费用配置
    priceConfigId: 0,
    billingStartTime: undefined,
    // Prometheus 配置
    enablePrometheus: 0,
    prometheusUrl: '',
    prometheusPort: 9090,
    prometheusProtocol: 'http',
    prometheusAuthEnabled: 0,
    prometheusAuthType: 'none',
    prometheusUsername: '',
    prometheusPassword: '',
    prometheusToken: '',
    prometheusAccessKey: '',
    prometheusAccessSecret: '',
    prometheusTlsEnabled: 0,
    prometheusInsecureSkipVerify: 0,
    prometheusCaCert: '',
    prometheusClientCert: '',
    prometheusClientKey: ''
  })

  // Prometheus 认证类型选项
  const prometheusAuthTypeOptions = [
    { value: 'none', label: '无认证', icon: Unlock },
    { value: 'basic', label: 'Basic 认证', icon: Lock },
    { value: 'token', label: 'Token 认证', icon: Shield },
    { value: 'apikey', label: 'API Key', icon: Settings }
  ]

  // 过滤后的价格配置列表
  const filteredPriceConfigs = computed(() => {
    if (!priceSearchKeyword.value.trim()) {
      return priceConfigs.value
    }
    const keyword = priceSearchKeyword.value.toLowerCase().trim()
    return priceConfigs.value.filter(
      (config) =>
        config.configName.toLowerCase().includes(keyword) ||
        (config.description && config.description.toLowerCase().includes(keyword))
    )
  })

  // 选中的价格配置详情
  const selectedPriceConfig = computed(() => {
    if (!selectedPriceConfigId.value) return null
    return priceConfigs.value.find((c) => c.id === selectedPriceConfigId.value) || null
  })

  // IP/域名验证函数
  const validateAddressList = (rule: any, value: string, callback: any) => {
    if (!value || value.trim() === '') {
      callback()
      return
    }
    const ipv4Regex =
      /^((25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
    const ipv6Regex =
      /^(([0-9a-fA-F]{1,4}:){7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|::)$/
    const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/

    const isValidAddress = (addr: string): boolean => {
      return ipv4Regex.test(addr) || ipv6Regex.test(addr) || domainRegex.test(addr)
    }

    const addresses = value
      .split(',')
      .map((addr) => addr.trim())
      .filter((addr) => addr)
    if (addresses.length === 0) {
      callback()
      return
    }

    const invalidAddresses = addresses.filter((addr) => !isValidAddress(addr))
    if (invalidAddresses.length > 0) {
      callback(
        new Error(
          `无效的地址格式: ${invalidAddresses.slice(0, 3).join(', ')}${invalidAddresses.length > 3 ? ' 等' : ''}`
        )
      )
    } else {
      callback()
    }
  }

  // 域名验证函数
  const validateDomainList = (rule: any, value: string, callback: any) => {
    if (!value || value.trim() === '') {
      callback()
      return
    }
    const domainRegex = /^([a-zA-Z0-9]([a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?\.)+[a-zA-Z]{2,}$/
    const domains = value
      .split(',')
      .map((d) => d.trim())
      .filter((d) => d)
    if (domains.length === 0) {
      callback()
      return
    }
    const invalidDomains = domains.filter((d) => !domainRegex.test(d))
    if (invalidDomains.length > 0) {
      callback(
        new Error(
          `无效的域名格式: ${invalidDomains.slice(0, 3).join(', ')}${invalidDomains.length > 3 ? ' 等' : ''}`
        )
      )
    } else {
      callback()
    }
  }

  // 基础信息表单验证规则
  const basicRules = {
    name: [
      { required: true, message: '请输入集群名称', trigger: 'blur' },
      { min: 2, max: 100, message: '长度在 2 到 100 个字符', trigger: 'blur' }
    ],
    clusterType: [{ required: true, message: '请选择集群类型', trigger: 'change' }],
    environment: [{ required: true, message: '请选择环境类型', trigger: 'change' }],
    provider: [{ required: true, message: '请选择云服务商', trigger: 'change' }],
    nodeLb: [{ validator: validateAddressList, trigger: 'blur' }],
    masterLb: [{ validator: validateAddressList, trigger: 'blur' }],
    ingressDomain: [{ validator: validateDomainList, trigger: 'blur' }]
  }

  // 认证表单验证规则
  const authRules = {
    authType: [{ required: true, message: '请选择认证类型', trigger: 'change' }],
    apiServerHost: [
      {
        validator: (rule: any, value: string, callback: any) => {
          if (formData.authType === 'kubeconfig' || formData.authType === 'incluster') {
            callback()
          } else if (!value) {
            callback(new Error('请输入 API Server 地址'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    kubeFile: [
      {
        validator: (rule: any, value: string, callback: any) => {
          if (formData.authType !== 'kubeconfig') {
            callback()
          } else if (!value) {
            callback(new Error('请输入 KubeConfig 内容'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    token: [
      {
        validator: (rule: any, value: string, callback: any) => {
          if (formData.authType !== 'token') {
            callback()
          } else if (!value) {
            callback(new Error('请输入 Token'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ]
  }

  // Prometheus 表单验证规则
  const prometheusRules = {
    prometheusUrl: [
      {
        validator: (rule: any, value: string, callback: any) => {
          if (!enablePrometheus.value) {
            callback()
          } else if (!value) {
            callback(new Error('请输入 Prometheus 地址'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    prometheusPort: [
      {
        validator: (rule: any, value: number, callback: any) => {
          if (!enablePrometheus.value) {
            callback()
          } else if (!value || value < 1 || value > 65535) {
            callback(new Error('请输入有效的端口号 (1-65535)'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ]
  }

  // 加载价格配置列表
  const loadPriceConfigs = async () => {
    priceConfigLoading.value = true
    try {
      const res = await searchBillingPriceConfigApi()
      priceConfigs.value = res.data || []
    } finally {
      priceConfigLoading.value = false
    }
  }

  // 选择价格配置
  const selectPriceConfig = (configId: number) => {
    selectedPriceConfigId.value = configId
  }

  // 启用 Prometheus 配置
  const handleEnablePrometheus = () => {
    enablePrometheus.value = true
    skipPrometheusWarningConfirmed.value = false
  }

  // 跳过 Prometheus 配置
  const handleSkipPrometheus = () => {
    enablePrometheus.value = false
  }

  // Prometheus 认证开关变化
  const handlePrometheusAuthChange = () => {
    formData.prometheusAuthType = formData.prometheusAuthEnabled ? 'basic' : 'none'
  }

  // 启用管控配置
  const handleEnableAccessControl = () => {
    enableAccessControl.value = true
    accessControlConfirmed.value = true
  }

  // 禁用管控配置
  const handleDisableAccessControl = () => {
    enableAccessControl.value = false
    accessControlConfirmed.value = true
  }

  // 复制注解到剪贴板
  const copyAnnotation = () => {
    navigator.clipboard.writeText('ikubeops.com/allow-create: "true"')
    ElMessage.success('已复制到剪贴板')
  }

  // 禁用未来的日期
  const disabledDate = (date: Date) => {
    return date.getTime() > Date.now()
  }

  // 选择集群源
  const selectClusterSource = (source: 'existing' | 'new') => {
    clusterSource.value = source
    if (source === 'existing') {
      currentStep.value = 1
    } else {
      ElMessage.info('新建集群功能正在开发中...')
    }
  }

  // 步骤点击跳转
  const handleStepClick = (index: number) => {
    if (index === 0) return
    if (index < currentStep.value) {
      currentStep.value = index
    }
  }

  // 上一步
  const handlePrev = async () => {
    prevLoading.value = true
    try {
      await new Promise((resolve) => setTimeout(resolve, 200))
      currentStep.value--
    } finally {
      prevLoading.value = false
    }
  }

  // 下一步
  const handleNext = async () => {
    nextLoading.value = true
    try {
      if (currentStep.value === 1) {
        try {
          await basicFormRef.value?.validate()
        } catch (error) {
          return
        }
      } else if (currentStep.value === 2) {
        try {
          await authFormRef.value?.validate()
        } catch (error) {
          return
        }
      } else if (currentStep.value === 3) {
        if (!selectedPriceConfigId.value) {
          ElMessage.warning('请选择一个价格配置')
          return
        }
      } else if (currentStep.value === 4) {
        // 监控配置步骤
        if (enablePrometheus.value) {
          try {
            await prometheusFormRef.value?.validate()
          } catch (error) {
            return
          }
        } else if (!skipPrometheusWarningConfirmed.value) {
          ElMessage.warning('请确认跳过监控配置的警告')
          return
        }
      } else if (currentStep.value === 5) {
        // 管控配置步骤
        if (!accessControlConfirmed.value) {
          ElMessage.warning('请选择是否启用集群管控')
          return
        }
      }

      await new Promise((resolve) => setTimeout(resolve, 200))

      // 进入费用配置步骤时加载价格配置
      if (currentStep.value === 2) {
        loadPriceConfigs()
      }

      currentStep.value++
    } finally {
      nextLoading.value = false
    }
  }

  // 测试集群连通性
  const handleTestConnection = async () => {
    try {
      await authFormRef.value?.validate()
    } catch (error) {
      return
    }

    testLoading.value = true
    try {
      const testParams: TestClusterConnectivityRequest = {
        authType: formData.authType as any,
        apiServerHost: formData.apiServerHost,
        kubeFile: formData.kubeFile,
        token: formData.token,
        caCert: formData.caCert,
        caFile: formData.caFile,
        clientCert: formData.clientCert,
        certFile: formData.certFile,
        clientKey: formData.clientKey,
        keyFile: formData.keyFile,
        insecureSkipVerify: formData.insecureSkipVerify
      }
      await testClusterConnectivityApi(testParams)
      ElMessage.success('连接测试成功！集群可正常访问')
    } finally {
      testLoading.value = false
    }
  }

  // 提交表单
  const handleSubmit = async () => {
    try {
      await basicFormRef.value?.validate()
      await authFormRef.value?.validate()
    } catch (error) {
      return
    }

    if (!selectedPriceConfigId.value) {
      ElMessage.warning('请选择一个价格配置')
      currentStep.value = 3
      return
    }

    submitLoading.value = true
    try {
      // 设置费用配置参数
      formData.priceConfigId = selectedPriceConfigId.value
      formData.billingStartTime = billingStartTime.value
        ? Math.floor(billingStartTime.value / 1000)
        : undefined

      // 设置 Prometheus 配置参数
      formData.enablePrometheus = enablePrometheus.value ? 1 : 0

      // 设置管控配置参数
      formData.allowCreate = enableAccessControl.value

      await addClusterApi(formData)
      ElNotification({
        title: '创建成功',
        message: `集群 "${formData.name}" 已成功添加`,
        type: 'success',
        duration: 3000
      })

      router.replace('/cluster/list')
    } finally {
      submitLoading.value = false
    }
  }

  // 返回上一页
  const goBack = async () => {
    backLoading.value = true
    try {
      await new Promise((resolve) => setTimeout(resolve, 200))
      router.replace('/cluster/list')
    } finally {
      backLoading.value = false
    }
  }

  // 计算属性：动态显示认证字段
  const showKubeconfigFields = computed(() => formData.authType === 'kubeconfig')
  const showTokenFields = computed(() => formData.authType === 'token')
  const showCertificateFields = computed(() => formData.authType === 'certificate')
  const showInclusterFields = computed(() => formData.authType === 'incluster')
  const showApiServerField = computed(
    () => formData.authType === 'token' || formData.authType === 'certificate'
  )

  // Prometheus 认证字段显示
  const showPrometheusBasicAuth = computed(() => formData.prometheusAuthType === 'basic')
  const showPrometheusTokenAuth = computed(() => formData.prometheusAuthType === 'token')
  const showPrometheusApiKeyAuth = computed(() => formData.prometheusAuthType === 'apikey')

  // 重置表单数据
  const resetFormData = () => {
    currentStep.value = 0
    clusterSource.value = ''
    submitLoading.value = false
    testLoading.value = false
    testPrometheusLoading.value = false
    backLoading.value = false
    nextLoading.value = false
    prevLoading.value = false
    priceConfigLoading.value = false
    priceConfigs.value = []
    selectedPriceConfigId.value = null
    billingStartTime.value = null
    priceSearchKeyword.value = ''
    enablePrometheus.value = false
    skipPrometheusWarningConfirmed.value = false
    enableAccessControl.value = false
    accessControlConfirmed.value = false

    Object.assign(formData, {
      name: '',
      avatar: '',
      description: '',
      clusterType: 'standard',
      environment: 'development',
      region: '',
      zone: '',
      datacenter: '',
      provider: 'self-hosted',
      isManaged: 0,
      nodeLb: '',
      masterLb: '',
      ingressDomain: '',
      authType: 'kubeconfig',
      apiServerHost: '',
      kubeFile: '',
      token: '',
      caCert: '',
      caFile: '',
      clientCert: '',
      certFile: '',
      clientKey: '',
      keyFile: '',
      insecureSkipVerify: 0,
      allowCreate: false,
      priceConfigId: 0,
      billingStartTime: undefined,
      enablePrometheus: 0,
      prometheusUrl: '',
      prometheusPort: 9090,
      prometheusProtocol: 'http',
      prometheusAuthEnabled: 0,
      prometheusAuthType: 'none',
      prometheusUsername: '',
      prometheusPassword: '',
      prometheusToken: '',
      prometheusAccessKey: '',
      prometheusAccessSecret: '',
      prometheusTlsEnabled: 0,
      prometheusInsecureSkipVerify: 0,
      prometheusCaCert: '',
      prometheusClientCert: '',
      prometheusClientKey: ''
    })

    setTimeout(() => {
      basicFormRef.value?.clearValidate()
      authFormRef.value?.clearValidate()
      prometheusFormRef.value?.clearValidate()
    }, 100)
  }

  onMounted(() => {
    resetFormData()
  })

  onActivated(() => {
    resetFormData()
  })

  onBeforeRouteLeave((to, from, next) => {
    resetFormData()
    next()
  })
</script>

<template>
  <div class="add-cluster-page">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <el-button text @click="goBack" class="back-btn" :loading="backLoading">
          <ArrowLeft />
          返回
        </el-button>
        <div class="header-title">
          <h1>添加集群</h1>
          <p>将现有 Kubernetes 集群添加到管理平台</p>
        </div>
      </div>
    </div>

    <!-- 步骤条 -->
    <div v-if="clusterSource" class="steps-container">
      <el-steps :active="currentStep" align-center size="small">
        <el-step
          v-for="(step, index) in steps"
          :key="index"
          :title="step.title"
          :description="step.description"
          :class="{ 'is-clickable': index < currentStep }"
          @click="handleStepClick(index)"
        />
      </el-steps>
    </div>

    <!-- 内容区域 -->
    <div class="content-container">
      <!-- 步骤 0: 选择集群类型 -->
      <div v-if="currentStep === 0" class="step-content">
        <div class="cluster-source-selection">
          <h2>请选择集群来源</h2>
          <div class="selection-cards">
            <div class="selection-card" @click="selectClusterSource('existing')">
              <div class="card-icon"><Server /></div>
              <h3>添加现有集群</h3>
              <p>将已存在的 Kubernetes 集群接入平台进行统一管理</p>
              <div class="card-features">
                <div class="feature-item"><CheckCircle /><span>支持多种认证方式</span></div>
                <div class="feature-item"><CheckCircle /><span>快速接入</span></div>
                <div class="feature-item"><CheckCircle /><span>统一管理</span></div>
              </div>
              <el-button type="primary">选择此项</el-button>
            </div>
            <div class="selection-card new-cluster" @click="selectClusterSource('new')">
              <div class="card-icon"><Sparkles /></div>
              <h3>创建新集群</h3>
              <p>在云平台或本地环境创建全新的 Kubernetes 集群</p>
              <div class="card-features">
                <div class="feature-item"><CheckCircle /><span>一键部署</span></div>
                <div class="feature-item"><CheckCircle /><span>自动配置</span></div>
                <div class="feature-item"><CheckCircle /><span>开箱即用</span></div>
              </div>
              <el-tag type="warning" class="dev-tag">开发中</el-tag>
            </div>
          </div>
        </div>
      </div>

      <!-- 步骤 1: 基础信息 -->
      <div v-if="currentStep === 1 && clusterSource === 'existing'" class="step-content">
        <el-form ref="basicFormRef" :model="formData" :rules="basicRules" label-width="0">
          <div class="info-card primary-card">
            <div class="card-header">
              <div class="header-icon"><Info /></div>
              <div class="header-text">
                <h3>集群标识</h3>
                <p>设置集群的唯一标识和显示信息</p>
              </div>
            </div>
            <div class="card-body">
              <div class="form-row">
                <div class="form-field full-width">
                  <label class="field-label required">集群名称</label>
                  <el-form-item prop="name">
                    <el-input
                      v-model="formData.name"
                      placeholder="请输入集群名称，如：prod-k8s-cluster"
                      clearable
                    >
                      <template #prefix><Server /></template>
                    </el-input>
                  </el-form-item>
                </div>
              </div>
              <div class="form-row">
                <div class="form-field full-width">
                  <label class="field-label">集群描述</label>
                  <el-form-item prop="description">
                    <el-input
                      v-model="formData.description"
                      type="textarea"
                      :rows="2"
                      placeholder="请描述集群的用途、特点等..."
                      maxlength="500"
                      show-word-limit
                    />
                  </el-form-item>
                </div>
              </div>
            </div>
          </div>

          <div class="info-card">
            <div class="card-header">
              <div class="header-icon gradient-icon"><Layers /></div>
              <div class="header-text">
                <h3>集群类型</h3>
                <p>选择最适合您业务场景的集群类型</p>
              </div>
            </div>
            <div class="card-body">
              <el-form-item prop="clusterType">
                <div class="cluster-type-grid">
                  <div
                    v-for="option in clusterTypeOptions"
                    :key="option.value"
                    class="cluster-type-card"
                    :class="{ active: formData.clusterType === option.value }"
                    @click="formData.clusterType = option.value"
                  >
                    <div class="type-icon"><component :is="option.icon" /></div>
                    <div class="type-info">
                      <h4>{{ option.label }}</h4>
                      <p>{{ option.desc }}</p>
                    </div>
                    <div class="type-check"
                      ><CheckCircle v-if="formData.clusterType === option.value"
                    /></div>
                  </div>
                </div>
              </el-form-item>
            </div>
          </div>

          <div class="info-card">
            <div class="card-header">
              <div class="header-icon gradient-icon-2"><Settings /></div>
              <div class="header-text">
                <h3>部署配置</h3>
                <p>配置集群的运行环境和基础设施信息</p>
              </div>
            </div>
            <div class="card-body">
              <div class="config-section">
                <div class="section-label">运行环境</div>
                <el-form-item prop="environment">
                  <div class="env-selector">
                    <div
                      v-for="env in environmentOptions"
                      :key="env.value"
                      class="env-item"
                      :class="{ active: formData.environment === env.value }"
                      @click="formData.environment = env.value"
                    >
                      <div class="env-icon" :style="{ background: env.color }"
                        ><component :is="env.icon"
                      /></div>
                      <span>{{ env.label }}</span>
                    </div>
                  </div>
                </el-form-item>
              </div>

              <div class="config-section">
                <div class="section-label">基础设施提供商</div>
                <el-form-item prop="provider">
                  <div class="provider-selector">
                    <div
                      v-for="provider in providerOptions"
                      :key="provider.value"
                      class="provider-item"
                      :class="{ active: formData.provider === provider.value }"
                      @click="formData.provider = provider.value"
                    >
                      <div class="provider-logo">{{ provider.logo }}</div>
                      <span>{{ provider.label }}</span>
                    </div>
                  </div>
                </el-form-item>
              </div>

              <div class="config-section">
                <div class="section-label">位置信息（可选）</div>
                <div class="location-grid">
                  <div class="location-item"
                    ><Globe /><el-input
                      v-model="formData.region"
                      placeholder="地域，如：cn-beijing"
                  /></div>
                  <div class="location-item"
                    ><MapPin /><el-input v-model="formData.zone" placeholder="可用区，如：zone-a"
                  /></div>
                  <div class="location-item"
                    ><Building2 /><el-input v-model="formData.datacenter" placeholder="数据中心"
                  /></div>
                </div>
              </div>

              <div class="config-section">
                <div class="section-label"
                  ><Network
                    style="width: 14px; height: 14px; margin-right: 6px"
                  />负载配置（可选）</div
                >
                <div class="network-grid">
                  <div class="network-item full-width">
                    <label>Ingress 域名</label>
                    <el-form-item prop="ingressDomain">
                      <el-input
                        v-model="formData.ingressDomain"
                        placeholder="如：example.com，多个用逗号分隔"
                        ><template #prefix><Globe /></template
                      ></el-input>
                    </el-form-item>
                  </div>
                  <div class="network-item">
                    <label>Node 负载均衡</label>
                    <el-form-item prop="nodeLb"
                      ><el-input v-model="formData.nodeLb" placeholder="如：192.168.1.100"
                    /></el-form-item>
                  </div>
                  <div class="network-item">
                    <label>Master 负载均衡</label>
                    <el-form-item prop="masterLb"
                      ><el-input v-model="formData.masterLb" placeholder="如：192.168.1.101"
                    /></el-form-item>
                  </div>
                </div>
              </div>

              <div class="config-section">
                <div class="managed-option">
                  <div class="option-info">
                    <label>托管集群</label>
                    <span class="option-desc">是否为云厂商托管集群（如 EKS、AKS、GKE）</span>
                  </div>
                  <el-switch
                    v-model="formData.isManaged"
                    :active-value="1"
                    :inactive-value="0"
                    active-text="是"
                    inactive-text="否"
                  />
                </div>
              </div>
            </div>
          </div>
        </el-form>
      </div>

      <!-- 步骤 2: 认证配置 -->
      <div v-if="currentStep === 2 && clusterSource === 'existing'" class="step-content">
        <el-form ref="authFormRef" :model="formData" :rules="authRules" label-width="100px">
          <div class="form-section">
            <div class="section-title"><Shield /><span>认证配置</span></div>
            <el-form-item label="认证类型" prop="authType">
              <div class="auth-type-cards">
                <div
                  v-for="option in authTypeOptions"
                  :key="option.value"
                  class="auth-type-card"
                  :class="{ active: formData.authType === option.value }"
                  @click="formData.authType = option.value"
                >
                  <div class="card-ribbon" :class="`ribbon-${option.value}`"></div>
                  <div class="card-icon-wrapper"
                    ><div class="card-icon" :class="`icon-${option.value}`"
                      ><component :is="option.icon" /></div
                  ></div>
                  <div class="card-content">
                    <div class="card-title">{{ option.label }}</div>
                    <div class="card-desc">{{ option.desc }}</div>
                  </div>
                  <div class="card-check"
                    ><CheckCircle v-if="formData.authType === option.value"
                  /></div>
                </div>
              </div>
            </el-form-item>

            <el-form-item v-if="showApiServerField" label="API Server" prop="apiServerHost">
              <el-input
                v-model="formData.apiServerHost"
                placeholder="https://kubernetes.example.com:6443"
              />
            </el-form-item>

            <div v-if="showKubeconfigFields">
              <el-form-item label="KubeConfig" prop="kubeFile" required>
                <el-input
                  v-model="formData.kubeFile"
                  type="textarea"
                  :rows="8"
                  placeholder="请粘贴完整的 KubeConfig 文件内容"
                />
              </el-form-item>
            </div>

            <div v-if="showTokenFields">
              <el-form-item label="Bearer Token" prop="token" required>
                <el-input
                  v-model="formData.token"
                  placeholder="请输入 Bearer Token"
                  show-password
                />
              </el-form-item>
              <el-form-item label="CA 证书" prop="caCert">
                <el-input
                  v-model="formData.caCert"
                  type="textarea"
                  :rows="4"
                  placeholder="CA 证书内容（PEM 格式，可选）"
                />
              </el-form-item>
            </div>

            <div v-if="showCertificateFields">
              <el-form-item label="客户端证书"
                ><el-input
                  v-model="formData.clientCert"
                  type="textarea"
                  :rows="4"
                  placeholder="客户端证书内容（PEM 格式）"
              /></el-form-item>
              <el-form-item label="客户端密钥"
                ><el-input
                  v-model="formData.clientKey"
                  type="textarea"
                  :rows="4"
                  placeholder="客户端密钥内容（PEM 格式）"
              /></el-form-item>
              <el-form-item label="CA 证书"
                ><el-input
                  v-model="formData.caCert"
                  type="textarea"
                  :rows="4"
                  placeholder="CA 证书内容（PEM 格式）"
              /></el-form-item>
            </div>

            <el-alert
              v-if="showInclusterFields"
              title="集群内部认证模式"
              type="info"
              show-icon
              :closable="false"
            >
              <div
                ><p>此模式适用于管理平台部署在 Kubernetes 集群内部的场景。</p
                ><p>系统将自动使用集群内部的 ServiceAccount 进行认证。</p></div
              >
            </el-alert>

            <el-form-item v-if="!showInclusterFields">
              <el-checkbox v-model="formData.insecureSkipVerify" :true-value="1" :false-value="0"
                >跳过 TLS 证书验证</el-checkbox
              >
              <div class="form-tip"><AlertCircle />不推荐在生产环境使用此选项</div>
            </el-form-item>

            <el-form-item>
              <el-button @click="handleTestConnection" :loading="testLoading">
                <CloudUpload v-if="!testLoading" />
                {{ testLoading ? '测试中...' : '测试连通性' }}
              </el-button>
              <span class="test-tip">建议在提交前测试集群连通性</span>
            </el-form-item>
          </div>
        </el-form>
      </div>

      <!-- 步骤 3: 费用配置 -->
      <div v-if="currentStep === 3 && clusterSource === 'existing'" class="step-content">
        <div class="pricing-section">
          <div class="section-header">
            <div class="header-icon"><DollarSign /></div>
            <div class="header-text">
              <h3>费用配置</h3>
              <p>选择集群的计费价格配置，此配置将用于资源计费</p>
            </div>
          </div>

          <div class="search-box">
            <el-input
              v-model="priceSearchKeyword"
              placeholder="搜索配置名称或描述..."
              clearable
              @clear="priceSearchKeyword = ''"
            >
              <template #prefix><Search /></template>
            </el-input>
          </div>

          <div v-loading="priceConfigLoading" class="config-list">
            <template v-if="filteredPriceConfigs.length > 0">
              <div
                v-for="config in filteredPriceConfigs"
                :key="config.id"
                class="config-item"
                :class="{ selected: selectedPriceConfigId === config.id }"
                @click="selectPriceConfig(config.id)"
              >
                <div class="config-radio"
                  ><div
                    class="radio-inner"
                    :class="{ checked: selectedPriceConfigId === config.id }"
                  ></div
                ></div>
                <div class="config-content">
                  <div class="config-header">
                    <span class="config-name">{{ config.configName }}</span>
                    <div class="config-tags"
                      ><el-tag v-if="config.isSystem === 1" type="info" size="small"
                        >系统内置</el-tag
                      ></div
                    >
                  </div>
                  <p class="config-desc">{{ config.description || '暂无描述' }}</p>
                  <div class="config-prices">
                    <span class="price-item"
                      ><Cpu class="price-icon" />CPU: ¥{{ config.cpuPrice }}/核/时</span
                    >
                    <span class="price-item"
                      ><MemoryStick class="price-icon" />内存: ¥{{
                        config.memoryPrice
                      }}/GiB/时</span
                    >
                    <span class="price-item"
                      ><HardDrive class="price-icon" />存储: ¥{{ config.storagePrice }}/GiB/时</span
                    >
                    <span class="price-item"
                      ><Layers class="price-icon" />GPU: ¥{{ config.gpuPrice }}/卡/时</span
                    >
                    <span class="price-item"
                      ><Box class="price-icon" />Pod: ¥{{ config.podPrice }}/个/时</span
                    >
                    <span class="price-item"
                      ><Settings class="price-icon" />管理费: ¥{{ config.managementFee }}/时</span
                    >
                  </div>
                </div>
              </div>
            </template>
            <el-empty
              v-else-if="!priceConfigLoading && priceConfigs.length === 0"
              description="暂无可用的价格配置，请先在费用管理中创建"
            />
          </div>

          <div v-if="selectedPriceConfigId" class="billing-time-section">
            <h4 class="section-title"><Settings />计费开始时间</h4>
            <el-date-picker
              v-model="billingStartTime"
              type="datetime"
              placeholder="选择计费开始时间（留空则默认当前时间）"
              :disabled-date="disabledDate"
              style="width: 100%"
              value-format="x"
            />
            <p class="time-hint">留空则默认从集群创建时间开始计费</p>
          </div>
        </div>
      </div>

      <!-- 步骤 4: 监控配置 -->
      <div v-if="currentStep === 4 && clusterSource === 'existing'" class="step-content">
        <div class="monitoring-section">
          <div class="section-header">
            <div class="header-icon monitoring-icon"><Activity /></div>
            <div class="header-text">
              <h3>监控配置</h3>
              <p>配置 Prometheus 监控连接信息（可选）</p>
            </div>
          </div>

          <div class="monitoring-toggle">
            <div
              class="toggle-option"
              :class="{ active: enablePrometheus }"
              @click="handleEnablePrometheus"
            >
              <div class="toggle-icon"><Monitor /></div>
              <div class="toggle-content">
                <h4>配置 Prometheus</h4>
                <p>立即配置 Prometheus 连接，启用监控功能</p>
              </div>
              <div class="toggle-check"><CheckCircle v-if="enablePrometheus" /></div>
            </div>
            <div
              class="toggle-option"
              :class="{ active: !enablePrometheus }"
              @click="handleSkipPrometheus"
            >
              <div class="toggle-icon skip-icon"><ArrowRight /></div>
              <div class="toggle-content">
                <h4>跳过此步骤</h4>
                <p>稍后在中间件管理中配置</p>
              </div>
              <div class="toggle-check"><CheckCircle v-if="!enablePrometheus" /></div>
            </div>
          </div>

          <div v-if="!enablePrometheus" class="skip-warning">
            <el-alert title="跳过监控配置提示" type="warning" show-icon :closable="false">
              <template #default>
                <div class="warning-content">
                  <p
                    ><strong>注意：</strong
                    >如果跳过此步骤，添加集群后请在集群管理中进入集群控制台<strong
                      >"中间件管理"</strong
                    >
                    页面配置 Prometheus 连接信息并设置为默认，否则监控功能不可用。</p
                  >
                  <p class="sub-warning">请确保 Prometheus 服务可被 kube-nova 管理平台正常连接。</p>
                  <el-checkbox v-model="skipPrometheusWarningConfirmed" class="confirm-checkbox"
                    >我已了解，确认跳过</el-checkbox
                  >
                </div>
              </template>
            </el-alert>
          </div>

          <div v-if="enablePrometheus" class="prometheus-form">
            <el-form
              ref="prometheusFormRef"
              :model="formData"
              :rules="prometheusRules"
              label-width="120px"
            >
              <div class="form-card">
                <h4 class="form-card-title"><Globe />连接信息</h4>
                <el-form-item label="Prometheus 地址" prop="prometheusUrl">
                  <el-input
                    v-model="formData.prometheusUrl"
                    placeholder="如：192.168.1.100 或 prometheus.example.com"
                  />
                </el-form-item>
                <el-form-item label="端口" prop="prometheusPort">
                  <el-input-number
                    v-model="formData.prometheusPort"
                    :min="1"
                    :max="65535"
                    placeholder="9090"
                    style="width: 100%"
                  />
                </el-form-item>
                <el-form-item label="协议">
                  <el-radio-group v-model="formData.prometheusProtocol">
                    <el-radio value="http">HTTP</el-radio>
                    <el-radio value="https">HTTPS</el-radio>
                  </el-radio-group>
                </el-form-item>
              </div>

              <div class="form-card">
                <h4 class="form-card-title"><Lock />认证配置</h4>
                <el-form-item label="启用认证">
                  <el-switch
                    v-model="formData.prometheusAuthEnabled"
                    :active-value="1"
                    :inactive-value="0"
                    @change="handlePrometheusAuthChange"
                  />
                </el-form-item>
                <template v-if="formData.prometheusAuthEnabled === 1">
                  <el-form-item label="认证类型">
                    <el-radio-group v-model="formData.prometheusAuthType">
                      <el-radio
                        v-for="opt in prometheusAuthTypeOptions.slice(1)"
                        :key="opt.value"
                        :value="opt.value"
                        >{{ opt.label }}</el-radio
                      >
                    </el-radio-group>
                  </el-form-item>
                  <template v-if="showPrometheusBasicAuth">
                    <el-form-item label="用户名"
                      ><el-input v-model="formData.prometheusUsername" placeholder="请输入用户名"
                    /></el-form-item>
                    <el-form-item label="密码"
                      ><el-input
                        v-model="formData.prometheusPassword"
                        type="password"
                        placeholder="请输入密码"
                        show-password
                    /></el-form-item>
                  </template>
                  <template v-if="showPrometheusTokenAuth">
                    <el-form-item label="Token"
                      ><el-input
                        v-model="formData.prometheusToken"
                        type="password"
                        placeholder="请输入 Token"
                        show-password
                    /></el-form-item>
                  </template>
                  <template v-if="showPrometheusApiKeyAuth">
                    <el-form-item label="Access Key"
                      ><el-input
                        v-model="formData.prometheusAccessKey"
                        placeholder="请输入 Access Key"
                    /></el-form-item>
                    <el-form-item label="Access Secret"
                      ><el-input
                        v-model="formData.prometheusAccessSecret"
                        type="password"
                        placeholder="请输入 Access Secret"
                        show-password
                    /></el-form-item>
                  </template>
                </template>
              </div>

              <div v-if="formData.prometheusProtocol === 'https'" class="form-card">
                <h4 class="form-card-title"><Shield />TLS 配置</h4>
                <el-form-item label="跳过证书验证">
                  <el-switch
                    v-model="formData.prometheusInsecureSkipVerify"
                    :active-value="1"
                    :inactive-value="0"
                  />
                  <span class="form-item-hint">不推荐在生产环境使用</span>
                </el-form-item>
                <template v-if="formData.prometheusInsecureSkipVerify === 0">
                  <el-form-item label="CA 证书"
                    ><el-input
                      v-model="formData.prometheusCaCert"
                      type="textarea"
                      :rows="3"
                      placeholder="CA 证书内容（PEM 格式）"
                  /></el-form-item>
                  <el-form-item label="客户端证书"
                    ><el-input
                      v-model="formData.prometheusClientCert"
                      type="textarea"
                      :rows="3"
                      placeholder="客户端证书内容（PEM 格式，可选）"
                  /></el-form-item>
                  <el-form-item label="客户端密钥"
                    ><el-input
                      v-model="formData.prometheusClientKey"
                      type="textarea"
                      :rows="3"
                      placeholder="客户端密钥内容（PEM 格式，可选）"
                  /></el-form-item>
                </template>
              </div>
            </el-form>
          </div>
        </div>
      </div>

      <!-- 步骤 5: 管控配置 -->
      <div v-if="currentStep === 5 && clusterSource === 'existing'" class="step-content">
        <div class="access-control-section">
          <div class="section-header">
            <div class="header-icon access-control-icon"><ShieldCheck /></div>
            <div class="header-text">
              <h3>管控配置</h3>
              <p>配置集群资源创建管控策略</p>
            </div>
          </div>

          <div class="access-control-toggle">
            <div
              class="toggle-option"
              :class="{ active: enableAccessControl }"
              @click="handleEnableAccessControl"
            >
              <div class="toggle-icon enable-icon"><ShieldCheck /></div>
              <div class="toggle-content">
                <h4>启用集群管控</h4>
                <p>平台将管控该集群的资源创建，只允许通过平台创建资源</p>
              </div>
              <div class="toggle-check"><CheckCircle v-if="enableAccessControl" /></div>
            </div>
            <div
              class="toggle-option"
              :class="{ active: !enableAccessControl && accessControlConfirmed }"
              @click="handleDisableAccessControl"
            >
              <div class="toggle-icon disable-icon"><ShieldOff /></div>
              <div class="toggle-content">
                <h4>不启用管控</h4>
                <p>集群资源可通过任何方式创建，平台不进行限制</p>
              </div>
              <div class="toggle-check"
                ><CheckCircle v-if="!enableAccessControl && accessControlConfirmed"
              /></div>
            </div>
          </div>

          <div v-if="enableAccessControl" class="access-control-info">
            <el-alert title="集群管控模式说明" type="info" show-icon :closable="false">
              <template #default>
                <div class="info-content">
                  <div class="info-item">
                    <div class="info-icon"><Lock /></div>
                    <div class="info-text">
                      <strong>管控生效后：</strong>
                      <p
                        >集群中的资源只能通过 kube-nova 管理平台创建，直接使用 kubectl
                        或其他工具创建资源将被拒绝。</p
                      >
                    </div>
                  </div>
                  <div class="info-item">
                    <div class="info-icon"><Tag /></div>
                    <div class="info-text">
                      <strong>临时越过管控：</strong>
                      <p>如需临时绕过管控限制，可在资源上添加注解：</p>
                      <div class="annotation-code">
                        <code>ikubeops.com/allow-create: "true"</code>
                        <el-button size="small" text type="primary" @click="copyAnnotation"
                          >复制</el-button
                        >
                      </div>
                    </div>
                  </div>
                  <div class="info-item">
                    <div class="info-icon"><AlertCircle /></div>
                    <div class="info-text">
                      <strong>注意事项：</strong>
                      <p
                        >启用管控后，请确保相关运维人员了解该策略，避免因创建资源被拒绝而影响业务。</p
                      >
                    </div>
                  </div>
                </div>
              </template>
            </el-alert>
          </div>

          <div v-if="!enableAccessControl && accessControlConfirmed" class="access-control-info">
            <el-alert title="未启用管控模式" type="warning" show-icon :closable="false">
              <template #default>
                <div class="info-content">
                  <div class="info-item">
                    <div class="info-icon warning-icon"><Unlock /></div>
                    <div class="info-text">
                      <strong>当前状态：</strong>
                      <p
                        >集群资源可以通过任何方式创建（kubectl、Helm、CI/CD
                        等），平台不会进行限制。</p
                      >
                    </div>
                  </div>
                  <div class="info-item">
                    <div class="info-icon warning-icon"><AlertCircle /></div>
                    <div class="info-text">
                      <strong>潜在风险：</strong>
                      <p
                        >未经平台管控的资源可能导致资源统计不准确、费用计算遗漏等问题。如需开启管控，可稍后在集群设置中修改。</p
                      >
                    </div>
                  </div>
                </div>
              </template>
            </el-alert>
          </div>
        </div>
      </div>

      <!-- 步骤 6: 确认信息 -->
      <div v-if="currentStep === 6 && clusterSource === 'existing'" class="step-content">
        <div class="confirm-section">
          <div class="section-title"><CheckCircle /><span>确认信息</span></div>
          <div class="confirm-content">
            <el-descriptions :column="2" border size="small">
              <el-descriptions-item label="集群名称">{{ formData.name }}</el-descriptions-item>
              <el-descriptions-item label="环境类型">{{
                environmentOptions.find((e) => e.value === formData.environment)?.label
              }}</el-descriptions-item>
              <el-descriptions-item label="集群类型">{{
                clusterTypeOptions.find((c) => c.value === formData.clusterType)?.label
              }}</el-descriptions-item>
              <el-descriptions-item label="云服务商">{{
                providerOptions.find((p) => p.value === formData.provider)?.label
              }}</el-descriptions-item>
              <el-descriptions-item label="认证类型">{{
                authTypeOptions.find((a) => a.value === formData.authType)?.label
              }}</el-descriptions-item>
              <el-descriptions-item label="托管集群">{{
                formData.isManaged === 1 ? '是' : '否'
              }}</el-descriptions-item>
              <el-descriptions-item label="费用配置" :span="2"
                ><el-tag type="success">{{
                  selectedPriceConfig?.configName || '未选择'
                }}</el-tag></el-descriptions-item
              >
              <el-descriptions-item label="Prometheus 配置" :span="2">
                <el-tag v-if="enablePrometheus" type="success"
                  >{{ formData.prometheusProtocol }}://{{ formData.prometheusUrl }}:{{
                    formData.prometheusPort
                  }}</el-tag
                >
                <el-tag v-else type="warning">未配置（需后续手动配置）</el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="集群管控" :span="2">
                <el-tag v-if="enableAccessControl" type="success">
                  已启用 - 仅允许通过平台创建资源
                </el-tag>
                <el-tag v-else type="info"> 未启用 - 允许任意方式创建资源 </el-tag>
              </el-descriptions-item>
              <el-descriptions-item label="地域" :span="2"
                >{{ formData.region || '未设置' }} /
                {{ formData.zone || '未设置' }}</el-descriptions-item
              >
              <el-descriptions-item label="描述" :span="2">{{
                formData.description || '暂无描述'
              }}</el-descriptions-item>
            </el-descriptions>

            <el-alert
              v-if="!enablePrometheus"
              title="监控配置提示"
              type="warning"
              show-icon
              :closable="false"
              class="monitor-alert"
            >
              <template #default>
                <div class="monitor-tip-content">
                  <div class="tip-icon"><Monitor /></div>
                  <div class="tip-text">
                    <p
                      ><strong>注意：</strong>添加集群后，请在集群管理中进入集群控制台<strong
                        >"中间件管理"</strong
                      >
                      页面配置 Prometheus 连接信息并设置为默认，否则监控功能不可用。</p
                    >
                    <p class="sub-tip">请确保 Prometheus 服务可被 kube-nova 管理平台正常连接。</p>
                  </div>
                </div>
              </template>
            </el-alert>

            <el-alert
              title="请确认以上信息无误"
              type="info"
              description="集群添加后，基本信息可以修改，但认证信息需要重新验证"
              show-icon
              :closable="false"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- 底部操作栏 -->
    <div v-if="clusterSource === 'existing'" class="footer-actions">
      <el-button v-if="currentStep > 0" @click="handlePrev" :loading="prevLoading"
        ><ArrowLeft v-if="!prevLoading" />上一步</el-button
      >
      <el-button
        v-if="currentStep < steps.length - 1"
        type="primary"
        @click="handleNext"
        :loading="nextLoading"
        >下一步<ArrowRight v-if="!nextLoading"
      /></el-button>
      <el-button
        v-if="currentStep === steps.length - 1"
        type="primary"
        @click="handleSubmit"
        :loading="submitLoading"
        >{{ submitLoading ? '创建中...' : '确认创建' }}</el-button
      >
    </div>
  </div>
</template>

<style scoped lang="scss">
  .add-cluster-page {
    min-height: 100vh;
    background: linear-gradient(180deg, #f8fafc 0%, #e2e8f0 100%);

    .page-header {
      background: white;
      padding: 16px 20px;
      border-bottom: 1px solid rgba(0, 0, 0, 0.06);

      .header-content {
        max-width: 1200px;
        margin: 0 auto;
        display: flex;
        align-items: center;
        gap: 20px;

        .back-btn {
          display: flex;
          align-items: center;
          gap: 6px;
          color: #6366f1;
          font-weight: 500;
          svg {
            width: 16px;
            height: 16px;
          }
        }

        .header-title {
          h1 {
            font-size: 20px;
            font-weight: 600;
            margin: 0 0 2px 0;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
          }
          p {
            color: #64748b;
            margin: 0;
            font-size: 13px;
          }
        }
      }
    }

    .steps-container {
      background: white;
      padding: 16px 20px;
      border-bottom: 1px solid rgba(0, 0, 0, 0.06);
      :deep(.el-step__title) {
        font-size: 13px;
      }
      :deep(.el-step__description) {
        font-size: 11px;
      }
      :deep(.el-step.is-clickable) {
        cursor: pointer;
        &:hover .el-step__title {
          color: #667eea;
        }
      }
    }

    .content-container {
      max-width: 1200px;
      margin: 0 auto;
      padding: 20px;
      min-height: calc(100vh - 280px);
      .step-content {
        animation: fadeInUp 0.4s ease;
      }

      .cluster-source-selection {
        max-width: 900px;
        margin: 0 auto;
        text-align: center;
        h2 {
          font-size: 22px;
          font-weight: 700;
          margin-bottom: 32px;
          color: #1e293b;
        }
        .selection-cards {
          display: grid;
          grid-template-columns: repeat(2, 1fr);
          gap: 24px;
          .selection-card {
            background: white;
            border-radius: 16px;
            padding: 32px;
            cursor: pointer;
            transition: all 0.3s ease;
            border: 2px solid transparent;
            position: relative;
            &:hover {
              transform: translateY(-6px);
              box-shadow: 0 20px 40px -12px rgba(0, 0, 0, 0.12);
              border-color: #e0e7ff;
            }
            &.new-cluster .dev-tag {
              position: absolute;
              top: 20px;
              right: 20px;
            }
            .card-icon {
              width: 72px;
              height: 72px;
              margin: 0 auto 20px;
              background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
              border-radius: 18px;
              display: flex;
              align-items: center;
              justify-content: center;
              color: white;
              svg {
                width: 36px;
                height: 36px;
              }
            }
            h3 {
              font-size: 18px;
              font-weight: 700;
              margin-bottom: 10px;
              color: #1e293b;
            }
            p {
              color: #64748b;
              margin-bottom: 20px;
              font-size: 13px;
            }
            .card-features {
              margin-bottom: 20px;
              text-align: left;
              .feature-item {
                display: flex;
                align-items: center;
                gap: 8px;
                margin-bottom: 8px;
                color: #475569;
                font-size: 13px;
                svg {
                  width: 16px;
                  height: 16px;
                  color: #10b981;
                }
              }
            }
            .el-button {
              width: 100%;
            }
          }
        }
      }

      .info-card,
      .form-section,
      .pricing-section,
      .monitoring-section,
      .access-control-section,
      .confirm-section {
        background: white;
        border-radius: 10px;
        padding: 20px;
        margin-bottom: 12px;
        box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);
      }

      .info-card.primary-card {
        border: 2px solid #e0e7ff;
      }

      .card-header,
      .section-header {
        display: flex;
        align-items: center;
        gap: 10px;
        margin-bottom: 16px;
        padding-bottom: 12px;
        border-bottom: 1px solid #f1f5f9;

        .header-icon {
          width: 36px;
          height: 36px;
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
          border-radius: 8px;
          display: flex;
          align-items: center;
          justify-content: center;
          color: white;
          &.gradient-icon {
            background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
          }
          &.gradient-icon-2 {
            background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
          }
          &.monitoring-icon {
            background: linear-gradient(135deg, #f59e0b 0%, #ef4444 100%);
          }
          &.access-control-icon {
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
          }
          svg {
            width: 18px;
            height: 18px;
          }
        }

        .header-text {
          h3 {
            font-size: 15px;
            font-weight: 600;
            margin: 0 0 2px 0;
            color: #1e293b;
          }
          p {
            font-size: 12px;
            color: #64748b;
            margin: 0;
          }
        }
      }

      .section-title {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 15px;
        font-weight: 600;
        color: #1e293b;
        margin-bottom: 16px;
        padding-bottom: 10px;
        border-bottom: 1px solid #f1f5f9;
        svg {
          width: 18px;
          height: 18px;
          color: #667eea;
        }
      }

      .access-control-toggle,
      .monitoring-toggle {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 12px;
        margin-bottom: 16px;

        .toggle-option {
          display: flex;
          align-items: center;
          gap: 12px;
          padding: 16px;
          border: 2px solid #e2e8f0;
          border-radius: 10px;
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            border-color: #cbd5e1;
          }
          &.active {
            border-color: #10b981;
            background: linear-gradient(135deg, #ecfdf5 0%, #d1fae5 100%);
          }

          .toggle-icon {
            width: 40px;
            height: 40px;
            background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            border-radius: 10px;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            svg {
              width: 20px;
              height: 20px;
            }
            &.enable-icon {
              background: linear-gradient(135deg, #10b981 0%, #059669 100%);
            }
            &.disable-icon,
            &.skip-icon {
              background: linear-gradient(135deg, #94a3b8 0%, #64748b 100%);
            }
          }

          .toggle-content {
            flex: 1;
            h4 {
              font-size: 14px;
              font-weight: 600;
              margin: 0 0 4px 0;
              color: #1e293b;
            }
            p {
              font-size: 12px;
              color: #64748b;
              margin: 0;
            }
          }

          .toggle-check svg {
            width: 20px;
            height: 20px;
            color: #10b981;
          }
        }
      }

      .monitoring-toggle .toggle-option.active {
        border-color: #818cf8;
        background: linear-gradient(135deg, #f0f4ff 0%, #e0e7ff 100%);
      }

      .access-control-info {
        margin-top: 16px;
        .info-content {
          .info-item {
            display: flex;
            gap: 12px;
            margin-bottom: 16px;
            &:last-child {
              margin-bottom: 0;
            }

            .info-icon {
              width: 32px;
              height: 32px;
              background: #dbeafe;
              border-radius: 8px;
              display: flex;
              align-items: center;
              justify-content: center;
              flex-shrink: 0;
              svg {
                width: 16px;
                height: 16px;
                color: #3b82f6;
              }
              &.warning-icon {
                background: #fef3c7;
                svg {
                  color: #f59e0b;
                }
              }
            }

            .info-text {
              flex: 1;
              strong {
                display: block;
                font-size: 13px;
                color: #1e293b;
                margin-bottom: 4px;
              }
              p {
                font-size: 12px;
                color: #64748b;
                margin: 0;
                line-height: 1.5;
              }

              .annotation-code {
                display: flex;
                align-items: center;
                gap: 8px;
                margin-top: 8px;
                padding: 8px 12px;
                background: #1e293b;
                border-radius: 6px;
                code {
                  font-family: var(--ax-font-mono);
                  font-size: 12px;
                  color: #22d3ee;
                }
              }
            }
          }
        }
      }

      .skip-warning {
        margin-bottom: 16px;
        .warning-content {
          p {
            margin: 0 0 8px 0;
            font-size: 13px;
            color: #92400e;
            line-height: 1.5;
            strong {
              color: #78350f;
            }
          }
          .sub-warning {
            font-size: 12px;
            color: #a16207;
          }
          .confirm-checkbox {
            margin-top: 12px;
          }
        }
      }

      .prometheus-form {
        .form-card {
          background: #f8fafc;
          border-radius: 8px;
          padding: 16px;
          margin-bottom: 12px;

          .form-card-title {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 14px;
            font-weight: 600;
            color: #475569;
            margin: 0 0 12px 0;
            padding-bottom: 8px;
            border-bottom: 1px solid #e2e8f0;
            svg {
              width: 16px;
              height: 16px;
              color: #667eea;
            }
          }

          .form-item-hint {
            margin-left: 10px;
            font-size: 12px;
            color: #94a3b8;
          }
        }
      }

      .form-field .field-label {
        display: block;
        font-size: 13px;
        font-weight: 600;
        color: #334155;
        margin-bottom: 8px;
        &.required::after {
          content: '*';
          color: #f56c6c;
          margin-left: 4px;
        }
      }
      .full-width {
        width: 100%;
      }

      .cluster-type-grid,
      .auth-type-cards {
        display: grid;
        gap: 10px;
      }
      .cluster-type-grid {
        grid-template-columns: repeat(3, 1fr);
      }
      .auth-type-cards {
        grid-template-columns: repeat(2, 1fr);
        margin-bottom: 16px;
      }

      .cluster-type-card,
      .auth-type-card {
        position: relative;
        padding: 14px;
        border: 2px solid #e2e8f0;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s ease;
        display: flex;
        align-items: center;
        gap: 10px;
        &:hover {
          border-color: #cbd5e1;
          background: #f8fafc;
        }
        &.active {
          border-color: #818cf8;
          background: linear-gradient(135deg, #f0f4ff 0%, #e0e7ff 100%);
        }
      }

      .type-icon,
      .card-icon {
        width: 40px;
        height: 40px;
        background: #f1f5f9;
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        flex-shrink: 0;
        svg {
          width: 20px;
          height: 20px;
        }
      }
      .cluster-type-card.active .type-icon,
      .auth-type-card.active .card-icon {
        background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        color: white;
      }

      .type-info,
      .card-content {
        flex: 1;
        min-width: 0;
        h4,
        .card-title {
          font-size: 13px;
          font-weight: 600;
          margin: 0 0 2px 0;
          color: #1e293b;
        }
        p,
        .card-desc {
          font-size: 11px;
          color: #64748b;
          margin: 0;
          line-height: 1.3;
        }
      }

      .type-check,
      .card-check {
        width: 20px;
        height: 20px;
        svg {
          width: 20px;
          height: 20px;
          color: #10b981;
        }
      }

      .config-section {
        margin-bottom: 16px;
        &:last-child {
          margin-bottom: 0;
        }
      }
      .section-label {
        font-size: 13px;
        font-weight: 600;
        color: #1e293b;
        margin-bottom: 10px;
        display: flex;
        align-items: center;
        padding-bottom: 6px;
        border-bottom: 1px solid #f1f5f9;
      }

      .env-selector,
      .provider-selector {
        display: flex;
        gap: 8px;
        flex-wrap: wrap;
      }
      .env-item,
      .provider-item {
        flex: 1;
        min-width: 80px;
        padding: 10px;
        border: 2px solid #e2e8f0;
        border-radius: 6px;
        cursor: pointer;
        transition: all 0.2s ease;
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 4px;
        &:hover {
          border-color: #cbd5e1;
        }
        &.active {
          border-color: #818cf8;
          background: #f0f4ff;
        }
        span {
          font-size: 11px;
          font-weight: 600;
          color: #334155;
        }
      }
      .env-icon {
        width: 28px;
        height: 28px;
        border-radius: 6px;
        display: flex;
        align-items: center;
        justify-content: center;
        color: white;
        svg {
          width: 14px;
          height: 14px;
        }
      }
      .provider-logo {
        font-size: 20px;
      }

      .location-grid {
        display: grid;
        grid-template-columns: repeat(3, 1fr);
        gap: 10px;
      }
      .location-item {
        display: flex;
        align-items: center;
        gap: 8px;
        svg {
          width: 16px;
          height: 16px;
          color: #64748b;
          flex-shrink: 0;
        }
      }

      .network-grid {
        display: grid;
        grid-template-columns: repeat(2, 1fr);
        gap: 12px;
      }
      .network-item {
        &.full-width {
          grid-column: 1 / -1;
        }
        label {
          display: block;
          font-size: 12px;
          font-weight: 600;
          color: #475569;
          margin-bottom: 6px;
        }
      }

      .managed-option {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 16px;
        background: #f8fafc;
        border-radius: 8px;
        border: 1px solid #e2e8f0;
        .option-info {
          label {
            display: block;
            font-size: 13px;
            font-weight: 600;
            color: #334155;
            margin-bottom: 2px;
          }
          .option-desc {
            font-size: 11px;
            color: #64748b;
          }
        }
      }

      .form-tip {
        display: flex;
        align-items: center;
        gap: 6px;
        margin-top: 6px;
        font-size: 12px;
        color: #f59e0b;
        svg {
          width: 14px;
          height: 14px;
        }
      }
      .test-tip {
        margin-left: 10px;
        font-size: 12px;
        color: #64748b;
      }

      .search-box {
        margin-bottom: 12px;
      }
      .config-list {
        max-height: 360px;
        overflow-y: auto;
        border: 1px solid #e2e8f0;
        border-radius: 8px;
        background: #fafafa;
      }
      .config-item {
        display: flex;
        gap: 12px;
        padding: 14px;
        border-bottom: 1px solid #e2e8f0;
        cursor: pointer;
        transition: all 0.2s;
        background: white;
        &:last-child {
          border-bottom: none;
        }
        &:hover:not(.selected) {
          background: #f8fafc;
        }
        &.selected {
          background: linear-gradient(135deg, #eef2ff 0%, #e0e7ff 100%);
        }
      }
      .config-radio {
        padding-top: 2px;
      }
      .radio-inner {
        width: 16px;
        height: 16px;
        border: 2px solid #cbd5e1;
        border-radius: 50%;
        position: relative;
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
        &.checked {
          border-color: #667eea;
          &::after {
            opacity: 1;
            transform: translate(-50%, -50%) scale(1);
          }
        }
      }
      .config-content {
        flex: 1;
        min-width: 0;
      }
      .config-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 4px;
      }
      .config-name {
        font-size: 14px;
        font-weight: 600;
        color: #1e293b;
      }
      .config-desc {
        font-size: 12px;
        color: #64748b;
        margin: 0 0 8px 0;
      }
      .config-prices {
        display: flex;
        flex-wrap: wrap;
        gap: 6px;
      }
      .price-item {
        display: flex;
        align-items: center;
        gap: 3px;
        font-size: 11px;
        color: #64748b;
        background: rgba(255, 255, 255, 0.8);
        padding: 2px 6px;
        border-radius: 4px;
        border: 1px solid #e2e8f0;
        .price-icon {
          width: 10px;
          height: 10px;
          color: #94a3b8;
        }
      }

      .billing-time-section {
        margin-top: 16px;
        padding-top: 16px;
        border-top: 1px solid #f1f5f9;
      }
      .time-hint {
        font-size: 11px;
        color: #94a3b8;
        margin: 6px 0 0 0;
      }

      .confirm-content {
        :deep(.el-descriptions) {
          margin-bottom: 16px;
        }
        .monitor-alert {
          margin-bottom: 12px;
        }
        .monitor-tip-content {
          display: flex;
          gap: 10px;
          align-items: flex-start;
        }
        .tip-icon {
          width: 32px;
          height: 32px;
          background: #fef3c7;
          border-radius: 8px;
          display: flex;
          align-items: center;
          justify-content: center;
          flex-shrink: 0;
          svg {
            width: 18px;
            height: 18px;
            color: #f59e0b;
          }
        }
        .tip-text {
          flex: 1;
          p {
            margin: 0 0 4px 0;
            font-size: 13px;
            color: #92400e;
            line-height: 1.5;
            strong {
              color: #78350f;
            }
          }
          .sub-tip {
            font-size: 12px;
            color: #a16207;
            margin: 0;
          }
        }
      }
    }

    .footer-actions {
      background: white;
      border-top: 1px solid rgba(0, 0, 0, 0.06);
      padding: 16px 20px;
      display: flex;
      justify-content: center;
      gap: 10px;
      position: sticky;
      bottom: 0;
      box-shadow: 0 -4px 6px -1px rgba(0, 0, 0, 0.05);
    }

    @keyframes fadeInUp {
      from {
        opacity: 0;
        transform: translateY(16px);
      }
      to {
        opacity: 1;
        transform: translateY(0);
      }
    }
  }

  @media (max-width: 768px) {
    .add-cluster-page .content-container {
      .cluster-source-selection .selection-cards {
        grid-template-columns: 1fr;
      }
      .cluster-type-grid {
        grid-template-columns: 1fr;
      }
      .auth-type-cards {
        grid-template-columns: 1fr;
      }
      .monitoring-toggle {
        grid-template-columns: 1fr;
      }
      .access-control-toggle {
        grid-template-columns: 1fr;
      }
      .location-grid {
        grid-template-columns: 1fr;
      }
      .network-grid {
        grid-template-columns: 1fr;
      }
    }
  }
</style>
