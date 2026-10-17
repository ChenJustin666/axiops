<template>
  <ElDialog
    :model-value="visible"
    :title="dialogTitle"
    width="900px"
    top="5vh"
    :close-on-click-modal="false"
    @close="handleClose"
    class="service-dialog"
  >
    <!-- 编辑模式切换 -->
    <div class="unified-mode-switch">
      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <div class="radio-content">
            <Settings :size="13" />
            <span>表单编辑</span>
          </div>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <div class="radio-content">
            <FileCode :size="13" />
            <span>YAML 编辑</span>
          </div>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态 -->
    <div v-if="dataLoading" class="unified-loading">
      <ElIcon class="is-loading" :size="32">
        <Loader2 />
      </ElIcon>
      <div class="loading-text">正在加载 Service 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="service-form-container">
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="120px"
        label-position="right"
        class="service-form"
      >
        <ElCollapse v-model="activeNames" class="form-collapse">
          <!-- 基本信息 -->
          <ElCollapseItem name="basic" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <FileText :size="16" class="title-icon" />
                <span>基本信息</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="Service 名称" prop="name">
                <ElInput
                  v-model="formData.name"
                  placeholder="请输入 Service 名称"
                  clearable
                  :disabled="dialogType === 'edit'"
                />
              </ElFormItem>

              <ElFormItem label="Service 类型" prop="type">
                <ElRadioGroup v-model="formData.type" @change="handleTypeChange">
                  <ElRadioButton value="ClusterIP">
                    <Network :size="14" />
                    ClusterIP
                  </ElRadioButton>
                  <ElRadioButton value="NodePort">
                    <Share2 :size="14" />
                    NodePort
                  </ElRadioButton>
                  <ElRadioButton value="LoadBalancer">
                    <CloudUpload :size="14" />
                    LoadBalancer
                  </ElRadioButton>
                  <ElRadioButton value="ExternalName">
                    <ExternalLink :size="14" />
                    ExternalName
                  </ElRadioButton>
                </ElRadioGroup>
              </ElFormItem>

              <!-- ExternalName 专用字段 -->
              <ElFormItem
                v-if="formData.type === 'ExternalName'"
                label="外部域名"
                prop="externalName"
              >
                <ElInput
                  v-model="formData.externalName"
                  placeholder="例如: my.database.example.com"
                  clearable
                />
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 端口配置 -->
          <ElCollapseItem
            v-if="formData.type !== 'ExternalName'"
            name="ports"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <Network :size="16" class="title-icon" />
                <span>端口配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem prop="ports" label-width="0">
                <div class="ports-table">
                  <!-- 表头 -->
                  <div class="ports-header">
                    <div class="col-name">端口名称</div>
                    <div class="col-protocol">协议</div>
                    <div class="col-port">Pod 端口</div>
                    <div class="col-arrow"></div>
                    <div class="col-port">Service 端口</div>
                    <div
                      class="col-nodeport"
                      v-if="formData.type === 'NodePort' || formData.type === 'LoadBalancer'"
                    >
                      宿主机端口
                    </div>
                    <div class="col-action">操作</div>
                  </div>

                  <!-- 端口列表 -->
                  <div v-for="(port, index) in formData.ports" :key="index" class="port-row">
                    <div class="col-name">
                      <ElInput v-model="port.name" placeholder="http" size="small" />
                    </div>
                    <div class="col-protocol">
                      <ElSelect v-model="port.protocol" size="small">
                        <ElOption label="TCP" value="TCP" />
                        <ElOption label="UDP" value="UDP" />
                        <ElOption label="SCTP" value="SCTP" />
                      </ElSelect>
                    </div>
                    <div class="col-port">
                      <ElInput v-model="port.targetPort" placeholder="8080" size="small" />
                    </div>
                    <div class="col-arrow">
                      <ArrowRight :size="16" class="arrow-icon" />
                    </div>
                    <div class="col-port">
                      <ElInputNumber
                        v-model="port.port"
                        :min="1"
                        :max="65535"
                        size="small"
                        controls-position="right"
                      />
                    </div>
                    <div
                      class="col-nodeport"
                      v-if="formData.type === 'NodePort' || formData.type === 'LoadBalancer'"
                    >
                      <ElInputNumber
                        v-model="port.nodePort"
                        :min="30000"
                        :max="32767"
                        size="small"
                        controls-position="right"
                        placeholder="自动分配"
                      />
                    </div>
                    <div class="col-action">
                      <ElButton
                        type="danger"
                        link
                        :icon="Trash2"
                        @click="removePort(index)"
                        :disabled="formData.ports.length === 1"
                      />
                    </div>
                  </div>

                  <!-- 添加端口按钮 -->
                  <div class="add-port-btn">
                    <ElButton type="primary" link :icon="Plus" @click="addPort">
                      添加端口
                    </ElButton>
                  </div>
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- Selector 配置 (非 ExternalName 类型) -->
          <ElCollapseItem
            v-if="formData.type !== 'ExternalName'"
            name="selector"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <Tag :size="16" class="title-icon" />
                <span>Selector 配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <!-- 选择服务和版本 -->
              <ElFormItem label="选择服务" prop="selectedApplicationId">
                <div style="display: flex; gap: 12px; width: 100%">
                  <ElSelect
                    v-model="formData.selectedApplicationId"
                    placeholder="请选择服务"
                    clearable
                    style="flex: 1"
                    @change="handleApplicationChange"
                    :loading="loadingApplications"
                  >
                    <ElOption
                      v-for="app in applications"
                      :key="app.id"
                      :label="app.nameCn"
                      :value="app.id"
                    >
                      <div class="application-option">
                        <Box :size="14" />
                        <span>{{ app.nameCn }}</span>
                      </div>
                    </ElOption>
                  </ElSelect>

                  <ElSelect
                    v-model="formData.selectedVersionId"
                    placeholder="请选择版本"
                    clearable
                    style="flex: 1"
                    :disabled="!formData.selectedApplicationId || loadingVersions"
                    @change="handleVersionChange"
                    :loading="loadingVersions"
                  >
                    <ElOption
                      v-for="version in versions"
                      :key="version.id"
                      :label="`v${version.version}`"
                      :value="version.id"
                    >
                      <div class="version-option">
                        <Tag :size="14" />
                        <span class="version-name">v{{ version.version }}</span>
                      </div>
                    </ElOption>
                  </ElSelect>
                </div>
              </ElFormItem>

              <!-- Selector 预览 -->
              <ElFormItem label="Selector">
                <div class="selector-preview">
                  <div class="preview-header">
                    <span class="preview-title">标签选择器</span>
                    <ElButton
                      type="primary"
                      link
                      size="small"
                      :icon="Plus"
                      @click="showCustomSelectorDialog = true"
                    >
                      添加自定义 Label
                    </ElButton>
                  </div>
                  <div v-if="computedSelector.length > 0" class="preview-tags">
                    <ElTag
                      v-for="(item, index) in computedSelector"
                      :key="index"
                      closable
                      @close="removeSelector(index)"
                      type="info"
                    >
                      {{ item.key }}: {{ item.value }}
                    </ElTag>
                  </div>
                  <div v-else style="color: #909399; font-size: 13px">
                    未设置 Selector，请选择服务版本或添加自定义 Label
                  </div>
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 高级配置 -->
          <ElCollapseItem name="advanced" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <Settings :size="16" class="title-icon" />
                <span>高级配置（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <div class="advanced-content">
                <!-- 会话保持 -->
                <div class="config-group">
                  <div class="group-label">
                    <Clock :size="14" />
                    会话保持
                  </div>
                  <ElFormItem label="会话亲和性" label-width="120px">
                    <ElRadioGroup v-model="formData.sessionAffinity">
                      <ElRadio value="None">无</ElRadio>
                      <ElRadio value="ClientIP">ClientIP</ElRadio>
                    </ElRadioGroup>
                  </ElFormItem>
                  <ElFormItem
                    v-if="formData.sessionAffinity === 'ClientIP'"
                    label="超时时间"
                    label-width="120px"
                  >
                    <ElInputNumber
                      v-model="formData.sessionAffinityTimeout"
                      :min="1"
                      :max="86400"
                      controls-position="right"
                      style="width: 200px"
                    />
                    <span class="form-hint">秒 (默认: 10800)</span>
                  </ElFormItem>
                </div>

                <!-- 流量策略 -->
                <div
                  class="config-group"
                  v-if="formData.type === 'NodePort' || formData.type === 'LoadBalancer'"
                >
                  <div class="group-label">
                    <Network :size="14" />
                    流量策略
                  </div>
                  <ElFormItem label="外部流量策略" label-width="120px">
                    <ElRadioGroup v-model="formData.externalTrafficPolicy">
                      <ElRadio value="Cluster">Cluster</ElRadio>
                      <ElRadio value="Local">Local (保留源 IP)</ElRadio>
                    </ElRadioGroup>
                  </ElFormItem>
                  <ElFormItem label="内部流量策略" label-width="120px">
                    <ElRadioGroup v-model="formData.internalTrafficPolicy">
                      <ElRadio value="Cluster">Cluster</ElRadio>
                      <ElRadio value="Local">Local</ElRadio>
                    </ElRadioGroup>
                  </ElFormItem>
                </div>

                <!-- IP 配置 -->
                <div class="config-group" v-if="formData.type !== 'ExternalName'">
                  <div class="group-label">
                    <Network :size="14" />
                    IP 配置
                  </div>
                  <ElFormItem
                    label="Cluster IP"
                    label-width="120px"
                    v-if="formData.type === 'ClusterIP'"
                  >
                    <ElInput
                      v-model="formData.clusterIP"
                      placeholder="留空自动分配，输入 None 创建 Headless Service"
                      clearable
                      style="width: 350px"
                    />
                    <span class="form-hint">None = Headless Service</span>
                  </ElFormItem>
                  <ElFormItem label="Cluster IPs" label-width="120px">
                    <div style="width: 100%">
                      <ElInput
                        v-for="(ip, index) in formData.clusterIPs"
                        :key="index"
                        v-model="formData.clusterIPs[index]"
                        placeholder="例如: 10.96.0.10"
                        clearable
                        style="width: 350px; margin-bottom: 8px"
                      >
                        <template #append>
                          <ElButton :icon="Trash2" @click="formData.clusterIPs.splice(index, 1)" />
                        </template>
                      </ElInput>
                      <ElButton
                        type="primary"
                        link
                        :icon="Plus"
                        @click="formData.clusterIPs.push('')"
                        size="small"
                      >
                        添加 Cluster IP
                      </ElButton>
                      <div class="form-hint" style="margin-top: 4px"
                        >用于双栈网络，指定多个 Cluster IP</div
                      >
                    </div>
                  </ElFormItem>
                  <ElFormItem label="IP 协议族策略" label-width="120px">
                    <ElSelect
                      v-model="formData.ipFamilyPolicy"
                      clearable
                      placeholder="默认: SingleStack"
                      style="width: 250px"
                    >
                      <ElOption label="SingleStack" value="SingleStack" />
                      <ElOption label="PreferDualStack" value="PreferDualStack" />
                      <ElOption label="RequireDualStack" value="RequireDualStack" />
                    </ElSelect>
                  </ElFormItem>
                  <ElFormItem label="IP 协议族" label-width="120px">
                    <ElSelect
                      v-model="formData.ipFamilies"
                      multiple
                      clearable
                      placeholder="选择 IP 协议族"
                      style="width: 250px"
                    >
                      <ElOption label="IPv4" value="IPv4" />
                      <ElOption label="IPv6" value="IPv6" />
                    </ElSelect>
                    <span class="form-hint">支持 IPv4、IPv6 或双栈</span>
                  </ElFormItem>
                  <ElFormItem label="外部 IP" label-width="120px">
                    <div style="width: 100%">
                      <ElInput
                        v-for="(item, index) in formData.externalIPs"
                        :key="index"
                        v-model="item.ip"
                        placeholder="例如: 192.168.1.100"
                        clearable
                        style="width: 350px; margin-bottom: 8px"
                      >
                        <template #append>
                          <ElButton :icon="Trash2" @click="formData.externalIPs.splice(index, 1)" />
                        </template>
                      </ElInput>
                      <ElButton
                        type="primary"
                        link
                        :icon="Plus"
                        @click="formData.externalIPs.push({ ip: '' })"
                        size="small"
                      >
                        添加外部 IP
                      </ElButton>
                      <div class="form-hint" style="margin-top: 4px"
                        >为 Service 指定外部 IP 地址</div
                      >
                    </div>
                  </ElFormItem>
                </div>

                <!-- LoadBalancer 配置 -->
                <div class="config-group" v-if="formData.type === 'LoadBalancer'">
                  <div class="group-label">
                    <CloudUpload :size="14" />
                    负载均衡器配置
                  </div>
                  <ElFormItem label="负载均衡器类" label-width="120px">
                    <ElInput
                      v-model="formData.loadBalancerClass"
                      placeholder="例如: service.k8s.aws/nlb"
                      clearable
                    />
                  </ElFormItem>
                  <ElFormItem label="负载均衡器 IP" label-width="120px">
                    <ElInput
                      v-model="formData.loadBalancerIP"
                      placeholder="指定负载均衡器 IP"
                      clearable
                    />
                  </ElFormItem>
                  <ElFormItem label="源 IP 限制" label-width="120px">
                    <div style="width: 100%">
                      <ElInput
                        v-for="(item, index) in formData.loadBalancerSourceRanges"
                        :key="index"
                        v-model="item.cidr"
                        placeholder="例如: 10.0.0.0/8"
                        clearable
                        style="width: 350px; margin-bottom: 8px"
                      >
                        <template #append>
                          <ElButton
                            :icon="Trash2"
                            @click="formData.loadBalancerSourceRanges.splice(index, 1)"
                          />
                        </template>
                      </ElInput>
                      <ElButton
                        type="primary"
                        link
                        :icon="Plus"
                        @click="formData.loadBalancerSourceRanges.push({ cidr: '' })"
                        size="small"
                      >
                        添加源 IP CIDR
                      </ElButton>
                      <div class="form-hint" style="margin-top: 4px"
                        >限制可访问 LoadBalancer 的源 IP 范围</div
                      >
                    </div>
                  </ElFormItem>
                  <ElFormItem
                    label="健康检查端口"
                    label-width="120px"
                    v-if="formData.externalTrafficPolicy === 'Local'"
                  >
                    <ElInputNumber
                      v-model="formData.healthCheckNodePort"
                      :min="30000"
                      :max="32767"
                      controls-position="right"
                      placeholder="自动分配"
                      style="width: 200px"
                    />
                    <span class="form-hint">Local 流量策略的健康检查端口</span>
                  </ElFormItem>
                  <ElFormItem label="分配 NodePort" label-width="120px">
                    <ElSwitch v-model="formData.allocateLoadBalancerNodePorts" />
                    <span class="form-hint">是否为 LoadBalancer 分配 NodePort</span>
                  </ElFormItem>
                </div>

                <!-- 其他高级配置 -->
                <div class="config-group" v-if="formData.type !== 'ExternalName'">
                  <div class="group-label">
                    <Settings :size="14" />
                    其他配置
                  </div>
                  <ElFormItem label="发布未就绪地址" label-width="120px">
                    <ElSwitch v-model="formData.publishNotReadyAddresses" />
                    <span class="form-hint">是否将未就绪的 Pod 地址发布到 Endpoints</span>
                  </ElFormItem>
                </div>
              </div>
            </div>
          </ElCollapseItem>

          <!-- 标签和注解（可选） -->
          <ElCollapseItem name="metadata" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <Tag :size="16" class="title-icon" />
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
    <div v-else class="unified-yaml-container">
      <YamlEditor
        :model-value="yamlContent"
        height="600px"
        @update:model-value="handleYamlChange"
      />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="unified-dialog-footer">
        <ElButton @click="handleClose">取消</ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="submitting">
          {{ dialogType === 'add' ? '创建' : '更新' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>

  <!-- 自定义 Selector 对话框 -->
  <ElDialog
    v-model="showCustomSelectorDialog"
    title="添加自定义 Label"
    width="500px"
    :close-on-click-modal="false"
  >
    <ElForm label-width="80px">
      <ElFormItem label="标签键">
        <ElInput v-model="customSelector.key" placeholder="例如: app" clearable />
      </ElFormItem>
      <ElFormItem label="标签值">
        <ElInput v-model="customSelector.value" placeholder="例如: nginx" clearable />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton @click="showCustomSelectorDialog = false">取消</ElButton>
      <ElButton type="primary" @click="handleAddCustomSelector">确定</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch, nextTick } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Loader2 } from 'lucide-vue-next'
  import * as yaml from 'js-yaml'
  import {
    Settings,
    FileCode,
    Network,
    Share2,
    CloudUpload,
    ExternalLink,
    Tag,
    Box,
    Plus,
    Trash2,
    Clock,
    ArrowRight,
    FileText
  } from 'lucide-vue-next'
  import {
    createServiceApi,
    updateServiceApi,
    getServiceDetailApi,
    searchApplicationApi,
    searchVersionApi,
    getVersionLabelsApi,
    type ServiceListItem,
    type ServicePort,
    type ProjectWorkspace
  } from '@/api'
  import YamlEditor from '@/components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../utils/k8s-validator'

  interface Props {
    visible: boolean
    dialogType: 'add' | 'edit'
    serviceData: ServiceListItem | null
    workspace?: ProjectWorkspace
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['close', 'success'])

  // 表单数据
  interface FormData {
    name: string
    type: 'ClusterIP' | 'NodePort' | 'LoadBalancer' | 'ExternalName'
    ports: ServicePort[]
    selectedApplicationId: number | null
    selectedVersionId: number | null
    customSelectors: Array<{ key: string; value: string }>
    labels: Array<{ key: string; value: string }>
    annotations: Array<{ key: string; value: string }>
    sessionAffinity: string
    sessionAffinityTimeout: number
    externalTrafficPolicy: string
    internalTrafficPolicy: string
    ipFamilyPolicy: string
    ipFamilies: string[]
    loadBalancerClass: string
    loadBalancerIP: string
    allocateLoadBalancerNodePorts: boolean
    externalName: string
    // 新增字段
    clusterIP: string // 支持 "None" (Headless), 固定 IP, 或空字符串(自动分配)
    clusterIPs: string[] // 双栈多 IP
    externalIPs: Array<{ ip: string }> // 外部 IP 列表
    loadBalancerSourceRanges: Array<{ cidr: string }> // 源 IP CIDR 限制
    healthCheckNodePort: number | null // 健康检查端口
    publishNotReadyAddresses: boolean // 发布未就绪地址
  }

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const activeNames = ref<string[]>(['basic', 'ports', 'selector', 'advanced', 'metadata'])

  const formData = reactive<FormData>({
    name: '',
    type: 'ClusterIP',
    ports: [
      {
        name: 'http',
        protocol: 'TCP',
        port: 80,
        targetPort: '80',
        appProtocol: ''
      }
    ],
    selectedApplicationId: null,
    selectedVersionId: null,
    customSelectors: [],
    labels: [],
    annotations: [],
    sessionAffinity: 'None',
    sessionAffinityTimeout: 10800,
    externalTrafficPolicy: 'Cluster',
    internalTrafficPolicy: 'Cluster',
    ipFamilyPolicy: '',
    ipFamilies: [],
    loadBalancerClass: '',
    loadBalancerIP: '',
    allocateLoadBalancerNodePorts: true,
    externalName: '',
    // 新增字段默认值
    clusterIP: '', // 空字符串表示自动分配
    clusterIPs: [],
    externalIPs: [],
    loadBalancerSourceRanges: [],
    healthCheckNodePort: null,
    publishNotReadyAddresses: false
  })

  // 表单验证规则
  const formRules = {
    name: createDns1123LabelRules(true, 'Service 名称'),
    type: [{ required: true, message: '请选择 Service 类型', trigger: 'change' }],
    externalName: [
      {
        required: true,
        message: '请输入外部域名',
        trigger: 'blur'
      }
    ]
  }

  const dialogTitle = computed(() => {
    return props.dialogType === 'add' ? '创建 Service' : '编辑 Service'
  })

  // 应用列表
  const applications = ref<any[]>([])
  const loadingApplications = ref(false)

  // 版本列表
  const versions = ref<any[]>([])
  const loadingVersions = ref(false)

  // 版本 labels
  const versionLabels = ref<Array<{ key: string; value: string }>>([])

  // 自定义 Selector
  const showCustomSelectorDialog = ref(false)
  const customSelector = reactive({
    key: '',
    value: ''
  })

  // 计算 Selector (版本 labels + 自定义 labels)
  const computedSelector = computed(() => {
    const selectors: Array<{ key: string; value: string }> = []

    // 添加版本 labels
    selectors.push(...versionLabels.value)

    // 添加自定义 labels
    formData.customSelectors.forEach((s) => {
      if (s.key && s.value) {
        selectors.push(s)
      }
    })

    return selectors
  })

  // 加载应用列表
  const loadApplications = async () => {
    if (!props.workspace) return

    loadingApplications.value = true
    try {
      const response = await searchApplicationApi({
        workspaceId: props.workspace.id
      })
      applications.value = response || []
    } catch (error: any) {
      console.error('加载应用列表失败:', error)
    } finally {
      loadingApplications.value = false
    }
  }

  // 加载版本列表
  const loadVersions = async (applicationId: number) => {
    loadingVersions.value = true
    try {
      const response = await searchVersionApi({
        applicationId
      })
      versions.value = response || []
    } catch (error: any) {
      console.error('加载版本列表失败:', error)
      versions.value = []
    } finally {
      loadingVersions.value = false
    }
  }

  // 加载版本 labels
  const loadVersionLabels = async (versionId: number) => {
    try {
      const labels = await getVersionLabelsApi(versionId)
      versionLabels.value = labels || []
    } catch (error: any) {
      console.error('加载版本标签失败:', error)
      versionLabels.value = []
    }
  }

  // 应用选择变化
  const handleApplicationChange = (applicationId: number | null) => {
    formData.selectedVersionId = null
    versions.value = []
    versionLabels.value = []

    if (applicationId) {
      loadVersions(applicationId)
    }
  }

  // 版本选择变化
  const handleVersionChange = (versionId: number | null) => {
    versionLabels.value = []
    if (versionId) {
      loadVersionLabels(versionId)
    }
  }

  // 添加自定义 Selector
  const handleAddCustomSelector = () => {
    if (!customSelector.key || !customSelector.value) {
      return
    }

    formData.customSelectors.push({
      key: customSelector.key,
      value: customSelector.value
    })

    customSelector.key = ''
    customSelector.value = ''
    showCustomSelectorDialog.value = false
  }

  // 移除 Selector
  const removeSelector = (index: number) => {
    const versionLabelsCount = versionLabels.value.length

    if (index < versionLabelsCount) {
      // 版本标签不允许删除
      return
    } else {
      const customIndex = index - versionLabelsCount
      formData.customSelectors.splice(customIndex, 1)
    }
  }

  // 端口管理
  const addPort = () => {
    formData.ports.push({
      name: '',
      protocol: 'TCP',
      port: 80,
      targetPort: '80',
      appProtocol: ''
    })
  }

  const removePort = (index: number) => {
    if (formData.ports.length > 1) {
      formData.ports.splice(index, 1)
    }
  }

  // Labels 管理
  const addLabel = () => {
    formData.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.labels.splice(index, 1)
  }

  // Annotations 管理
  const addAnnotation = () => {
    formData.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.annotations.splice(index, 1)
  }

  // 类型变化
  const handleTypeChange = () => {
    if (formData.type === 'ExternalName') {
      // ExternalName 类型不需要端口和 Selector
      formData.ports = []
    } else if (formData.ports.length === 0) {
      // 切换回其他类型时添加默认端口
      addPort()
    }
  }

  // 表单转 YAML
  const formToYaml = () => {
    const selector: Record<string, string> = {}
    computedSelector.value.forEach((s) => {
      if (s.key && s.value) {
        selector[s.key] = s.value
      }
    })

    const labels: Record<string, string> = {}
    formData.labels.forEach((l) => {
      if (l.key && l.value) {
        labels[l.key] = l.value
      }
    })

    const annotations: Record<string, string> = {}
    formData.annotations.forEach((a) => {
      if (a.key && a.value) {
        annotations[a.key] = a.value
      }
    })

    const serviceObj: any = {
      apiVersion: 'v1',
      kind: 'Service',
      metadata: {
        name: formData.name,
        namespace: props.workspace?.namespace || 'default',
        labels: labels
      },
      spec: {
        type: formData.type
      }
    }

    // 添加 annotations
    if (Object.keys(annotations).length > 0) {
      serviceObj.metadata.annotations = annotations
    }

    if (formData.type === 'ExternalName') {
      serviceObj.spec.externalName = formData.externalName
    } else {
      serviceObj.spec.selector = selector
      serviceObj.spec.sessionAffinity = formData.sessionAffinity
      serviceObj.spec.ports = formData.ports.map((p) => {
        const port: any = {
          name: p.name,
          protocol: p.protocol,
          port: p.port,
          targetPort: isNaN(Number(p.targetPort)) ? p.targetPort : Number(p.targetPort)
        }
        if (p.appProtocol) {
          port.appProtocol = p.appProtocol
        }
        if (p.nodePort) {
          port.nodePort = p.nodePort
        }
        return port
      })

      // clusterIP 配置
      if (formData.clusterIP) {
        serviceObj.spec.clusterIP = formData.clusterIP
      }

      // clusterIPs 配置（双栈）
      if (formData.clusterIPs.length > 0) {
        const validClusterIPs = formData.clusterIPs.filter((ip) => ip.trim())
        if (validClusterIPs.length > 0) {
          serviceObj.spec.clusterIPs = validClusterIPs
        }
      }

      // externalIPs 配置
      if (formData.externalIPs.length > 0) {
        const validExternalIPs = formData.externalIPs
          .filter((item) => item.ip.trim())
          .map((item) => item.ip)
        if (validExternalIPs.length > 0) {
          serviceObj.spec.externalIPs = validExternalIPs
        }
      }

      // 会话保持配置
      if (formData.sessionAffinity === 'ClientIP') {
        serviceObj.spec.sessionAffinityConfig = {
          clientIP: {
            timeoutSeconds: formData.sessionAffinityTimeout
          }
        }
      }

      // 流量策略
      if (formData.externalTrafficPolicy !== 'Cluster') {
        serviceObj.spec.externalTrafficPolicy = formData.externalTrafficPolicy
      }
      if (formData.internalTrafficPolicy !== 'Cluster') {
        serviceObj.spec.internalTrafficPolicy = formData.internalTrafficPolicy
      }

      // IP 配置
      if (formData.ipFamilyPolicy) {
        serviceObj.spec.ipFamilyPolicy = formData.ipFamilyPolicy
      }
      if (formData.ipFamilies.length > 0) {
        serviceObj.spec.ipFamilies = formData.ipFamilies
      }

      // publishNotReadyAddresses 配置
      if (formData.publishNotReadyAddresses) {
        serviceObj.spec.publishNotReadyAddresses = true
      }

      // LoadBalancer 配置
      if (formData.type === 'LoadBalancer') {
        if (formData.loadBalancerClass) {
          serviceObj.spec.loadBalancerClass = formData.loadBalancerClass
        }
        if (formData.loadBalancerIP) {
          serviceObj.spec.loadBalancerIP = formData.loadBalancerIP
        }
        if (formData.loadBalancerSourceRanges.length > 0) {
          const validRanges = formData.loadBalancerSourceRanges
            .filter((item) => item.cidr.trim())
            .map((item) => item.cidr)
          if (validRanges.length > 0) {
            serviceObj.spec.loadBalancerSourceRanges = validRanges
          }
        }
        if (formData.healthCheckNodePort) {
          serviceObj.spec.healthCheckNodePort = formData.healthCheckNodePort
        }
        if (!formData.allocateLoadBalancerNodePorts) {
          serviceObj.spec.allocateLoadBalancerNodePorts = false
        }
      }
    }

    if (Object.keys(serviceObj.metadata.labels).length === 0) {
      delete serviceObj.metadata.labels
    }

    yamlContent.value = yaml.dump(serviceObj, { indent: 2 })
  }

  // YAML 转表单
  const yamlToForm = () => {
    try {
      const obj: any = yaml.load(yamlContent.value)

      if (!obj || typeof obj !== 'object') {
        ElMessage.error('YAML 格式不正确')
        return
      }

      if (obj.kind !== 'Service') {
        ElMessage.error('YAML 类型必须是 Service')
        return
      }

      if (props.workspace) {
        if (!obj.metadata) {
          obj.metadata = {}
        }
        obj.metadata.namespace = props.workspace.namespace
      }

      formData.name = obj.metadata?.name || ''
      formData.type = obj.spec?.type || 'ClusterIP'
      formData.externalName = obj.spec?.externalName || ''
      formData.sessionAffinity = obj.spec?.sessionAffinity || 'None'

      if (obj.spec?.sessionAffinityConfig?.clientIP?.timeoutSeconds) {
        formData.sessionAffinityTimeout = obj.spec.sessionAffinityConfig.clientIP.timeoutSeconds
      }

      formData.externalTrafficPolicy = obj.spec?.externalTrafficPolicy || 'Cluster'
      formData.internalTrafficPolicy = obj.spec?.internalTrafficPolicy || 'Cluster'
      formData.ipFamilyPolicy = obj.spec?.ipFamilyPolicy || ''
      formData.ipFamilies = obj.spec?.ipFamilies || []
      formData.loadBalancerClass = obj.spec?.loadBalancerClass || ''
      formData.loadBalancerIP = obj.spec?.loadBalancerIP || ''
      formData.allocateLoadBalancerNodePorts = obj.spec?.allocateLoadBalancerNodePorts !== false

      // 新增字段解析
      formData.clusterIP = obj.spec?.clusterIP || ''
      formData.clusterIPs = obj.spec?.clusterIPs || []
      formData.externalIPs = (obj.spec?.externalIPs || []).map((ip: string) => ({ ip }))
      formData.loadBalancerSourceRanges = (obj.spec?.loadBalancerSourceRanges || []).map(
        (cidr: string) => ({ cidr })
      )
      formData.healthCheckNodePort = obj.spec?.healthCheckNodePort || null
      formData.publishNotReadyAddresses = obj.spec?.publishNotReadyAddresses || false

      formData.ports =
        obj.spec?.ports?.map((p: any) => ({
          name: p.name || '',
          protocol: p.protocol || 'TCP',
          port: p.port || 80,
          targetPort: String(p.targetPort || '80'),
          nodePort: p.nodePort,
          appProtocol: p.appProtocol || ''
        })) || []

      // Labels
      const labels = obj.metadata?.labels || {}
      formData.labels = Object.entries(labels).map(([key, value]) => ({
        key,
        value: String(value)
      }))

      // Annotations
      const annotations = obj.metadata?.annotations || {}
      formData.annotations = Object.entries(annotations).map(([key, value]) => ({
        key,
        value: String(value)
      }))

      // Selector - 解析为 customSelectors
      if (obj.spec?.selector) {
        formData.customSelectors = Object.entries(obj.spec.selector).map(([key, value]) => ({
          key,
          value: String(value)
        }))
      } else {
        formData.customSelectors = []
      }

      // 清空应用和版本选择（因为从 YAML 解析的 selector 是自定义的）
      formData.selectedApplicationId = null
      formData.selectedVersionId = null
      versionLabels.value = []

      // 同步回 YAML 以确保格式正确
      formToYaml()
    } catch (error) {
      console.error('YAML 解析失败:', error)
      ElMessage.error('YAML 解析失败，请检查格式')
    }
  }

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  // 加载编辑数据
  const loadEditData = async () => {
    if (!props.serviceData || !props.workspace) return

    dataLoading.value = true
    try {
      const detail = await getServiceDetailApi({
        workloadId: props.workspace.id,
        name: props.serviceData.name
      })

      formData.name = detail.name
      formData.type = detail.type as any
      formData.sessionAffinity = detail.sessionAffinity

      formData.ports =
        detail.ports?.map((p) => ({
          name: p.name,
          protocol: p.protocol,
          port: p.port,
          targetPort: String(p.targetPort),
          nodePort: p.nodePort,
          appProtocol: (p as any).appProtocol || ''
        })) || []

      // Labels
      if (detail.labels) {
        formData.labels = Object.entries(detail.labels).map(([key, value]) => ({
          key,
          value: String(value)
        }))
      }

      // Annotations
      if (detail.annotations) {
        formData.annotations = Object.entries(detail.annotations).map(([key, value]) => ({
          key,
          value: String(value)
        }))
      }

      // Selector - 转换为自定义 selectors
      if (detail.selector) {
        formData.customSelectors = Object.entries(detail.selector).map(([key, value]) => ({
          key,
          value: String(value)
        }))
      }

      // 高级配置
      formData.sessionAffinityTimeout =
        detail.sessionAffinityConfig?.clientIP?.timeoutSeconds || 10800
      formData.externalTrafficPolicy = detail.externalTrafficPolicy || 'Cluster'
      formData.internalTrafficPolicy = detail.internalTrafficPolicy || 'Cluster'
      formData.ipFamilyPolicy = detail.ipFamilyPolicy || ''
      formData.ipFamilies = detail.ipFamilies || []

      // 新增字段加载
      formData.clusterIP = detail.clusterIP || ''
      formData.clusterIPs = (detail as any).clusterIPs || []
      formData.externalIPs = ((detail as any).externalIPs || []).map((ip: string) => ({ ip }))
      formData.publishNotReadyAddresses = (detail as any).publishNotReadyAddresses || false

      // LoadBalancer 配置
      if (detail.type === 'LoadBalancer') {
        formData.loadBalancerClass = detail.loadBalancerClass || ''
        formData.loadBalancerIP = detail.loadBalancerIP || ''
        formData.loadBalancerSourceRanges = ((detail as any).loadBalancerSourceRanges || []).map(
          (cidr: string) => ({ cidr })
        )
        formData.healthCheckNodePort = (detail as any).healthCheckNodePort || null
        formData.allocateLoadBalancerNodePorts = detail.allocateLoadBalancerNodePorts !== false
      }

      // ExternalName 配置
      if (detail.type === 'ExternalName') {
        formData.externalName = detail.externalName || ''
      }

      formToYaml()
    } catch (error: any) {
      console.error('加载 Service 详情失败:', error)
    } finally {
      dataLoading.value = false
    }
  }

  // 提交
  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    if (editMode.value === 'yaml') {
      yamlToForm()
    }

    try {
      await formRef.value?.validate()
    } catch {
      return
    }

    if (formData.type !== 'ExternalName' && formData.ports.length === 0) {
      return
    }

    if (formData.type !== 'ExternalName' && computedSelector.value.length === 0) {
      return
    }

    submitting.value = true
    try {
      const selector: Record<string, string> = {}
      computedSelector.value.forEach((s) => {
        if (s.key && s.value) {
          selector[s.key] = s.value
        }
      })

      const labels: Record<string, string> = {}
      formData.labels.forEach((l) => {
        if (l.key && l.value) {
          labels[l.key] = l.value
        }
      })

      const annotations: Record<string, string> = {}
      formData.annotations.forEach((a) => {
        if (a.key && a.value) {
          annotations[a.key] = a.value
        }
      })

      const requestData: any = {
        workloadId: props.workspace.id,
        name: formData.name,
        type: formData.type,
        labels
      }

      // 添加 annotations（如果有）
      if (Object.keys(annotations).length > 0) {
        requestData.annotations = annotations
      }

      if (formData.type === 'ExternalName') {
        requestData.externalName = formData.externalName
      } else {
        requestData.ports = formData.ports.map((p) => {
          const port: any = {
            name: p.name,
            protocol: p.protocol,
            port: p.port,
            targetPort: String(p.targetPort)
          }
          if (p.appProtocol) {
            port.appProtocol = p.appProtocol
          }
          if (p.nodePort) {
            port.nodePort = p.nodePort
          }
          return port
        })
        requestData.selector = selector
        requestData.sessionAffinity = formData.sessionAffinity

        // clusterIP 配置
        if (formData.clusterIP) {
          requestData.clusterIP = formData.clusterIP
        }

        // clusterIPs 配置（双栈）
        if (formData.clusterIPs.length > 0) {
          const validClusterIPs = formData.clusterIPs.filter((ip) => ip.trim())
          if (validClusterIPs.length > 0) {
            requestData.clusterIPs = validClusterIPs
          }
        }

        // externalIPs 配置
        if (formData.externalIPs.length > 0) {
          const validExternalIPs = formData.externalIPs
            .filter((item) => item.ip.trim())
            .map((item) => item.ip)
          if (validExternalIPs.length > 0) {
            requestData.externalIPs = validExternalIPs
          }
        }

        // 会话保持配置
        if (formData.sessionAffinity === 'ClientIP') {
          requestData.sessionAffinityConfig = {
            clientIP: {
              timeoutSeconds: formData.sessionAffinityTimeout
            }
          }
        }

        // 流量策略
        if (formData.externalTrafficPolicy !== 'Cluster') {
          requestData.externalTrafficPolicy = formData.externalTrafficPolicy
        }
        if (formData.internalTrafficPolicy !== 'Cluster') {
          requestData.internalTrafficPolicy = formData.internalTrafficPolicy
        }

        // IP 配置
        if (formData.ipFamilyPolicy) {
          requestData.ipFamilyPolicy = formData.ipFamilyPolicy
        }
        if (formData.ipFamilies.length > 0) {
          requestData.ipFamilies = formData.ipFamilies
        }

        // publishNotReadyAddresses 配置
        if (formData.publishNotReadyAddresses) {
          requestData.publishNotReadyAddresses = true
        }

        // LoadBalancer 配置
        if (formData.type === 'LoadBalancer') {
          if (formData.loadBalancerClass) {
            requestData.loadBalancerClass = formData.loadBalancerClass
          }
          if (formData.loadBalancerIP) {
            requestData.loadBalancerIP = formData.loadBalancerIP
          }
          if (formData.loadBalancerSourceRanges.length > 0) {
            const validRanges = formData.loadBalancerSourceRanges
              .filter((item) => item.cidr.trim())
              .map((item) => item.cidr)
            if (validRanges.length > 0) {
              requestData.loadBalancerSourceRanges = validRanges
            }
          }
          if (formData.healthCheckNodePort) {
            requestData.healthCheckNodePort = formData.healthCheckNodePort
          }
          if (!formData.allocateLoadBalancerNodePorts) {
            requestData.allocateLoadBalancerNodePorts = false
          }
        }
      }

      if (props.dialogType === 'add') {
        await createServiceApi(requestData)
        ElMessage.success('创建成功')
      } else {
        await updateServiceApi(requestData)
        ElMessage.success('更新成功')
      }

      emit('success')
    } finally {
      submitting.value = false
    }
  }

  const handleClose = () => {
    emit('close')
  }

  // 初始化
  watch(
    () => props.visible,
    async (visible) => {
      if (visible) {
        await loadApplications()

        if (props.dialogType === 'edit' && props.serviceData) {
          await loadEditData()
        } else {
          // 重置表单
          Object.assign(formData, {
            name: '',
            type: 'ClusterIP',
            ports: [
              {
                name: 'http',
                protocol: 'TCP',
                port: 80,
                targetPort: '80',
                appProtocol: ''
              }
            ],
            selectedApplicationId: null,
            selectedVersionId: null,
            customSelectors: [],
            labels: [],
            annotations: [],
            sessionAffinity: 'None',
            sessionAffinityTimeout: 10800,
            externalTrafficPolicy: 'Cluster',
            internalTrafficPolicy: 'Cluster',
            ipFamilyPolicy: '',
            ipFamilies: [],
            loadBalancerClass: '',
            loadBalancerIP: '',
            allocateLoadBalancerNodePorts: true,
            externalName: '',
            // 重置新增字段
            clusterIP: '',
            clusterIPs: [],
            externalIPs: [],
            loadBalancerSourceRanges: [],
            healthCheckNodePort: null,
            publishNotReadyAddresses: false
          })
          versionLabels.value = []
          editMode.value = 'form'
          formToYaml()
        }

        await nextTick()
        formRef.value?.clearValidate()
      }
    },
    { immediate: true }
  )

  // 监听编辑模式切换
  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return

    if (newMode === 'yaml' && oldMode === 'form') {
      // 从表单切换到 YAML，同步表单数据到 YAML
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      // 从 YAML 切换到表单，同步 YAML 数据到表单
      yamlToForm()
    }
  })

  // 监听表单变化，同步到 YAML
  watch(
    () => [
      formData.name,
      formData.type,
      formData.ports,
      formData.labels,
      computedSelector.value,
      formData.sessionAffinity,
      formData.externalName
    ],
    () => {
      if (editMode.value === 'form') {
        formToYaml()
      }
    },
    { deep: true }
  )
</script>

<style lang="scss" scoped>
  .service-form-container {
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;

      &:hover {
        background: #c0c4cc;
      }
    }
  }

  .service-form {
    :deep(.form-collapse) {
      border: none;
      background: transparent;

      .el-collapse-item {
        background: #fff;
        border: 1px solid #dcdfe6;
        border-radius: 4px;
        margin-bottom: 12px;
        overflow: hidden;

        &.is-active {
          border-color: #409eff;
        }

        &:last-child {
          margin-bottom: 0;
        }
      }

      .el-collapse-item__header {
        height: 48px;
        line-height: 48px;
        background: #f5f7fa;
        border-bottom: 1px solid #e4e7ed;
        padding: 0 16px;
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
        padding: 0;
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

    .collapse-content {
      padding: 20px 24px;
    }
  }

  .application-option,
  .version-option {
    display: flex;
    align-items: center;
    gap: 8px;

    .version-name {
      color: #606266;
      font-size: 13px;
    }
  }

  .selector-preview {
    width: 100%;
    padding: 12px;
    background: white;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    min-height: 60px;

    .preview-header {
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;

      .preview-title {
        font-size: 13px;
        color: #606266;
        font-weight: 500;
      }
    }

    .preview-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
    }
  }

  .ports-table {
    width: 100%;
    border: 1px solid #e4e7ed;
    border-radius: 4px;
    background: white;
    overflow: hidden;

    .ports-header {
      display: flex;
      align-items: center;
      padding: 10px 12px;
      background: #f5f7fa;
      border-bottom: 1px solid #e4e7ed;
      font-size: 12px;
      font-weight: 600;
      color: #606266;
      gap: 8px;

      .col-name {
        flex: 0 0 110px;
      }
      .col-protocol {
        flex: 0 0 90px;
      }
      .col-app-protocol {
        flex: 0 0 100px;
      }
      .col-port {
        flex: 0 0 110px;
      }
      .col-arrow {
        flex: 0 0 28px;
        text-align: center;
      }
      .col-nodeport {
        flex: 0 0 110px;
      }
      .col-action {
        flex: 0 0 60px;
        text-align: center;
      }
    }

    .port-row {
      display: flex;
      align-items: center;
      padding: 10px 12px;
      gap: 8px;
      border-bottom: 1px solid #f0f0f0;

      &:last-child {
        border-bottom: none;
      }

      .col-name {
        flex: 0 0 110px;
      }
      .col-protocol {
        flex: 0 0 90px;
      }
      .col-app-protocol {
        flex: 0 0 100px;
      }
      .col-port {
        flex: 0 0 110px;
      }
      .col-arrow {
        flex: 0 0 28px;
        display: flex;
        justify-content: center;
        align-items: center;

        .arrow-icon {
          color: #409eff;
        }
      }
      .col-nodeport {
        flex: 0 0 110px;
      }
      .col-action {
        flex: 0 0 60px;
        display: flex;
        justify-content: center;
      }
    }

    .add-port-btn {
      padding: 10px 12px;
      text-align: center;
      border-top: 1px dashed #e4e7ed;
      background: #fafbfc;
    }
  }

  .advanced-content {
    max-height: 360px;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;
    }
  }

  .config-group {
    margin-bottom: 20px;
    padding-bottom: 16px;
    border-bottom: 1px dashed #dcdfe6;

    &:last-child {
      margin-bottom: 0;
      padding-bottom: 0;
      border-bottom: none;
    }

    .group-label {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 13px;
      font-weight: 600;
      color: #606266;
      margin-bottom: 12px;
      padding-left: 10px;
      border-left: 3px solid #dcdfe6;
    }
  }

  .form-hint {
    font-size: 12px;
    color: #909399;
    margin-left: 10px;
  }

  :deep(.el-radio-button) {
    .el-radio-button__inner {
      display: inline-flex;
      align-items: center;
      gap: 4px;
    }
  }
</style>

<style lang="scss">
  .service-dialog.el-dialog .el-dialog__body {
    padding: 0 !important;
    margin: 0 !important;
  }
</style>
