<template>
  <ElDialog
    :model-value="visible"
    :title="dialogType === 'add' ? '创建 Probe' : '编辑 Probe'"
    width="960px"
    align-center
    destroy-on-close
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @update:model-value="handleClose"
    @closed="handleClosed"
  >
    <!-- 顶部工具栏 -->
    <div class="dialog-toolbar">
      <!-- 帮助按钮 -->
      <ElPopover placement="bottom-start" :width="520" trigger="click">
        <template #reference>
          <ElButton :icon="InfoIcon" circle size="small" />
        </template>
        <div class="help-popover">
          <h4>Probe 使用说明</h4>
          <div class="help-content">
            <p><strong>什么是 Probe？</strong></p>
            <p>
              Probe 是 Prometheus Operator 提供的 CRD，用于配置 Blackbox Exporter 进行黑盒监控探测。
            </p>

            <p><strong>探测类型：</strong></p>
            <ul>
              <li><strong>HTTP/HTTPS：</strong>探测 Web 服务可用性和响应状态</li>
              <li><strong>TCP：</strong>探测端口连接性</li>
              <li><strong>ICMP：</strong>Ping 探测网络可达性</li>
              <li><strong>DNS：</strong>探测 DNS 解析</li>
            </ul>

            <p><strong>注意事项：</strong></p>
            <ul>
              <li>需要先部署 Blackbox Exporter</li>
              <li>探测间隔建议不小于 30s</li>
              <li>ICMP 探测需要容器特权</li>
            </ul>
          </div>
        </div>
      </ElPopover>

      <!-- 编辑模式切换 -->
      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <FormInput :size="14" />
          <span>表单</span>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <Code :size="14" />
          <span>YAML</span>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态提示 -->
    <div v-if="dataLoading" class="loading-container">
      <ElIcon class="is-loading" :size="32">
        <LoaderIcon />
      </ElIcon>
      <span>正在加载 Probe 配置...</span>
    </div>

    <!-- 表单模式 -->
    <div v-show="editMode === 'form' && !dataLoading" class="form-container">
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="130px"
        :disabled="submitting"
        class="probe-form"
      >
        <ElCollapse v-model="activeCollapse" class="form-collapse">
          <!-- 基础信息 -->
          <ElCollapseItem name="basic" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><InfoFilled /></ElIcon>
                <span>基础信息</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="命名空间" prop="namespace" required>
                <ElSelect
                  v-model="formData.namespace"
                  placeholder="选择命名空间"
                  filterable
                  style="width: 100%"
                  :loading="namespaceLoading"
                  @visible-change="handleNamespaceVisibleChange"
                >
                  <template #prefix>
                    <FolderTree :size="14" />
                  </template>
                  <ElOption v-for="ns in namespaces" :key="ns" :label="ns" :value="ns">
                    <div class="option-item">
                      <span class="option-name">{{ ns }}</span>
                      <ElTag v-if="ns === 'monitoring'" type="success" size="small">推荐</ElTag>
                    </div>
                  </ElOption>
                </ElSelect>
                <div class="form-item-tip">
                  <InfoIcon :size="12" />
                  <span>命名空间选择后具有最高优先级，不会被 YAML 覆盖</span>
                </div>
              </ElFormItem>

              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Probe 名称" prop="name" required>
                    <ElInput
                      v-model="formData.name"
                      placeholder="例如: blackbox-probe"
                      :disabled="dialogType === 'edit'"
                      maxlength="63"
                      show-word-limit
                    >
                      <template #suffix>
                        <ElTooltip
                          content="小写字母、数字、连字符，以字母或数字开头结尾"
                          placement="top"
                        >
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="Job 名称" prop="jobName" required>
                    <ElInput
                      v-model="formData.jobName"
                      placeholder="例如: blackbox-exporter"
                      maxlength="63"
                    >
                      <template #suffix>
                        <ElTooltip
                          content="Prometheus 中显示的 Job 名称，用于标识这组探测任务"
                          placement="top"
                        >
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </div>
          </ElCollapseItem>

          <!-- Prober 配置 -->
          <ElCollapseItem name="prober" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Aim /></ElIcon>
                <span>Prober 配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="Prober URL" prop="prober.url" required>
                    <ElInput v-model="formData.prober.url" placeholder="blackbox-exporter:19115">
                      <template #prepend>
                        <ElSelect v-model="formData.prober.scheme" style="width: 90px">
                          <ElOption label="http" value="http" />
                          <ElOption label="https" value="https" />
                        </ElSelect>
                      </template>
                      <template #suffix>
                        <ElTooltip placement="top">
                          <template #content>
                            <div style="max-width: 320px">
                              <div style="margin-bottom: 8px; font-weight: 600"
                                >Blackbox Exporter 服务地址</div
                              >
                              <div style="margin-bottom: 4px">格式：服务名:端口 或 IP:端口</div>
                              <div style="margin-bottom: 4px">默认：blackbox-exporter:19115</div>
                            </div>
                          </template>
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="Prober Path" prop="prober.path">
                    <ElInput v-model="formData.prober.path" placeholder="/probe">
                      <template #suffix>
                        <ElTooltip
                          content="Blackbox Exporter 的探测路径，通常使用默认值 /probe"
                          placement="top"
                        >
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElFormItem label="探测模块" prop="module" required>
                <ElSelect
                  v-model="formData.module"
                  placeholder="选择探测模块"
                  filterable
                  allow-create
                  style="width: 100%"
                  @change="handleModuleChange"
                >
                  <ElOptionGroup label="🌐 HTTP/HTTPS 探测">
                    <ElOption value="http_2xx">
                      <div class="module-option">
                        <span class="module-name">http_2xx</span>
                        <span class="module-desc">HTTP GET 请求，期望 2xx 状态码</span>
                      </div>
                    </ElOption>
                    <ElOption value="http_post_2xx">
                      <div class="module-option">
                        <span class="module-name">http_post_2xx</span>
                        <span class="module-desc">HTTP POST 请求，期望 2xx 状态码</span>
                      </div>
                    </ElOption>
                    <ElOption value="http_3xx">
                      <div class="module-option">
                        <span class="module-name">http_3xx</span>
                        <span class="module-desc">HTTP 请求，期望 3xx 重定向</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>

                  <ElOptionGroup label="🔌 TCP 探测">
                    <ElOption value="tcp_connect">
                      <div class="module-option">
                        <span class="module-name">tcp_connect</span>
                        <span class="module-desc">TCP 端口连接探测</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>

                  <ElOptionGroup label="📡 ICMP 探测">
                    <ElOption value="icmp">
                      <div class="module-option">
                        <span class="module-name">icmp</span>
                        <span class="module-desc">ICMP Ping 探测（需要特权）</span>
                      </div>
                    </ElOption>
                    <ElOption value="icmp_ttl5">
                      <div class="module-option">
                        <span class="module-name">icmp_ttl5</span>
                        <span class="module-desc">ICMP Ping，TTL=5</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>

                  <ElOptionGroup label="🔍 DNS 探测">
                    <ElOption value="dns_udp">
                      <div class="module-option">
                        <span class="module-name">dns_udp</span>
                        <span class="module-desc">DNS UDP 查询</span>
                      </div>
                    </ElOption>
                    <ElOption value="dns_tcp">
                      <div class="module-option">
                        <span class="module-name">dns_tcp</span>
                        <span class="module-desc">DNS TCP 查询</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>

                  <ElOptionGroup label="🔐 SSL/TLS 探测">
                    <ElOption value="ssl_expiry">
                      <div class="module-option">
                        <span class="module-name">ssl_expiry</span>
                        <span class="module-desc">检查 SSL 证书过期时间</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>

                  <ElOptionGroup label="📧 其他协议">
                    <ElOption value="smtp_starttls">
                      <div class="module-option">
                        <span class="module-name">smtp_starttls</span>
                        <span class="module-desc">SMTP STARTTLS 探测</span>
                      </div>
                    </ElOption>
                    <ElOption value="pop3s_banner">
                      <div class="module-option">
                        <span class="module-name">pop3s_banner</span>
                        <span class="module-desc">POP3S 横幅探测</span>
                      </div>
                    </ElOption>
                    <ElOption value="ssh_banner">
                      <div class="module-option">
                        <span class="module-name">ssh_banner</span>
                        <span class="module-desc">SSH 横幅探测</span>
                      </div>
                    </ElOption>
                    <ElOption value="irc_banner">
                      <div class="module-option">
                        <span class="module-name">irc_banner</span>
                        <span class="module-desc">IRC 横幅探测</span>
                      </div>
                    </ElOption>
                  </ElOptionGroup>
                </ElSelect>

                <div class="form-item-tip" style="margin-top: 8px">
                  <InfoIcon :size="12" />
                  <span>{{ currentModuleDescription }}</span>
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 探测目标 -->
          <ElCollapseItem name="targets" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Location /></ElIcon>
                <span>探测目标</span>
              </div>
            </template>
            <div class="collapse-content">
              <div class="targets-container">
                <div class="targets-header">
                  <span>静态目标列表</span>
                  <ElButton type="primary" :icon="Plus" size="small" plain @click="addTarget">
                    添加目标
                  </ElButton>
                </div>

                <div v-if="formData.targets.length > 0" class="targets-list">
                  <div v-for="(target, index) in formData.targets" :key="index" class="target-item">
                    <ElInput
                      v-model="formData.targets[index]"
                      :placeholder="currentTargetPlaceholder"
                      style="flex: 1"
                    >
                      <template #prepend>
                        <span style="width: 50px">目标 {{ index + 1 }}</span>
                      </template>
                    </ElInput>
                    <ElButton
                      type="danger"
                      :icon="Trash2"
                      circle
                      size="small"
                      @click="removeTarget(index)"
                    />
                  </div>
                </div>

                <ElAlert v-else type="info" :closable="false" show-icon>
                  <template #title>请添加至少一个探测目标</template>
                </ElAlert>

                <!-- 动态目标格式说明 -->
                <div class="target-format-guide" v-if="currentTargetGuide">
                  <div class="guide-header">
                    <component :is="currentModuleIcon" :size="16" />
                    <span>{{ currentModuleType }} 目标格式说明</span>
                  </div>
                  <div class="guide-content">
                    <div v-for="(example, idx) in currentTargetGuide" :key="idx" class="guide-item">
                      <CheckCircle2 :size="12" />
                      <span>{{ example }}</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <!-- 高级配置 -->
          <ElCollapseItem name="advanced" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Setting /></ElIcon>
                <span>高级配置（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElRow :gutter="24">
                <ElCol :span="12">
                  <ElFormItem label="探测间隔" prop="interval">
                    <ElInput v-model="formData.interval" placeholder="例如: 30s, 1m, 5m">
                      <template #append>秒/分钟</template>
                      <template #suffix>
                        <ElTooltip placement="top">
                          <template #content>
                            <div>探测执行频率</div>
                            <div>格式：数字+单位（s秒/m分钟/h小时）</div>
                            <div>示例：30s, 1m, 5m, 1h</div>
                          </template>
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="探测超时" prop="scrapeTimeout">
                    <ElInput v-model="formData.scrapeTimeout" placeholder="例如: 10s, 30s">
                      <template #append>秒</template>
                      <template #suffix>
                        <ElTooltip placement="top">
                          <template #content>
                            <div>单次探测的超时时间</div>
                            <div>格式：数字+s（秒）</div>
                            <div>必须小于探测间隔</div>
                          </template>
                          <HelpCircle :size="13" class="help-icon" />
                        </ElTooltip>
                      </template>
                    </ElInput>
                  </ElFormItem>
                </ElCol>
              </ElRow>
            </div>
          </ElCollapseItem>

          <!-- 标签和注解 -->
          <ElCollapseItem name="labels" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><PriceTag /></ElIcon>
                <span>标签和注解（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="标签 (Labels)">
                <ElTable :data="formData.labels" border style="width: 100%">
                  <ElTableColumn label="标签名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="标签名" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="标签值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="标签值" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeLabel($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addLabel"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加标签
                </ElButton>
              </ElFormItem>

              <ElFormItem label="注解 (Annotations)">
                <ElTable :data="formData.annotations" border style="width: 100%">
                  <ElTableColumn label="注解名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="注解名" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="注解值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="注解值" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeAnnotation($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addAnnotation"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加注解
                </ElButton>
              </ElFormItem>
            </div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
    </div>

    <!-- YAML 模式 -->
    <div v-show="editMode === 'yaml' && !dataLoading" class="yaml-container">
      <ArtYamlEditor v-model="yamlContent" height="550px" @change="handleYamlChange" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleCancel" :disabled="submitting">取消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ submitting ? '提交中...' : dialogType === 'add' ? '创建' : '保存' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, watch, nextTick, computed } from 'vue'
  import { ElMessage } from 'element-plus'
  import { InfoFilled, Aim, Location, Setting, PriceTag } from '@element-plus/icons-vue'
  import yaml from 'js-yaml'
  import {
    Target,
    FormInput,
    Code,
    Plus,
    Trash2,
    HelpCircle,
    Info as InfoIcon,
    Globe,
    FolderTree,
    CheckCircle2,
    Wifi,
    Network,
    Radio,
    Search,
    Lock,
    Mail,
    Loader as LoaderIcon
  } from 'lucide-vue-next'
  import {
    createProbeApi,
    updateProbeApi,
    getProbeYamlApi,
    type ProbeListItem
  } from '@/api/workload/monitor'
  import { getClusterNamespaceListApi } from '@/api/manager/cluster'
  import ArtYamlEditor from '@/components/yaml-editor-pro/index.vue'

  interface Props {
    visible: boolean
    dialogType: 'add' | 'edit'
    probeData?: ProbeListItem
    clusterUuid: string
  }

  interface KeyValue {
    key: string
    value: string
  }

  const props = withDefaults(defineProps<Props>(), {
    probeData: undefined
  })

  const emit = defineEmits(['close', 'success'])

  const handleClose = (val: boolean) => {
    if (!val) {
      emit('close')
    }
  }

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const namespaceLoading = ref(false)
  const namespaces = ref<string[]>([])

  // 折叠面板激活状态
  const activeCollapse = ref(['basic', 'prober', 'targets', 'advanced', 'labels'])

  const formData = ref({
    namespace: 'monitoring',
    name: '',
    jobName: '',
    prober: {
      url: 'blackbox-exporter:19115',
      scheme: 'http',
      path: '/probe'
    },
    module: 'http_2xx',
    interval: '',
    scrapeTimeout: '',
    targets: [] as string[],
    labels: [] as KeyValue[],
    annotations: [] as KeyValue[]
  })

  const formRules = {
    namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }],
    name: [
      { required: true, message: '请输入 Probe 名称', trigger: 'blur' },
      { pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/, message: '名称格式不正确', trigger: 'blur' }
    ],
    jobName: [{ required: true, message: '请输入 Job 名称', trigger: 'blur' }],
    'prober.url': [{ required: true, message: '请输入 Prober URL', trigger: 'blur' }],
    module: [{ required: true, message: '请选择探测模块', trigger: 'change' }]
  }

  // 模块配置映射
  const moduleConfigs: Record<
    string,
    {
      type: string
      icon: any
      description: string
      placeholder: string
      examples: string[]
    }
  > = {
    // HTTP 探测
    http_2xx: {
      type: 'HTTP/HTTPS',
      icon: Wifi,
      description: '通过 HTTP GET 请求探测 Web 服务，期望返回 2xx 状态码（200-299）',
      placeholder: 'https://example.com 或 http://192.168.1.1:8080',
      examples: [
        'https://www.example.com',
        'http://192.168.1.100:8080',
        'http://my-service.default.svc:80',
        'https://api.example.com/health'
      ]
    },
    http_post_2xx: {
      type: 'HTTP POST',
      icon: Wifi,
      description: '通过 HTTP POST 请求探测 Web 服务，期望返回 2xx 状态码',
      placeholder: 'https://api.example.com/endpoint',
      examples: [
        'https://api.example.com/webhook',
        'http://service.namespace.svc:8080/api',
        'http://192.168.1.100:3000/check'
      ]
    },
    http_3xx: {
      type: 'HTTP 重定向',
      icon: Wifi,
      description: '通过 HTTP 请求探测重定向服务，期望返回 3xx 状态码（301, 302 等）',
      placeholder: 'http://example.com (期望重定向)',
      examples: ['http://example.com (重定向到 https)', 'http://old-domain.com (重定向到新域名)']
    },
    // TCP 探测
    tcp_connect: {
      type: 'TCP',
      icon: Network,
      description: '探测 TCP 端口连接性，检查端口是否开放和可连接',
      placeholder: '192.168.1.1:3306 或 mysql.default.svc:3306',
      examples: [
        '192.168.1.100:3306 (MySQL)',
        '192.168.1.100:5432 (PostgreSQL)',
        'redis.default.svc:6379 (Redis)',
        'kafka.kafka.svc:9092 (Kafka)',
        '10.0.0.5:22 (SSH)'
      ]
    },
    // ICMP 探测
    icmp: {
      type: 'ICMP Ping',
      icon: Radio,
      description: '使用 ICMP 协议进行 Ping 探测，检查网络可达性（需要容器特权）',
      placeholder: '8.8.8.8 或 example.com',
      examples: [
        '8.8.8.8 (Google DNS)',
        '1.1.1.1 (Cloudflare DNS)',
        'example.com (域名)',
        '192.168.1.1 (网关)'
      ]
    },
    icmp_ttl5: {
      type: 'ICMP (TTL=5)',
      icon: Radio,
      description: 'ICMP Ping 探测，设置 TTL=5，用于检测近距离网络',
      placeholder: '192.168.1.1',
      examples: ['192.168.1.1 (本地网关)', '10.0.0.1 (内网路由器)']
    },
    // DNS 探测
    dns_udp: {
      type: 'DNS (UDP)',
      icon: Search,
      description: '通过 UDP 协议查询 DNS 记录，检查 DNS 服务是否正常',
      placeholder: 'example.com',
      examples: ['example.com', 'www.google.com', 'my-service.default.svc.cluster.local']
    },
    dns_tcp: {
      type: 'DNS (TCP)',
      icon: Search,
      description: '通过 TCP 协议查询 DNS 记录，适用于大型 DNS 响应或安全场景',
      placeholder: 'example.com',
      examples: ['example.com', 'large-dns-response.com']
    },
    // SSL/TLS 探测
    ssl_expiry: {
      type: 'SSL 证书',
      icon: Lock,
      description: '检查 SSL/TLS 证书的有效期和配置，用于证书过期监控',
      placeholder: 'https://example.com',
      examples: ['https://www.example.com', 'https://api.example.com:443']
    },
    // 其他协议
    smtp_starttls: {
      type: 'SMTP',
      icon: Mail,
      description: '探测 SMTP 邮件服务器，检查 STARTTLS 支持',
      placeholder: 'smtp.example.com:587',
      examples: ['smtp.gmail.com:587', 'smtp.example.com:25']
    },
    pop3s_banner: {
      type: 'POP3S',
      icon: Mail,
      description: '探测 POP3S 邮件服务器横幅信息',
      placeholder: 'pop3.example.com:995',
      examples: ['pop3.gmail.com:995', 'pop3.example.com:995']
    },
    ssh_banner: {
      type: 'SSH',
      icon: Network,
      description: '探测 SSH 服务器横幅信息',
      placeholder: '192.168.1.100:22',
      examples: ['192.168.1.100:22', 'ssh.example.com:22']
    },
    irc_banner: {
      type: 'IRC',
      icon: Network,
      description: '探测 IRC 服务器横幅信息',
      placeholder: 'irc.example.com:6667',
      examples: ['irc.freenode.net:6667', 'irc.example.com:6667']
    }
  }

  // 当前模块配置
  const currentModuleConfig = computed(() => {
    return (
      moduleConfigs[formData.value.module] || {
        type: '自定义',
        icon: Target,
        description: '自定义探测模块，请确保已在 Blackbox Exporter 中配置此模块',
        placeholder: '根据自定义模块要求输入目标',
        examples: []
      }
    )
  })

  const currentModuleType = computed(() => currentModuleConfig.value.type)
  const currentModuleIcon = computed(() => currentModuleConfig.value.icon)
  const currentModuleDescription = computed(() => currentModuleConfig.value.description)
  const currentTargetPlaceholder = computed(() => currentModuleConfig.value.placeholder)
  const currentTargetGuide = computed(() => currentModuleConfig.value.examples)

  // 加载命名空间列表
  const handleNamespaceVisibleChange = async (visible: boolean) => {
    if (visible && namespaces.value.length === 0) {
      await loadNamespaces()
    }
  }

  const loadNamespaces = async () => {
    if (!props.clusterUuid || namespaceLoading.value) return

    namespaceLoading.value = true
    try {
      const nsList = await getClusterNamespaceListApi(props.clusterUuid)
      namespaces.value = nsList || []

      if (
        props.dialogType === 'add' &&
        nsList.includes('monitoring') &&
        !formData.value.namespace
      ) {
        formData.value.namespace = 'monitoring'
      }
    } catch (error) {
      console.error('加载命名空间失败:', error)
      namespaces.value = []
    } finally {
      namespaceLoading.value = false
    }
  }

  // 处理探测模块变化
  const handleModuleChange = (module: string) => {
    console.log('探测模块切换:', module)
    // 可以在这里添加更多的联动逻辑
  }

  const addTarget = () => {
    formData.value.targets.push('')
  }

  const removeTarget = (index: number) => {
    formData.value.targets.splice(index, 1)
  }

  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  const objectToArray = (obj?: Record<string, string>): KeyValue[] => {
    if (!obj) return []
    return Object.entries(obj)
      .filter(([key]) => key)
      .map(([key, value]) => ({ key, value }))
  }

  const arrayToObject = (arr: KeyValue[]): Record<string, string> => {
    const obj: Record<string, string> = {}
    arr.forEach((item) => {
      if (item.key && item.key.trim()) obj[item.key.trim()] = item.value
    })
    return obj
  }

  const formToYaml = () => {
    const probeObj: any = {
      apiVersion: 'monitoring.coreos.com/v1',
      kind: 'Probe',
      metadata: {
        name: formData.value.name || 'probe-name',
        namespace: formData.value.namespace || 'monitoring'
      },
      spec: {
        jobName: formData.value.jobName || 'blackbox-exporter'
      }
    }

    // Labels
    const labelsObj = arrayToObject(formData.value.labels)
    if (Object.keys(labelsObj).length > 0) {
      probeObj.metadata.labels = labelsObj
    }

    // Annotations
    const annotationsObj = arrayToObject(formData.value.annotations)
    if (Object.keys(annotationsObj).length > 0) {
      probeObj.metadata.annotations = annotationsObj
    }

    // Prober
    if (formData.value.prober.url) {
      probeObj.spec.prober = {
        url: formData.value.prober.url
      }
      if (formData.value.prober.scheme && formData.value.prober.scheme !== 'http') {
        probeObj.spec.prober.scheme = formData.value.prober.scheme
      }
      if (formData.value.prober.path && formData.value.prober.path !== '/probe') {
        probeObj.spec.prober.path = formData.value.prober.path
      }
    }

    // Module
    if (formData.value.module) {
      probeObj.spec.module = formData.value.module
    }

    // Interval
    if (formData.value.interval) {
      probeObj.spec.interval = formData.value.interval
    }

    // ScrapeTimeout
    if (formData.value.scrapeTimeout) {
      probeObj.spec.scrapeTimeout = formData.value.scrapeTimeout
    }

    // Targets
    if (formData.value.targets.length > 0) {
      const validTargets = formData.value.targets.filter((t) => t && t.trim())
      if (validTargets.length > 0) {
        probeObj.spec.targets = {
          staticConfig: {
            static: validTargets
          }
        }
      }
    }

    yamlContent.value = yaml.dump(probeObj, { indent: 2 })
  }

  const yamlToForm = (yamlStr: string) => {
    try {
      const obj = yaml.load(yamlStr) as any

      const selectedNamespace = formData.value.namespace

      formData.value.name = obj.metadata?.name || ''
      formData.value.namespace = selectedNamespace || obj.metadata?.namespace || 'monitoring'
      formData.value.jobName = obj.spec?.jobName || ''

      // Prober - 确保有默认值
      formData.value.prober = {
        url: obj.spec?.prober?.url || 'blackbox-exporter:19115',
        scheme: obj.spec?.prober?.scheme || 'http',
        path: obj.spec?.prober?.path || '/probe'
      }

      formData.value.module = obj.spec?.module || 'http_2xx'
      formData.value.interval = obj.spec?.interval || ''
      formData.value.scrapeTimeout = obj.spec?.scrapeTimeout || ''

      // Targets
      if (obj.spec?.targets?.staticConfig?.static) {
        formData.value.targets = obj.spec.targets.staticConfig.static
      } else {
        formData.value.targets = []
      }

      // Labels
      formData.value.labels = objectToArray(obj.metadata?.labels)

      // Annotations
      formData.value.annotations = objectToArray(obj.metadata?.annotations)
    } catch (error) {
      console.error('❌ [yamlToForm] YAML 解析失败:', error)
      throw error
    }
  }

  const loadProbeYAML = async () => {
    if (!props.clusterUuid || !props.probeData) {
      return
    }

    dataLoading.value = true
    try {
      await loadNamespaces()

      const yamlStr = await getProbeYamlApi({
        clusterUuid: props.clusterUuid,
        namespace: props.probeData.namespace,
        name: props.probeData.name
      })

      yamlContent.value = yamlStr
      formData.value.namespace = props.probeData.namespace
      yamlToForm(yamlStr)
      formData.value.namespace = props.probeData.namespace
    } catch (error: any) {
      console.error('❌ [loadProbeYAML] 加载失败:', error)
      ElMessage.error('加载 Probe 配置失败')
      emit('close')
    } finally {
      dataLoading.value = false
    }
  }

  watch(
    () => props.visible,
    async (val) => {
      if (val) {
        if (props.dialogType === 'edit' && props.probeData) {
          await loadProbeYAML()
        } else if (props.dialogType === 'add') {
          await loadNamespaces()

          // 确保默认值正确设置
          formData.value.prober.url = 'blackbox-exporter:19115'
          formData.value.prober.scheme = 'http'
          formData.value.prober.path = '/probe'
          formData.value.module = 'http_2xx'
          formData.value.targets = ['']

          await nextTick()
          formToYaml()
        }
      }
    },
    { immediate: true }
  )

  watch(editMode, (newMode, oldMode) => {
    if (newMode === 'yaml' && oldMode === 'form') {
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      try {
        const currentNamespace = formData.value.namespace
        yamlToForm(yamlContent.value)
        formData.value.namespace = currentNamespace
        ElMessage({
          type: 'success',
          message: '已切换到表单模式',
          duration: 2000
        })
      } catch (error) {
        console.error('❌ [watch editMode] YAML 解析失败:', error)
        ElMessage.error('YAML 格式错误，请检查后重试')
        nextTick(() => {
          editMode.value = 'yaml'
        })
      }
    }
  })

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  const handleSubmit = async () => {
    if (!props.clusterUuid) {
      return
    }

    if (editMode.value === 'form') {
      try {
        await formRef.value?.validate()
      } catch (error) {
        return
      }

      if (formData.value.targets.length === 0 || !formData.value.targets.some((t) => t.trim())) {
        ElMessage.warning('请至少添加一个有效的探测目标')
        return
      }

      formToYaml()
    }

    try {
      const obj = yaml.load(yamlContent.value) as any
      obj.metadata.namespace = formData.value.namespace
      yamlContent.value = yaml.dump(obj, { indent: 2 })
    } catch (error) {
      console.error('修正 YAML namespace 失败:', error)
    }

    submitting.value = true
    try {
      const requestData = {
        clusterUuid: props.clusterUuid,
        namespace: formData.value.namespace,
        yamlStr: yamlContent.value
      }

      if (props.dialogType === 'add') {
        await createProbeApi(requestData)
        ElMessage.success('创建成功')
      } else {
        await updateProbeApi(requestData)
        ElMessage.success('更新成功')
      }

      emit('success')
      emit('close')
    } catch (error: any) {
      console.error('提交失败:', error)
    } finally {
      submitting.value = false
    }
  }

  const handleCancel = () => {
    emit('close')
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    formData.value = {
      namespace: 'monitoring',
      name: '',
      jobName: '',
      prober: {
        url: 'blackbox-exporter:19115',
        scheme: 'http',
        path: '/probe'
      },
      module: 'http_2xx',
      interval: '',
      scrapeTimeout: '',
      targets: [],
      labels: [],
      annotations: []
    }

    namespaces.value = []
    yamlContent.value = ''
    editMode.value = 'form'
  }

  watch(
    () => [formData.value],
    () => {
      if (editMode.value === 'form' && !dataLoading.value) {
        formToYaml()
      }
    },
    { deep: true }
  )
</script>

<style lang="scss" scoped>
  @use '@/views/cluster/resource/styles/resource-form.scss' as *;

  .dialog-toolbar {
    @include dialog-toolbar;

    :deep(.el-radio-group) {
      .el-radio-button__inner {
        display: flex;
        align-items: center;
        gap: 6px;
      }
    }
  }

  .help-popover {
    @include help-popover;
  }

  .loading-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    color: #909399;
    gap: 16px;

    .el-icon {
      color: #409eff;
    }
  }

  .form-container {
    @include form-container;
  }

  .yaml-container {
    @include yaml-container;
  }

  .dialog-footer {
    @include dialog-footer;
  }

  .form-item-tip {
    @include form-item-tip;
    display: flex;
    align-items: center;
    gap: 6px;
    margin-top: 6px;
    padding: 8px 12px;
    background: #f0f9ff;
    border-radius: 4px;
    border: 1px solid #bfdbfe;
    color: #1e40af;

    svg {
      flex-shrink: 0;
    }
  }

  .help-icon {
    cursor: help;
    color: #909399;
    transition: color 0.2s;

    &:hover {
      color: #606266;
    }
  }

  // Collapse styles
  .form-collapse {
    border: none;
    background: transparent;

    :deep(.el-collapse-item) {
      margin-bottom: 12px;
      border: 1px solid #dcdfe6;
      border-radius: 4px;
      overflow: hidden;
      background: #fff;

      &.is-active {
        border-color: #409eff;
      }

      &:last-child {
        margin-bottom: 0;
      }

      .el-collapse-item__header {
        height: 48px;
        line-height: 48px;
        padding: 0 16px;
        background: #f5f7fa;
        border-bottom: 1px solid #e4e7ed;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        transition: all 0.3s;

        &:hover {
          background: #ecf5ff;
        }

        .el-collapse-item__arrow {
          margin-right: 8px;
          color: #909399;
          transition: transform 0.3s;
        }

        &.is-active {
          .el-collapse-item__arrow {
            transform: rotate(90deg);
            color: #409eff;
          }
        }
      }

      .el-collapse-item__wrap {
        background: #fff;
        border-bottom: none;
      }

      .el-collapse-item__content {
        padding: 20px 24px;
      }
    }
  }

  .collapse-section-item {
    .collapse-title {
      display: flex;
      align-items: center;
      width: 100%;

      .title-icon {
        margin-right: 8px;
        font-size: 16px;
        color: #409eff;
      }

      span {
        flex: 1;
      }
    }
  }

  .targets-container {
    .targets-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;
      font-weight: 500;
      font-size: 13px;
      color: #495057;
    }

    .targets-list {
      display: flex;
      flex-direction: column;
      gap: 10px;
      margin-bottom: 12px;
    }

    .target-item {
      display: flex;
      align-items: center;
      gap: 8px;
    }
  }

  .target-format-guide {
    margin-top: 16px;
    padding: 14px;
    background: #f5f7fa;
    border-radius: 4px;
    border: 1px solid #e4e7ed;

    .guide-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      font-weight: 600;
      font-size: 13px;
      color: #409eff;

      svg {
        color: #409eff;
      }
    }

    .guide-content {
      display: flex;
      flex-direction: column;
      gap: 8px;
    }

    .guide-item {
      display: flex;
      align-items: center;
      gap: 8px;
      padding: 6px 10px;
      background: white;
      border-radius: 4px;
      font-size: 12px;
      color: #334155;
      font-family: var(--ax-font-mono);

      svg {
        flex-shrink: 0;
        color: #67c23a;
      }
    }
  }

  .labels-list {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 8px;
  }

  .label-row {
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 12px;
    background: #fafbfc;
    border-radius: 4px;
    border: 1px solid #e4e7ed;

    .separator {
      color: #6c757d;
      font-weight: 600;
      font-size: 13px;
    }
  }

  .option-item {
    display: flex;
    align-items: center;
    gap: 8px;

    .option-name {
      flex: 1;
      font-weight: 500;
      font-size: 12px;
    }
  }

  .module-option {
    display: flex;
    flex-direction: column;
    gap: 2px;

    .module-name {
      font-weight: 600;
      font-size: 13px;
      color: #303133;
    }

    .module-desc {
      font-size: 11px;
      color: #909399;
    }
  }

  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    font-weight: 500;
    color: #303133;
  }

  :deep(.el-input__inner) {
    font-size: 13px;
  }

  :deep(.el-select) {
    font-size: 13px;
  }

  :deep(.el-button) {
    font-size: 12px;
  }

  :deep(.el-input-group__prepend) {
    background-color: #f5f7fa;
    color: #606266;
    font-weight: 500;
    font-size: 12px;
  }

  :deep(.el-input-group__append) {
    background-color: #f5f7fa;
    color: #606266;
    font-size: 12px;
  }
</style>
