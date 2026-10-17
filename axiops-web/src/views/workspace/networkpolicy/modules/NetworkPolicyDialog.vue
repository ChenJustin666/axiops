<template>
  <ElDialog
    :model-value="visible"
    :title="dialogType === 'add' ? '创建 NetworkPolicy' : '编辑 NetworkPolicy'"
    width="1200px"
    top="3vh"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
    @closed="handleClosed"
  >
    <!-- 编辑模式切换 -->
    <div class="mode-switch">
      <!-- 模板选择（仅新增模式） -->
      <ElDropdown v-if="dialogType === 'add'" trigger="click" @command="handleSelectTemplate">
        <ElButton type="primary" plain :icon="FileText"> 使用模板 </ElButton>
        <template #dropdown>
          <ElDropdownMenu>
            <ElDropdownItem v-for="tpl in templates" :key="tpl.id" :command="tpl">
              <div class="template-item">
                <span class="template-name">{{ tpl.name }}</span>
                <span class="template-desc">{{ tpl.description }}</span>
              </div>
            </ElDropdownItem>
          </ElDropdownMenu>
        </template>
      </ElDropdown>

      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <div class="radio-content">
            <FormInput :size="13" />
            <span>表单编辑</span>
          </div>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <div class="radio-content">
            <Code :size="13" />
            <span>YAML 编辑</span>
          </div>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态 -->
    <div v-if="dataLoading" class="loading-state">
      <ElSkeleton :rows="8" animated />
      <div class="loading-text">正在加载 NetworkPolicy 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="network-policy-form">
      <ElCollapse v-model="activeCollapse" class="form-collapse">
        <!-- 基础信息 -->
        <ElCollapseItem name="basic" class="collapse-section-item">
          <template #title>
            <div class="collapse-title">
              <Database :size="16" />
              <span>基础信息</span>
            </div>
          </template>

          <ElForm ref="formRef" :model="formData" :rules="formRules" label-width="120px">
            <ElFormItem label="名称" prop="name">
              <ElInput
                v-model="formData.name"
                placeholder="请输入 NetworkPolicy 名称"
                :disabled="dialogType === 'edit'"
                maxlength="63"
                show-word-limit
              />
            </ElFormItem>

            <ElFormItem label="命名空间">
              <ElInput :value="workspace?.namespace" disabled />
            </ElFormItem>

            <ElFormItem label="策略类型" prop="policyTypes">
              <ElCheckboxGroup v-model="formData.policyTypes">
                <ElCheckbox value="Ingress">入站 (Ingress)</ElCheckbox>
                <ElCheckbox value="Egress">出站 (Egress)</ElCheckbox>
              </ElCheckboxGroup>
            </ElFormItem>
          </ElForm>
        </ElCollapseItem>

        <!-- Pod 选择器 -->
        <ElCollapseItem name="podSelector" class="collapse-section-item">
          <template #title>
            <div class="collapse-title">
              <Target :size="16" />
              <span>Pod 选择器</span>
            </div>
          </template>

          <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 12px">
            <template #title>留空表示应用于命名空间内的所有 Pod</template>
          </ElAlert>

          <div class="label-selector">
            <div class="key-value-list">
              <div
                v-for="(item, index) in formData.podSelector"
                :key="'pod-' + index"
                class="key-value-item"
              >
                <ElInput v-model="item.key" placeholder="键" class="key-input" maxlength="63" />
                <span class="separator">=</span>
                <ElInput v-model="item.value" placeholder="值" class="value-input" maxlength="63" />
                <ElButton type="danger" :icon="Trash2" circle @click="removePodSelector(index)" />
              </div>
            </div>

            <div class="label-actions">
              <ElButton size="small" :icon="Plus" @click="addPodSelector">添加标签</ElButton>
              <ElButton type="primary" size="small" plain @click="openPodLabelSelectorDialog">
                选择标签
              </ElButton>
            </div>
          </div>
        </ElCollapseItem>

        <!-- 入站规则 -->
        <ElCollapseItem
          v-if="formData.policyTypes.includes('Ingress')"
          name="ingressRules"
          class="collapse-section-item"
        >
          <template #title>
            <div class="collapse-title">
              <ArrowDown :size="16" />
              <span>入站规则</span>
            </div>
          </template>

          <div v-if="formData.ingressRules.length === 0" class="empty-state warning">
            <ElIcon><XCircle /></ElIcon>
            <span>没有入站规则将拒绝所有入站流量</span>
          </div>

          <div
            v-for="(rule, index) in formData.ingressRules"
            :key="'ingress-' + index"
            class="rule-card"
          >
            <div class="rule-header">
              <div class="rule-title">
                <ElTag type="success" size="small">规则 {{ index + 1 }}</ElTag>
              </div>
              <ElButton
                type="danger"
                :icon="Trash2"
                circle
                size="small"
                @click="removeIngressRule(index)"
              />
            </div>

            <div class="rule-body">
              <!-- 端口配置 -->
              <div class="rule-section">
                <div class="section-label">
                  <span>端口配置</span>
                  <ElCheckbox v-model="rule.hasPorts" @change="handlePortsChange(rule)">
                    指定端口
                  </ElCheckbox>
                </div>
                <div v-if="rule.hasPorts" class="ports-list">
                  <div
                    v-for="(port, pIndex) in rule.ports"
                    :key="'ingress-port-' + index + '-' + pIndex"
                    class="port-item"
                  >
                    <ElSelect v-model="port.protocol" placeholder="协议" style="width: 100px">
                      <ElOption label="TCP" value="TCP" />
                      <ElOption label="UDP" value="UDP" />
                      <ElOption label="SCTP" value="SCTP" />
                    </ElSelect>
                    <ElInput
                      v-model.number="port.port"
                      placeholder="端口"
                      type="number"
                      style="flex: 1"
                    />
                    <ElButton
                      type="danger"
                      :icon="Trash2"
                      circle
                      size="small"
                      @click="removePort(rule, index, pIndex)"
                    />
                  </div>
                  <ElButton size="small" :icon="Plus" @click="addPort(rule)">添加端口</ElButton>
                </div>
              </div>

              <!-- 来源配置 -->
              <div class="rule-section">
                <div class="section-label">
                  <span>来源配置</span>
                  <ElCheckbox v-model="rule.hasFrom">限制来源</ElCheckbox>
                </div>
                <div v-if="rule.hasFrom" class="peers-list">
                  <PeerConfigItem
                    v-for="(peer, pIndex) in rule.from"
                    :key="'ingress-peer-' + index + '-' + pIndex"
                    :peer="peer"
                    :all-namespaces="allNamespaces"
                    :current-namespace="workspace?.namespace"
                    :cluster="cluster"
                    @remove="removePeer(rule.from, pIndex)"
                    @update:peer="(p) => updatePeer(rule.from, pIndex, p)"
                  />
                  <ElButton size="small" :icon="Plus" @click="addPeer(rule.from)"
                    >添加来源</ElButton
                  >
                </div>
              </div>
            </div>
          </div>

          <ElButton
            type="primary"
            :icon="Plus"
            plain
            @click="addIngressRule"
            style="width: 100%; margin-top: 12px"
          >
            添加入站规则
          </ElButton>
        </ElCollapseItem>

        <!-- 出站规则 -->
        <ElCollapseItem
          v-if="formData.policyTypes.includes('Egress')"
          name="egressRules"
          class="collapse-section-item"
        >
          <template #title>
            <div class="collapse-title">
              <ArrowUp :size="16" />
              <span>出站规则</span>
            </div>
          </template>

          <div v-if="formData.egressRules.length === 0" class="empty-state warning">
            <ElIcon><XCircle /></ElIcon>
            <span>没有出站规则将拒绝所有出站流量</span>
          </div>

          <div
            v-for="(rule, index) in formData.egressRules"
            :key="'egress-' + index"
            class="rule-card"
          >
            <div class="rule-header">
              <div class="rule-title">
                <ElTag type="warning" size="small">规则 {{ index + 1 }}</ElTag>
              </div>
              <ElButton
                type="danger"
                :icon="Trash2"
                circle
                size="small"
                @click="removeEgressRule(index)"
              />
            </div>

            <div class="rule-body">
              <!-- 端口配置 -->
              <div class="rule-section">
                <div class="section-label">
                  <span>端口配置</span>
                  <ElCheckbox v-model="rule.hasPorts" @change="handlePortsChange(rule)">
                    指定端口
                  </ElCheckbox>
                </div>
                <div v-if="rule.hasPorts" class="ports-list">
                  <div
                    v-for="(port, pIndex) in rule.ports"
                    :key="'egress-port-' + index + '-' + pIndex"
                    class="port-item"
                  >
                    <ElSelect v-model="port.protocol" placeholder="协议" style="width: 100px">
                      <ElOption label="TCP" value="TCP" />
                      <ElOption label="UDP" value="UDP" />
                      <ElOption label="SCTP" value="SCTP" />
                    </ElSelect>
                    <ElInput
                      v-model.number="port.port"
                      placeholder="端口"
                      type="number"
                      style="flex: 1"
                    />
                    <ElButton
                      type="danger"
                      :icon="Trash2"
                      circle
                      size="small"
                      @click="removePort(rule, index, pIndex)"
                    />
                  </div>
                  <ElButton size="small" :icon="Plus" @click="addPort(rule)">添加端口</ElButton>
                </div>
              </div>

              <!-- 目标配置 -->
              <div class="rule-section">
                <div class="section-label">
                  <span>目标配置</span>
                  <ElCheckbox v-model="rule.hasTo">限制目标</ElCheckbox>
                </div>
                <div v-if="rule.hasTo" class="peers-list">
                  <PeerConfigItem
                    v-for="(peer, pIndex) in rule.to"
                    :key="'egress-peer-' + index + '-' + pIndex"
                    :peer="peer"
                    :all-namespaces="allNamespaces"
                    :current-namespace="workspace?.namespace"
                    :cluster="cluster"
                    @remove="removePeer(rule.to, pIndex)"
                    @update:peer="(p) => updatePeer(rule.to, pIndex, p)"
                  />
                  <ElButton size="small" :icon="Plus" @click="addPeer(rule.to)">添加目标</ElButton>
                </div>
              </div>
            </div>
          </div>

          <ElButton
            type="primary"
            :icon="Plus"
            plain
            @click="addEgressRule"
            style="width: 100%; margin-top: 12px"
          >
            添加出站规则
          </ElButton>
        </ElCollapseItem>

        <!-- 标签 -->
        <ElCollapseItem name="labels" class="collapse-section-item">
          <template #title>
            <div class="collapse-title">
              <Tag :size="16" />
              <span>标签（Labels）</span>
            </div>
          </template>

          <div class="label-selector">
            <div class="key-value-list">
              <div
                v-for="(item, index) in formData.labels"
                :key="'label-' + index"
                class="key-value-item"
              >
                <ElInput v-model="item.key" placeholder="键" class="key-input" maxlength="63" />
                <span class="separator">=</span>
                <ElInput v-model="item.value" placeholder="值" class="value-input" maxlength="63" />
                <ElButton type="danger" :icon="Trash2" circle @click="removeLabel(index)" />
              </div>
            </div>

            <div class="label-actions">
              <ElButton size="small" :icon="Plus" @click="addLabel">添加标签</ElButton>
            </div>
          </div>
        </ElCollapseItem>

        <!-- 注解 -->
        <ElCollapseItem name="annotations" class="collapse-section-item">
          <template #title>
            <div class="collapse-title">
              <Tag :size="16" />
              <span>注解（Annotations）</span>
            </div>
          </template>

          <div class="label-selector">
            <div class="key-value-list">
              <div
                v-for="(item, index) in formData.annotations"
                :key="'annotation-' + index"
                class="key-value-item"
              >
                <ElInput v-model="item.key" placeholder="键" class="key-input" maxlength="63" />
                <span class="separator">=</span>
                <ElInput v-model="item.value" placeholder="值" class="value-input" maxlength="63" />
                <ElButton type="danger" :icon="Trash2" circle @click="removeAnnotation(index)" />
              </div>
            </div>

            <div class="label-actions">
              <ElButton size="small" :icon="Plus" @click="addAnnotation">添加注解</ElButton>
            </div>
          </div>
        </ElCollapseItem>
      </ElCollapse>
    </div>

    <!-- YAML 模式 -->
    <div v-else-if="editMode === 'yaml'" class="yaml-content">
      <ArtYamlEditor v-model="yamlContent" height="650px" @change="handleYamlChange" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleCancel">取 消</ElButton>
        <ElButton type="primary" :loading="submitting" @click="handleSubmit">
          {{ dialogType === 'add' ? '创 建' : '更 新' }}
        </ElButton>
      </div>
    </template>

    <!-- Pod 标签选择弹窗 -->
    <PodLabelSelectorDialog
      v-model="podLabelSelectorVisible"
      :cluster="cluster"
      :namespace="workspace?.namespace || ''"
      @confirm="handlePodLabelsSelected"
    />
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch, getCurrentInstance } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    Database,
    Tag,
    Target,
    ArrowDown,
    ArrowUp,
    FormInput,
    Code,
    Plus,
    Trash2,
    XCircle,
    FileText
  } from 'lucide-vue-next'
  import {
    createNetworkPolicyApi,
    updateNetworkPolicyApi,
    getNetworkPolicyYamlApi,
    getNetworkPolicyDetailApi,
    type NetworkPolicyListItem,
    type NetworkPolicyDetail
  } from '../../../../api/workload/networkpolicy'
  import type { ProjectWorkspace, ProjectCluster } from '../../../../api/manager/project'
  import ArtYamlEditor from '../../../../components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../core/utils/k8s-validator'
  import {
    parseYaml,
    stringifyYaml,
    objectToKeyValueArray,
    keyValueArrayToObject,
    mergeYamlPreserveUnknown
  } from '../../core/utils/yaml-merger'
  import { getNetworkPolicyTemplates } from '../utils/templates'
  import PeerConfigItem from './PeerConfigItem.vue'
  import PodLabelSelectorDialog from './PodLabelSelectorDialog.vue'
  import { getClusterNamespaceListApi } from '../../../../api/manager/cluster'

  // ============================================================================
  // 类型定义
  // ============================================================================

  interface KeyValue {
    key: string
    value: string
  }

  interface NetworkPolicyPort {
    protocol?: string
    port?: number
  }

  // Pod 选择器配置
  interface PodSelectorConfig {
    enabled: boolean
    mode: 'current' | 'specific'
    namespace?: string
    labels: KeyValue[]
  }

  // Namespace 选择器配置
  interface NamespaceSelectorConfig {
    enabled: boolean
    namespace?: string
    labels: KeyValue[]
  }

  // IP 块配置
  interface IPBlockConfig {
    cidr?: string
    except?: { value: string }[]
  }

  // 对端配置（重新设计）
  interface NetworkPolicyPeer {
    // 选择器类型：selector 或 ipblock
    selectorType: 'selector' | 'ipblock'

    // Pod 选择器配置（与 namespaceSelector 可同时存在）
    podSelector?: PodSelectorConfig

    // Namespace 选择器配置（与 podSelector 可同时存在）
    namespaceSelector?: NamespaceSelectorConfig

    // IP 块配置（与选择器互斥）
    ipBlock?: IPBlockConfig
  }

  interface NetworkPolicyRule {
    hasPorts: boolean
    ports: NetworkPolicyPort[]
    hasFrom?: boolean
    hasTo?: boolean
    from?: NetworkPolicyPeer[]
    to?: NetworkPolicyPeer[]
  }

  interface Props {
    modelValue: boolean
    dialogType: 'add' | 'edit'
    networkPolicyData?: NetworkPolicyListItem
    workspace: ProjectWorkspace | null
    cluster: ProjectCluster | null
  }

  // ============================================================================
  // Props & Emits
  // ============================================================================

  const props = defineProps<Props>()
  const emit = defineEmits(['update:modelValue', 'success'])

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const templates = getNetworkPolicyTemplates()

  // ============================================================================
  // 状态
  // ============================================================================

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const rawSourceYaml = ref('')

  // ElCollapse 默认展开项
  const activeCollapse = ref(['basic', 'podSelector'])

  // Pod 标签选择弹窗
  const podLabelSelectorVisible = ref(false)
  const podLabelsTarget = ref<'main' | 'peer' | 'namespace'>('main')
  const currentPeerRef = ref<NetworkPolicyPeer | null>(null)

  // 所有命名空间列表
  const allNamespaces = ref<string[]>([])

  const formData = ref({
    name: '',
    policyTypes: ['Ingress'] as string[],
    podSelector: [] as KeyValue[],
    ingressRules: [] as NetworkPolicyRule[],
    egressRules: [] as NetworkPolicyRule[],
    labels: [] as KeyValue[],
    annotations: [] as KeyValue[]
  })

  const formRules = {
    name: createDns1123LabelRules(true, 'NetworkPolicy 名称')
  }

  // ============================================================================
  // 数据加载
  // ============================================================================

  const loadAllNamespaces = async () => {
    if (!props.cluster) return
    try {
      const list = await getClusterNamespaceListApi(props.cluster.clusterUuid)
      allNamespaces.value = list
    } catch (error) {
      console.error('加载命名空间列表失败:', error)
    }
  }

  watch(visible, async (val) => {
    if (val) {
      await loadAllNamespaces()
    }

    if (
      val &&
      props.dialogType === 'edit' &&
      props.networkPolicyData &&
      props.workspace &&
      props.cluster
    ) {
      dataLoading.value = true
      try {
        const detail = await getNetworkPolicyDetailApi({
          clusterUuid: props.cluster.clusterUuid,
          namespace: props.workspace.namespace,
          name: props.networkPolicyData.name
        })

        formData.value = {
          name: detail.name,
          policyTypes: detail.policyTypes || ['Ingress'],
          podSelector: objectToKeyValueArray(detail.podSelector),
          ingressRules: parseRulesFromDetail(detail.ingressRules),
          egressRules: parseRulesFromDetail(detail.egressRules),
          labels: objectToKeyValueArray(detail.labels),
          annotations: objectToKeyValueArray(detail.annotations || {})
        }

        const yamlStr = await getNetworkPolicyYamlApi({
          clusterUuid: props.cluster.clusterUuid,
          namespace: props.workspace.namespace,
          name: props.networkPolicyData.name
        })
        rawSourceYaml.value = yamlStr

        formToYaml()
      } catch (error) {
        console.error('加载 NetworkPolicy 详情失败:', error)
        visible.value = false
      } finally {
        dataLoading.value = false
      }
    } else if (val && props.dialogType === 'add') {
      rawSourceYaml.value = ''
      formData.value = {
        name: '',
        policyTypes: ['Ingress'],
        podSelector: [],
        ingressRules: [],
        egressRules: [],
        labels: [],
        annotations: []
      }
      formToYaml()
    }
  })

  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return
    if (newMode === 'yaml' && oldMode === 'form') {
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      yamlToForm()
    }
  })

  watch(
    () => [
      formData.value.name,
      formData.value.policyTypes,
      formData.value.podSelector,
      formData.value.ingressRules,
      formData.value.egressRules,
      formData.value.labels,
      formData.value.annotations
    ],
    () => {
      if (editMode.value === 'form') {
        formToYaml()
      }
    },
    { deep: true }
  )

  // ============================================================================
  // 解析函数
  // ============================================================================

  function parseRulesFromDetail(rules: any[] | undefined): NetworkPolicyRule[] {
    if (!rules || rules.length === 0) return []

    return rules.map((rule) => {
      const hasPorts = rule.ports && rule.ports.length > 0
      const hasFrom = rule.from && rule.from.length > 0
      const hasTo = rule.to && rule.to.length > 0

      const result: NetworkPolicyRule = {
        hasPorts,
        ports: hasPorts
          ? rule.ports.map((p: any) => ({
              protocol: p.protocol || 'TCP',
              port: p.port
            }))
          : []
      }

      if (hasFrom) {
        result.hasFrom = true
        result.from = rule.from.map((peer: any) => parsePeerToForm(peer))
      }

      if (hasTo) {
        result.hasTo = true
        result.to = rule.to.map((peer: any) => parsePeerToForm(peer))
      }

      return result
    })
  }

  function parsePeerToForm(peer: any): NetworkPolicyPeer {
    // IPBlock 模式
    if (peer.ipBlock) {
      return {
        selectorType: 'ipblock',
        ipBlock: {
          cidr: peer.ipBlock.cidr,
          except: peer.ipBlock.except?.map((e: string) => ({ value: e })) || []
        }
      }
    }

    // 获取 Pod 选择器
    const hasPodSelector =
      peer.podSelector && Object.keys(peer.podSelector).some((k) => k !== 'matchExpressions')
    // 获取 Namespace 选择器
    const hasNamespaceSelector =
      peer.namespaceSelector &&
      Object.keys(peer.namespaceSelector).some((k) => k !== 'matchExpressions')

    // 选择器模式
    const result: NetworkPolicyPeer = {
      selectorType: 'selector'
    }

    // 解析 Pod 选择器
    if (hasPodSelector) {
      const labels = peer.podSelector?.matchLabels || peer.podSelector || {}
      result.podSelector = {
        enabled: true,
        mode: 'current',
        labels: objectToKeyValueArray(labels)
      }
    } else {
      result.podSelector = {
        enabled: false,
        mode: 'current',
        labels: []
      }
    }

    // 解析 Namespace 选择器
    if (hasNamespaceSelector) {
      const labels = peer.namespaceSelector?.matchLabels || peer.namespaceSelector || {}
      result.namespaceSelector = {
        enabled: true,
        labels: objectToKeyValueArray(labels)
      }

      // 尝试从标签中提取命名空间
      const nsLabel = labels['kubernetes.io/metadata.name']
      if (nsLabel && typeof nsLabel === 'string') {
        result.namespaceSelector.namespace = nsLabel
        // 如果 Pod 选择器也启用且指定了命名空间，则设置 mode
        if (result.podSelector?.enabled && nsLabel !== props.workspace?.namespace) {
          result.podSelector.mode = 'specific'
          result.podSelector.namespace = nsLabel
        }
      }
    } else {
      result.namespaceSelector = {
        enabled: false,
        labels: []
      }
    }

    return result
  }

  // ============================================================================
  // YAML 转换
  // ============================================================================

  const formToYaml = () => {
    const podSelectorLabels = keyValueArrayToObject(formData.value.podSelector)

    // 构建 podSelector，必须使用 matchLabels 包裹标签
    let podSelector: any
    if (Object.keys(podSelectorLabels).length > 0) {
      podSelector = { matchLabels: podSelectorLabels }
    } else {
      // 空选择器，选择所有 Pod
      podSelector = {}
    }

    const ingressRules = formData.value.ingressRules
      .filter((r) => formData.value.policyTypes.includes('Ingress'))
      .map((rule) => {
        const result: any = {}
        if (rule.hasPorts && rule.ports.length > 0) {
          result.ports = rule.ports
            .filter((p) => p.port || p.protocol)
            .map((p) => ({
              protocol: p.protocol || 'TCP',
              port: p.port
            }))
        }
        if (rule.hasFrom && rule.from) {
          result.from = rule.from
            .map((p) => parsePeerFromForm(p))
            .filter((r) => Object.keys(r).length > 0)
        }
        return Object.keys(result).length > 0 ? result : null
      })
      .filter((r) => r !== null)

    const egressRules = formData.value.egressRules
      .filter((r) => formData.value.policyTypes.includes('Egress'))
      .map((rule) => {
        const result: any = {}
        if (rule.hasPorts && rule.ports.length > 0) {
          result.ports = rule.ports
            .filter((p) => p.port || p.protocol)
            .map((p) => ({
              protocol: p.protocol || 'TCP',
              port: p.port
            }))
        }
        if (rule.hasTo && rule.to) {
          result.to = rule.to
            .map((p) => parsePeerFromForm(p))
            .filter((r) => Object.keys(r).length > 0)
        }
        return Object.keys(result).length > 0 ? result : null
      })
      .filter((r) => r !== null)

    const labelsObj = keyValueArrayToObject(formData.value.labels)
    const annotationsObj = keyValueArrayToObject(formData.value.annotations)

    const npObj: any = {
      apiVersion: 'networking.k8s.io/v1',
      kind: 'NetworkPolicy',
      metadata: {
        name: formData.value.name || 'networkpolicy',
        namespace: props.workspace?.namespace || 'default'
      },
      spec: {
        podSelector,
        policyTypes: formData.value.policyTypes
      }
    }

    if (labelsObj && Object.keys(labelsObj).length > 0) {
      npObj.metadata.labels = labelsObj
    }

    if (annotationsObj && Object.keys(annotationsObj).length > 0) {
      npObj.metadata.annotations = annotationsObj
    }

    if (ingressRules.length > 0) {
      npObj.spec.ingress = ingressRules
    }

    if (egressRules.length > 0) {
      npObj.spec.egress = egressRules
    }

    // 如果是编辑模式且有原始 YAML，则保留未知字段
    if (props.dialogType === 'edit' && rawSourceYaml.value) {
      try {
        const merged = mergeYamlPreserveUnknown(rawSourceYaml.value, npObj)
        yamlContent.value = stringifyYaml(merged)
      } catch {
        yamlContent.value = stringifyYaml(npObj)
      }
    } else {
      yamlContent.value = stringifyYaml(npObj)
    }
  }

  function parsePeerFromForm(peer: NetworkPolicyPeer): any {
    // IPBlock 模式
    if (peer.selectorType === 'ipblock' && peer.ipBlock?.cidr) {
      const result: any = { ipBlock: { cidr: peer.ipBlock.cidr } }
      if (peer.ipBlock.except && peer.ipBlock.except.length > 0) {
        const exceptValues = peer.ipBlock.except.filter((e) => e.value).map((e) => e.value)
        if (exceptValues.length > 0) {
          result.ipBlock.except = exceptValues
        }
      }
      return result
    }

    const result: any = {}

    // Namespace 选择器
    if (peer.namespaceSelector?.enabled) {
      const labelsObj = keyValueArrayToObject(peer.namespaceSelector.labels.filter((l) => l.key))
      if (Object.keys(labelsObj).length > 0) {
        result.namespaceSelector = { matchLabels: labelsObj }
      } else {
        // 如果没有标签，使用空选择器
        result.namespaceSelector = {}
      }
    }

    // Pod 选择器（可以与 Namespace 选择器组合）
    if (peer.podSelector?.enabled) {
      const labelsObj = keyValueArrayToObject(peer.podSelector.labels.filter((l) => l.key))
      if (Object.keys(labelsObj).length > 0) {
        result.podSelector = { matchLabels: labelsObj }
      } else {
        // 如果没有标签，使用空选择器
        result.podSelector = {}
      }
    }

    return result
  }

  const yamlToForm = () => {
    try {
      const obj = parseYaml<any>(yamlContent.value)

      if (!obj || typeof obj !== 'object') {
        ElMessage.error('YAML 格式不正确')
        return
      }

      if (obj.kind !== 'NetworkPolicy') {
        ElMessage.error('YAML 类型必须是 NetworkPolicy')
        return
      }

      const spec = obj.spec || {}

      formData.value = {
        name: obj.metadata?.name || '',
        policyTypes: spec.policyTypes || ['Ingress'],
        podSelector: objectToKeyValueArray(spec.podSelector?.matchLabels || spec.podSelector),
        ingressRules: parseRulesFromYaml(spec.ingress),
        egressRules: parseRulesFromYaml(spec.egress),
        labels: objectToKeyValueArray(obj.metadata?.labels),
        annotations: objectToKeyValueArray(obj.metadata?.annotations)
      }

      formToYaml()
    } catch (error) {
      console.error('YAML 解析失败:', error)
      ElMessage.error('YAML 解析失败，请检查格式')
    }
  }

  function parseRulesFromYaml(rules: any[] | undefined): NetworkPolicyRule[] {
    if (!rules || rules.length === 0) return []

    return rules.map((rule) => {
      const hasPorts = rule.ports && rule.ports.length > 0
      const hasFrom = rule.from && rule.from.length > 0
      const hasTo = rule.to && rule.to.length > 0

      const result: NetworkPolicyRule = {
        hasPorts,
        ports: hasPorts
          ? rule.ports.map((p: any) => ({
              protocol: p.protocol || 'TCP',
              port: p.port
            }))
          : []
      }

      if (hasFrom) {
        result.hasFrom = true
        result.from = rule.from.map((peer: any) => parsePeerToForm(peer))
      }

      if (hasTo) {
        result.hasTo = true
        result.to = rule.to.map((peer: any) => parsePeerToForm(peer))
      }

      return result
    })
  }

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  // ============================================================================
  // Pod 选择器操作
  // ============================================================================

  const addPodSelector = () => {
    formData.value.podSelector.push({ key: '', value: '' })
  }

  const removePodSelector = (index: number) => {
    formData.value.podSelector.splice(index, 1)
  }

  const openPodLabelSelectorDialog = () => {
    podLabelsTarget.value = 'main'
    currentPeerRef.value = null
    podLabelSelectorVisible.value = true
  }

  const handlePodLabelsSelected = (labels: KeyValue[]) => {
    if (podLabelsTarget.value === 'main') {
      // 主 Pod 选择器
      for (const label of labels) {
        const exists = formData.value.podSelector.some((l) => l.key === label.key)
        if (!exists) {
          formData.value.podSelector.push({ ...label })
        }
      }
    } else if (podLabelsTarget.value === 'peer' && currentPeerRef.value?.podSelector) {
      // Peer 的 Pod 选择器
      for (const label of labels) {
        const exists = currentPeerRef.value.podSelector.labels.some((l) => l.key === label.key)
        if (!exists) {
          currentPeerRef.value.podSelector.labels.push({ ...label })
        }
      }
    } else if (podLabelsTarget.value === 'namespace' && currentPeerRef.value?.namespaceSelector) {
      // Namespace 选择器
      for (const label of labels) {
        const exists = currentPeerRef.value.namespaceSelector.labels.some(
          (l) => l.key === label.key
        )
        if (!exists) {
          currentPeerRef.value.namespaceSelector.labels.push({ ...label })
        }
      }
    }

    podLabelSelectorVisible.value = false
  }

  // ============================================================================
  // 规则操作
  // ============================================================================

  const addIngressRule = () => {
    formData.value.ingressRules.push({
      hasPorts: false,
      hasFrom: false,
      ports: [],
      from: []
    })
  }

  const removeIngressRule = (index: number) => {
    formData.value.ingressRules.splice(index, 1)
  }

  const addEgressRule = () => {
    formData.value.egressRules.push({
      hasPorts: false,
      hasTo: false,
      ports: [],
      to: []
    })
  }

  const removeEgressRule = (index: number) => {
    formData.value.egressRules.splice(index, 1)
  }

  // 端口操作
  const addPort = (rule: NetworkPolicyRule) => {
    rule.ports.push({ protocol: 'TCP', port: undefined })
  }

  const removePort = (rule: NetworkPolicyRule, ruleIndex: number, portIndex: number) => {
    rule.ports.splice(portIndex, 1)
    if (rule.ports.length === 0) {
      rule.hasPorts = false
    }
  }

  const handlePortsChange = (rule: NetworkPolicyRule) => {
    if (rule.hasPorts && rule.ports.length === 0) {
      rule.ports.push({ protocol: 'TCP', port: undefined })
    }
  }

  // 对端操作
  const addPeer = (peers: NetworkPolicyPeer[]) => {
    peers.push({
      selectorType: 'selector',
      podSelector: {
        enabled: false,
        mode: 'current',
        labels: []
      },
      namespaceSelector: {
        enabled: false,
        labels: []
      },
      ipBlock: {
        except: []
      }
    })
  }

  const removePeer = (peers: NetworkPolicyPeer[], index: number) => {
    peers.splice(index, 1)
  }

  const updatePeer = (peers: NetworkPolicyPeer[], index: number, newPeer: NetworkPolicyPeer) => {
    // 使用 Object.assign 避免创建新引用
    // 只有在内容真正变化时才更新
    const currentPeer = peers[index]
    if (JSON.stringify(currentPeer) !== JSON.stringify(newPeer)) {
      Object.assign(currentPeer, newPeer)
    }
  }

  const handleOpenPodSelector = (selector: PodSelectorConfig, namespace?: string) => {
    // 如果指定了命名空间，使用该命名空间；否则使用当前工作空间命名空间
    const targetNamespace = namespace || props.workspace?.namespace
    podLabelsTarget.value = 'peer'

    // 打开 Pod 标签选择弹窗，传递命名空间信息
    currentPeerRef.value = {
      selectorType: 'selector',
      podSelector: selector,
      namespaceSelector: { enabled: false, labels: [] },
      ipBlock: { except: [] }
    }

    // 保存目标命名空间供弹窗使用
    if (targetNamespace) {
      currentPeerRef.value.podSelector!.namespace = targetNamespace
      currentPeerRef.value.podSelector!.mode =
        namespace === props.workspace?.namespace ? 'current' : 'specific'
    }

    podLabelSelectorVisible.value = true
  }

  // 暴露给 PeerConfigItem 组件使用
  const openPodLabelSelectorForPeer = (peer: NetworkPolicyPeer, type: 'pod' | 'namespace') => {
    currentPeerRef.value = peer
    podLabelsTarget.value = type === 'pod' ? 'peer' : 'namespace'
    podLabelSelectorVisible.value = true
  }

  // ============================================================================
  // 标签/注解操作
  // ============================================================================

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

  // ============================================================================
  // 模板选择
  // ============================================================================

  const handleSelectTemplate = (tpl: any) => {
    const replacedYaml = tpl.yaml
      .replace(/name: .+/g, (match: string) => {
        if (match.includes('metadata:')) {
          return `name: ${formData.value.name || 'networkpolicy'}`
        }
        return match
      })
      .replace(/namespace: default/g, `namespace: ${props.workspace?.namespace || 'default'}`)
    yamlContent.value = replacedYaml

    if (editMode.value === 'form') {
      yamlToForm()
    }
  }

  // ============================================================================
  // 提交处理
  // ============================================================================

  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    if (editMode.value === 'yaml') {
      await submitYaml()
    } else {
      try {
        await formRef.value?.validate()
      } catch {
        return
      }

      if (formData.value.policyTypes.length === 0) {
        ElMessage.warning('请至少选择一种策略类型')
        return
      }

      formToYaml()
      await submitYaml()
    }
  }

  const submitYaml = async () => {
    if (!props.workspace) return

    const currentClusterUuid = (props.workspace as any).clusterUuid || ''

    try {
      const obj = parseYaml<any>(yamlContent.value)
      if (!obj) {
        ElMessage.error('YAML 格式不正确')
        return
      }

      submitting.value = true

      const requestData = {
        clusterUuid: currentClusterUuid,
        namespace: props.workspace.namespace,
        yamlStr: yamlContent.value
      }

      if (props.dialogType === 'add') {
        await createNetworkPolicyApi(requestData)
        ElMessage.success('创建成功')
      } else {
        await updateNetworkPolicyApi(requestData)
        ElMessage.success('更新成功')
      }

      emit('success')
      visible.value = false
    } catch (error: any) {
      // 错误由全局拦截器处理
    } finally {
      submitting.value = false
    }
  }

  const handleClose = () => {
    visible.value = false
  }

  const handleCancel = () => {
    visible.value = false
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    formData.value = {
      name: '',
      policyTypes: ['Ingress'],
      podSelector: [],
      ingressRules: [],
      egressRules: [],
      labels: [],
      annotations: []
    }
    yamlContent.value = ''
    rawSourceYaml.value = ''
    editMode.value = 'form'
    activeCollapse.value = ['basic', 'podSelector']
  }

  // 暴露给子组件使用
  defineExpose({
    openPodLabelSelectorForPeer
  })
</script>

<style lang="scss" scoped>
  @use '../styles/network-policy-form.scss' as *;

  .mode-switch {
    margin-bottom: 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .radio-content {
      display: flex;
      align-items: center;
      gap: 4px;
    }

    .template-item {
      display: flex;
      flex-direction: column;
      gap: 2px;

      .template-name {
        font-size: 13px;
        font-weight: 500;
      }

      .template-desc {
        font-size: 11px;
        color: #909399;
      }
    }
  }

  .loading-state {
    padding: 40px;
    text-align: center;

    .loading-text {
      margin-top: 12px;
      color: #909399;
      font-size: 13px;
    }
  }

  .yaml-content {
    min-height: 600px;
  }

  .dialog-footer {
    display: flex;
    gap: 12px;
    justify-content: flex-end;
  }

  .network-policy-form {
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

    &::-webkit-scrollbar-track {
      background: #f5f7fa;
    }

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
        padding: 20px 24px;
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

        :deep(svg) {
          color: #409eff;
          margin-right: 8px;
        }
      }
    }

    .label-selector {
      .key-value-list {
        margin-bottom: 12px;

        .key-value-item {
          display: flex;
          align-items: center;
          gap: 8px;
          margin-bottom: 8px;
          padding: 10px 12px;
          background: #f5f7fa;
          border-radius: 4px;
          border: 1px solid #e4e7ed;
          transition: border-color 0.2s;

          &:hover {
            border-color: #409eff;
          }

          .key-input {
            flex: 0 0 180px;
          }

          .value-input {
            flex: 1;
          }

          .separator {
            flex-shrink: 0;
            color: #909399;
            font-weight: 500;
            font-size: 14px;
          }
        }
      }

      .label-actions {
        display: flex;
        gap: 8px;
      }
    }

    .rule-card {
      margin-bottom: 16px;
      padding: 16px;
      background: #fff;
      border-radius: 4px;
      border: 1px solid #e4e7ed;
      transition: border-color 0.2s;

      &:hover {
        border-color: #c0c4cc;
      }

      &:last-child {
        margin-bottom: 0;
      }

      .rule-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px dashed #e4e7ed;

        .rule-title {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 14px;
          font-weight: 600;
          color: #303133;
        }
      }

      .rule-body {
        .rule-section {
          margin-bottom: 16px;
          padding: 12px;
          background: #fafafa;
          border-radius: 4px;
          border: 1px solid #ebeef5;

          &:last-child {
            margin-bottom: 0;
          }

          .section-label {
            display: flex;
            align-items: center;
            justify-content: space-between;
            margin-bottom: 10px;
            font-size: 13px;
            font-weight: 500;
            color: #606266;
          }

          .ports-list,
          .peers-list {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-top: 10px;
          }

          .port-item {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 10px;
            background: #fff;
            border-radius: 4px;
            border: 1px solid #e4e7ed;
            transition: border-color 0.2s;

            &:hover {
              border-color: #409eff;
            }
          }
        }
      }
    }

    .empty-state {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 24px;
      color: #909399;
      font-size: 13px;
      background: #f5f7fa;
      border-radius: 4px;
      border: 1px dashed #dcdfe6;

      &.warning {
        color: #e6a23c;
        background: #fdf6ec;
        border-color: #f5dab1;
      }
    }
  }
</style>
