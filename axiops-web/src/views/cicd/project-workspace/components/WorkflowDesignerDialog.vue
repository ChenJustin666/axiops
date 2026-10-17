<template>
  <ElDialog
    v-model="dialogVisible"
    :title="`${workflow?.name || '工作流'} · DAG 编排`"
    width="min(1180px, 94vw)"
    top="3vh"
    destroy-on-close
  >
    <div v-loading="loading" class="designer-shell">
      <header class="designer-toolbar">
        <div class="workflow-fields">
          <ElInput v-model="workflowName" :disabled="!editable" placeholder="工作流名称" />
          <ElInput v-model="workflowDescription" :disabled="!editable" placeholder="工作流描述" />
          <ElSelect
            v-model="targetEnvironmentId"
            :disabled="!editable || !!workflowDetail?.target_environment_id"
            :placeholder="hasDeployNode ? '选择项目已配置的目标环境' : '可选：使用项目环境'"
          >
            <ElOption
              v-for="environment in targetEnvironments"
              :key="environment.id"
              :value="environment.id"
              :label="`${environment.name} · ${environment.namespace || environment.host_group_ref}${environment.is_production ? ' · 生产' : ''}`"
              :disabled="
                environment.status !== 'active' || environment.binding_status !== 'validated'
              "
            />
          </ElSelect>
          <small v-if="workflowDetail?.target_environment_id" class="environment-readonly-hint"
            >目标环境来自项目环境配置，执行时只读核对，不在执行阶段切换。</small
          >
        </div>
        <div v-if="editable" class="node-actions">
          <ElButton :icon="Hammer" @click="addNode('build')">构建</ElButton>
          <ElButton :icon="TestTube2" @click="addNode('test')">测试</ElButton>
          <ElButton :icon="ShieldCheck" @click="addNode('approval')">审批</ElButton>
          <ElButton :icon="BellRing" @click="addNode('notify')">通知</ElButton>
          <ElButton :icon="Rocket" @click="addNode(deployNodeType)">部署</ElButton>
        </div>
      </header>

      <section v-if="serviceOrder.length > 1" class="service-order" aria-label="服务部署顺序">
        <div class="service-order-heading">
          <strong>服务部署顺序</strong>
          <small>{{
            serviceSequenceActive
              ? '按顺序提交部署，各自检查结果'
              : legacyServiceSequence
                ? '旧顺序会等待就绪；点击转换后保存'
                : '当前部署并行'
          }}</small>
          <ElButton
            v-if="editable && !serviceSequenceActive"
            link
            type="primary"
            :disabled="saving"
            @click="applyServiceOrder(serviceOrder)"
            >{{ legacyServiceSequence ? '转换为提交顺序' : '按此顺序提交' }}</ElButton
          >
        </div>
        <ol class="service-order-list">
          <li v-for="(serviceId, index) in serviceOrder" :key="serviceId">
            <span class="order-number">{{ index + 1 }}</span>
            <span class="order-name">{{ serviceName(serviceId) }}</span>
            <ElButton
              v-if="editable"
              :icon="ArrowLeft"
              text
              circle
              :disabled="index === 0 || saving"
              :title="`提前部署 ${serviceName(serviceId)}`"
              @click="moveService(serviceId, -1)"
            />
            <ElButton
              v-if="editable"
              :icon="ArrowRight"
              text
              circle
              :disabled="index === serviceOrder.length - 1 || saving"
              :title="`延后部署 ${serviceName(serviceId)}`"
              @click="moveService(serviceId, 1)"
            />
          </li>
        </ol>
      </section>

      <div class="designer-body">
        <section class="flow-panel">
          <VueFlow
            v-model:nodes="flowNodes"
            :edges="flowEdges"
            :nodes-draggable="editable"
            :nodes-connectable="editable"
            :elements-selectable="true"
            :is-valid-connection="isValidConnection"
            :default-edge-options="defaultEdgeOptions"
            fit-view-on-init
            class="workflow-flow"
            @connect="handleConnect"
            @node-click="handleNodeClick"
            @edge-click="handleEdgeClick"
            @pane-click="clearSelection"
          >
            <Controls :show-interactive="false" />
            <MiniMap pannable zoomable />
            <template #node-default="{ data, selected }">
              <div
                class="workflow-node"
                :class="[`is-${data.workflow.type}`, { 'is-selected': selected }]"
              >
                <Handle type="target" :position="Position.Left" :connectable="editable" />
                <span class="node-icon">
                  <component :is="nodeMeta(data.workflow.type).icon" :size="17" />
                </span>
                <span>
                  <strong>{{ data.workflow.name }}</strong>
                  <small>{{ nodeMeta(data.workflow.type).label }}</small>
                </span>
                <Handle type="source" :position="Position.Right" :connectable="editable" />
              </div>
            </template>
          </VueFlow>
        </section>

        <aside class="inspector-panel">
          <template v-if="selectedWorkflowNode">
            <div class="inspector-heading">
              <span>
                <strong>节点配置</strong>
                <small>{{ selectedWorkflowNode.key }}</small>
              </span>
              <ElButton
                v-if="editable"
                :icon="Trash2"
                text
                circle
                title="删除节点"
                @click="removeSelectedNode"
              />
            </div>
            <ElForm label-position="top">
              <ElFormItem label="节点名称">
                <ElInput v-model="selectedWorkflowNode.name" :disabled="!editable" />
              </ElFormItem>
              <ElFormItem label="节点类型">
                <ElInput :model-value="nodeMeta(selectedWorkflowNode.type).label" disabled />
              </ElFormItem>
              <ElFormItem
                v-if="
                  selectedWorkflowNode.type === 'build' ||
                  selectedWorkflowNode.type === 'test' ||
                  isDeployNode(selectedWorkflowNode.type)
                "
                label="目标服务"
                required
              >
                <ElSelect
                  v-model="selectedServiceId"
                  :disabled="!editable"
                  filterable
                  style="width: 100%"
                  @change="syncSelectedService"
                >
                  <ElOption
                    v-for="service in services"
                    :key="service.id"
                    :label="service.name"
                    :value="service.id"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem v-if="selectedWorkflowNode.type === 'build'" label="Git 分支">
                <ElInput v-model="selectedBranch" :disabled="!editable" />
              </ElFormItem>
              <ElFormItem
                v-if="isDeployNode(selectedWorkflowNode.type)"
                label="服务 Revision"
                required
              >
                <ElSelect v-model="selectedRevisionId" :disabled="!editable" style="width: 100%">
                  <ElOption
                    v-for="revision in selectedServiceRevisions"
                    :key="revision.id"
                    :label="`Revision ${revision.revision_number}`"
                    :value="revision.id"
                  />
                </ElSelect>
              </ElFormItem>
              <ElFormItem v-if="selectedWorkflowNode.type === 'helm_deploy'" label="Release 名称">
                <ElInput v-model="selectedReleaseName" :disabled="!editable" />
              </ElFormItem>
              <template v-if="selectedWorkflowNode.type === 'test'">
                <ElFormItem label="测试执行镜像" required>
                  <ElInput
                    v-model="selectedTestImage"
                    :disabled="!editable"
                    placeholder="例如 maven:3.9-eclipse-temurin-17"
                  />
                </ElFormItem>
                <ElFormItem label="测试命令" required>
                  <ElInput
                    v-model="selectedTestCommand"
                    :disabled="!editable"
                    type="textarea"
                    :rows="3"
                    placeholder="go test ./..."
                  />
                </ElFormItem>
                <ElFormItem label="工作目录">
                  <ElInput
                    v-model="selectedWorkingDir"
                    :disabled="!editable"
                    placeholder="仓库内相对路径；留空为仓库根目录"
                  />
                </ElFormItem>
              </template>
              <ElFormItem v-if="selectedWorkflowNode.type === 'notify'" label="通知说明">
                <ElInput
                  v-model="selectedNotifyMessage"
                  :disabled="!editable"
                  placeholder="发布结果通知"
                />
              </ElFormItem>
              <div v-if="selectedWorkflowNode.type !== 'approval'" class="form-grid">
                <ElFormItem label="超时（秒）">
                  <ElInputNumber
                    v-model="selectedWorkflowNode.timeout_seconds"
                    :disabled="!editable"
                    :min="0"
                    :max="86400"
                    controls-position="right"
                    style="width: 100%"
                  />
                </ElFormItem>
                <ElFormItem v-if="selectedWorkflowNode.type === 'build'" label="失败自动重试次数">
                  <ElInputNumber
                    v-model="selectedWorkflowNode.retry_limit"
                    :disabled="!editable"
                    :min="0"
                    :max="10"
                    controls-position="right"
                    style="width: 100%"
                  />
                </ElFormItem>
              </div>
              <p v-if="selectedWorkflowNode.type === 'build'" class="inspector-note"
                >重试固定原 Commit 和构建配置，并保留历史日志；未取得完整 Commit
                或达到节点超时后停止重试。</p
              >
            </ElForm>
          </template>
          <template v-else-if="selectedEdgeId">
            <div class="inspector-heading">
              <span
                ><strong>依赖关系</strong><small>{{ selectedEdgeLabel }}</small></span
              >
            </div>
            <p class="inspector-note">目标节点会在来源节点成功后进入可执行状态。</p>
            <ElButton v-if="editable" :icon="Unlink" @click="removeSelectedEdge">
              删除依赖
            </ElButton>
          </template>
          <ElEmpty v-else :image-size="60" description="选择节点或依赖关系" />
        </aside>
      </div>

      <div v-if="validationIssues.length" class="validation-panel">
        <strong>编排校验未通过</strong>
        <span v-for="issue in validationIssues" :key="`${issue.field}-${issue.code}`">
          {{ issue.message }}
        </span>
      </div>
    </div>

    <template #footer>
      <ElButton @click="dialogVisible = false">关闭</ElButton>
      <ElButton v-if="editable" type="primary" :loading="saving" @click="saveWorkflow">
        校验并保存
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, markRaw, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    Handle,
    MarkerType,
    Position,
    VueFlow,
    type Connection,
    type Edge,
    type Node
  } from '@vue-flow/core'
  import { Controls } from '@vue-flow/controls'
  import { MiniMap } from '@vue-flow/minimap'
  import {
    ArrowLeft,
    ArrowRight,
    BellRing,
    CircleCheckBig,
    Hammer,
    Rocket,
    ShieldCheck,
    TestTube2,
    Trash2,
    Unlink
  } from 'lucide-vue-next'
  import {
    hasLegacyWorkflowDeploymentSequence,
    hasWorkflowDeploymentSequence,
    withWorkflowDeploymentOrder,
    workflowDeploymentOrder
  } from '@/utils/workflow-service-order'
  import {
    getWorkflowApi,
    listEnvironmentsApi,
    listServiceRevisionsApi,
    updateWorkflowApi,
    validateWorkflowApi,
    type AxiOpsProject,
    type AxiOpsEnvironment,
    type AxiOpsService,
    type AxiOpsServiceRevision,
    type AxiOpsWorkflow,
    type WorkflowNode,
    type WorkflowNodeType,
    type WorkflowValidationIssue
  } from '@/api/axiops'
  import '@vue-flow/core/dist/style.css'
  import '@vue-flow/core/dist/theme-default.css'
  import '@vue-flow/controls/dist/style.css'
  import '@vue-flow/minimap/dist/style.css'

  interface FlowNodeData {
    workflow: WorkflowNode
  }

  const props = defineProps<{
    visible: boolean
    workflow: AxiOpsWorkflow | null
    project: AxiOpsProject | null
    services: AxiOpsService[]
    editable: boolean
  }>()
  const emit = defineEmits<{ 'update:visible': [value: boolean]; saved: [] }>()

  const dialogVisible = computed({
    get: () => props.visible,
    set: (value) => emit('update:visible', value)
  })
  const loading = ref(false)
  const saving = ref(false)
  const workflowDetail = ref<AxiOpsWorkflow | null>(null)
  const workflowName = ref('')
  const workflowDescription = ref('')
  const targetEnvironmentId = ref(0)
  const targetEnvironments = ref<AxiOpsEnvironment[]>([])
  const flowNodes = ref<Node<FlowNodeData>[]>([])
  const flowEdges = ref<Edge[]>([])
  const selectedNodeId = ref('')
  const selectedEdgeId = ref('')
  const selectedServiceRevisions = ref<AxiOpsServiceRevision[]>([])
  const validationIssues = ref<WorkflowValidationIssue[]>([])
  let nodeSequence = 0
  let loadGeneration = 0

  const defaultEdgeOptions = {
    type: 'smoothstep',
    markerEnd: MarkerType.ArrowClosed
  }
  const deployNodeType = computed<WorkflowNodeType>(() => {
    if (props.project?.delivery_type === 'helm') return 'helm_deploy'
    if (props.project?.delivery_type === 'host') return 'host_deploy'
    return 'yaml_deploy'
  })
  const selectedFlowNode = computed(() =>
    flowNodes.value.find((item) => item.id === selectedNodeId.value)
  )
  const selectedWorkflowNode = computed(() => selectedFlowNode.value?.data?.workflow)
  const selectedEdgeLabel = computed(() => {
    const edge = flowEdges.value.find((item) => item.id === selectedEdgeId.value)
    if (!edge) return ''
    const source = flowNodes.value.find((item) => item.id === edge.source)?.data?.workflow.name
    const target = flowNodes.value.find((item) => item.id === edge.target)?.data?.workflow.name
    return `${source || edge.source} -> ${target || edge.target}`
  })

  const updateSelectedConfig = (key: string, value: unknown) => {
    if (!selectedWorkflowNode.value) return
    selectedWorkflowNode.value.config = {
      ...(selectedWorkflowNode.value.config || {}),
      [key]: value
    }
  }
  const selectedServiceId = computed({
    get: () => Number(selectedWorkflowNode.value?.config?.service_id || 0),
    set: (value: number) => updateSelectedConfig('service_id', value)
  })
  const selectedRevisionId = computed({
    get: () => Number(selectedWorkflowNode.value?.config?.service_revision_id || 0),
    set: (value: number) => updateSelectedConfig('service_revision_id', value)
  })
  const selectedBranch = computed({
    get: () => String(selectedWorkflowNode.value?.config?.branch || ''),
    set: (value: string) => updateSelectedConfig('branch', value)
  })
  const selectedReleaseName = computed({
    get: () => String(selectedWorkflowNode.value?.config?.release_name || ''),
    set: (value: string) => updateSelectedConfig('release_name', value)
  })
  const selectedTestCommand = computed({
    get: () => String(selectedWorkflowNode.value?.config?.command || ''),
    set: (value: string) => updateSelectedConfig('command', value)
  })
  const selectedTestImage = computed({
    get: () => String(selectedWorkflowNode.value?.config?.image || ''),
    set: (value: string) => updateSelectedConfig('image', value)
  })
  const selectedWorkingDir = computed({
    get: () => String(selectedWorkflowNode.value?.config?.working_dir || ''),
    set: (value: string) => updateSelectedConfig('working_dir', value)
  })
  const selectedNotifyMessage = computed({
    get: () => String(selectedWorkflowNode.value?.config?.message || ''),
    set: (value: string) => updateSelectedConfig('message', value)
  })

  const nodeMeta = (type: WorkflowNodeType) => {
    if (type === 'build') return { label: '构建', icon: markRaw(Hammer) }
    if (type === 'test') return { label: '自动化测试', icon: markRaw(TestTube2) }
    if (type === 'approval') return { label: '人工审批', icon: markRaw(ShieldCheck) }
    if (type === 'notify') return { label: '通知', icon: markRaw(BellRing) }
    if (isDeployNode(type)) return { label: '部署', icon: markRaw(Rocket) }
    return { label: type, icon: markRaw(CircleCheckBig) }
  }
  const isDeployNode = (type: WorkflowNodeType) =>
    ['yaml_deploy', 'helm_deploy', 'host_deploy'].includes(type)
  const hasDeployNode = computed(() =>
    flowNodes.value.some((item) => item.data && isDeployNode(item.data.workflow.type))
  )

  const createFlowNode = (workflowNode: WorkflowNode, index: number): Node<FlowNodeData> => ({
    id: workflowNode.key,
    type: 'default',
    position: { x: 70 + (index % 3) * 230, y: 70 + Math.floor(index / 3) * 150 },
    data: { workflow: { ...workflowNode, config: { ...(workflowNode.config || {}) } } }
  })

  const createEdge = (source: string, target: string): Edge => ({
    id: `${source}--${target}`,
    source,
    target,
    type: 'smoothstep',
    markerEnd: MarkerType.ArrowClosed
  })

  const loadWorkflow = async () => {
    if (!props.workflow) return
    const request = ++loadGeneration
    workflowDetail.value = null
    workflowName.value = ''
    workflowDescription.value = ''
    targetEnvironmentId.value = 0
    targetEnvironments.value = []
    flowNodes.value = []
    flowEdges.value = []
    loading.value = true
    validationIssues.value = []
    selectedNodeId.value = ''
    selectedEdgeId.value = ''
    try {
      const [detail, environments] = await Promise.all([
        getWorkflowApi(props.workflow.id),
        listEnvironmentsApi(props.workflow.project_id)
      ])
      if (request !== loadGeneration || !props.visible) return
      targetEnvironments.value = environments || []
      targetEnvironmentId.value = detail.target_environment_id || 0
      workflowDetail.value = detail
      workflowName.value = detail.name
      workflowDescription.value = detail.description || ''
      flowNodes.value = (detail.nodes || []).map(createFlowNode)
      flowEdges.value = (detail.nodes || []).flatMap((node) =>
        (node.depends_on || []).map((dependency) => createEdge(dependency, node.key))
      )
    } catch (error: any) {
      if (request === loadGeneration)
        ElMessage.error(error?.message || '工作流读取失败，请重新打开')
    } finally {
      if (request === loadGeneration) loading.value = false
    }
  }

  const addNode = (type: WorkflowNodeType) => {
    nodeSequence += 1
    const key = `${type.replace('_deploy', '')}-${Date.now().toString(36)}-${nodeSequence}`
    const service = props.services[0]
    const config: Record<string, unknown> = {}
    if (service && (type === 'build' || type === 'test' || isDeployNode(type))) {
      config.service_id = service.id
      if (type === 'build') config.branch = service.repo_branch
      if (type === 'test') config.service_id = service.id
      if (isDeployNode(type)) config.service_revision_id = service.current_revision_id
      if (type === 'helm_deploy') config.release_name = service.code
    }
    if (type === 'test') config.command = ''
    const workflowNode: WorkflowNode = {
      key,
      name: nodeMeta(type).label,
      type,
      config,
      timeout_seconds: type === 'build' ? 3600 : type === 'approval' ? 0 : 1800,
      retry_limit: type === 'build' ? 1 : 0
    }
    flowNodes.value.push(createFlowNode(workflowNode, flowNodes.value.length))
    selectedNodeId.value = key
    selectedEdgeId.value = ''
    void loadSelectedServiceRevisions()
  }

  const graphHasPath = (from: string, target: string, edges: Edge[]) => {
    const adjacency = new Map<string, string[]>()
    for (const edge of edges) {
      adjacency.set(edge.source, [...(adjacency.get(edge.source) || []), edge.target])
    }
    const pending = [from]
    const visited = new Set<string>()
    while (pending.length) {
      const current = pending.pop()
      if (!current || visited.has(current)) continue
      if (current === target) return true
      visited.add(current)
      pending.push(...(adjacency.get(current) || []))
    }
    return false
  }

  const isValidConnection = (connection: Connection) => {
    const { source, target } = connection
    if (!source || !target || source === target) return false
    if (flowEdges.value.some((edge) => edge.source === source && edge.target === target))
      return false
    return !graphHasPath(target, source, flowEdges.value)
  }

  const handleConnect = (connection: Connection) => {
    if (!isValidConnection(connection) || !connection.source || !connection.target) return
    flowEdges.value.push(createEdge(connection.source, connection.target))
  }
  const handleNodeClick = (event: { node: Node<FlowNodeData> }) => {
    selectedNodeId.value = event.node.id
    selectedEdgeId.value = ''
    void loadSelectedServiceRevisions()
  }
  const handleEdgeClick = (event: { edge: Edge }) => {
    selectedEdgeId.value = event.edge.id
    selectedNodeId.value = ''
  }
  const clearSelection = () => {
    selectedNodeId.value = ''
    selectedEdgeId.value = ''
  }
  const removeSelectedNode = () => {
    if (!selectedNodeId.value) return
    flowNodes.value = flowNodes.value.filter((item) => item.id !== selectedNodeId.value)
    flowEdges.value = flowEdges.value.filter(
      (item) => item.source !== selectedNodeId.value && item.target !== selectedNodeId.value
    )
    clearSelection()
  }
  const removeSelectedEdge = () => {
    flowEdges.value = flowEdges.value.filter((item) => item.id !== selectedEdgeId.value)
    selectedEdgeId.value = ''
  }

  const loadSelectedServiceRevisions = async () => {
    selectedServiceRevisions.value = []
    if (!selectedWorkflowNode.value || !isDeployNode(selectedWorkflowNode.value.type)) return
    if (!selectedServiceId.value) return
    selectedServiceRevisions.value = (await listServiceRevisionsApi(selectedServiceId.value)) || []
  }

  const syncSelectedService = () => {
    const service = props.services.find((item) => item.id === selectedServiceId.value)
    if (!service || !selectedWorkflowNode.value) return
    if (selectedWorkflowNode.value.type === 'build') selectedBranch.value = service.repo_branch
    if (selectedWorkflowNode.value.type === 'test')
      selectedWorkflowNode.value.config = {
        ...(selectedWorkflowNode.value.config || {}),
        service_id: service.id
      }
    if (isDeployNode(selectedWorkflowNode.value.type)) {
      selectedRevisionId.value = service.current_revision_id || 0
      if (selectedWorkflowNode.value.type === 'helm_deploy')
        selectedReleaseName.value = service.code
      void loadSelectedServiceRevisions()
    }
  }

  const serializeNodes = () =>
    flowNodes.value
      .filter((item) => item.data?.workflow)
      .map(({ id, data }) => ({
        ...data!.workflow,
        key: id,
        name: data!.workflow.name.trim(),
        depends_on: flowEdges.value
          .filter((edge) => edge.target === id)
          .map((edge) => edge.source)
          .sort()
      }))

  const serviceOrder = computed(() => workflowDeploymentOrder(serializeNodes()))
  const serviceSequenceActive = computed(() => hasWorkflowDeploymentSequence(serializeNodes()))
  const legacyServiceSequence = computed(() =>
    hasLegacyWorkflowDeploymentSequence(serializeNodes())
  )
  const serviceName = (id: number) =>
    props.services.find((service) => service.id === id)?.name || `服务 #${id}`
  const applyServiceOrder = (next: number[]) => {
    if (!props.editable || saving.value) return
    const ordered = withWorkflowDeploymentOrder(serializeNodes(), next)
    flowEdges.value = ordered.flatMap((node) =>
      (node.depends_on || []).map((dependency) => createEdge(dependency, node.key))
    )
    const orderedByKey = new Map(ordered.map((node) => [node.key, node]))
    flowNodes.value = flowNodes.value.map((node) => {
      const serviceId = Number(node.data?.workflow.config?.service_id || 0)
      const row = next.indexOf(serviceId)
      if (row < 0) return node
      const column =
        node.data!.workflow.type === 'build' ? 0 : isDeployNode(node.data!.workflow.type) ? 2 : 1
      return {
        ...node,
        data: { ...node.data!, workflow: orderedByKey.get(node.id)! },
        position: { x: 70 + column * 180, y: 70 + row * 150 }
      }
    })
  }
  const moveService = (id: number, direction: -1 | 1) => {
    const next = [...serviceOrder.value]
    const index = next.indexOf(id)
    const target = index + direction
    if (index < 0 || target < 0 || target >= next.length) return
    ;[next[index], next[target]] = [next[target], next[index]]
    applyServiceOrder(next)
  }

  const validateNodeConfigs = (nodes: WorkflowNode[]) => {
    if (!nodes.length) return '工作流至少需要一个节点'
    for (const node of nodes) {
      if (!node.name) return `节点 ${node.key} 缺少名称`
      if (node.type === 'build' && !Number(node.config?.service_id))
        return `构建节点 ${node.name} 未选择服务`
      if (node.type === 'test' && !Number(node.config?.service_id))
        return `测试节点 ${node.name} 未选择服务`
      if (node.type === 'test' && !String(node.config?.command || '').trim())
        return `测试节点 ${node.name} 未配置测试命令`
      if (node.type === 'test' && !String(node.config?.image || '').trim())
        return `测试节点 ${node.name} 未选择测试执行镜像`
      if (
        isDeployNode(node.type) &&
        (!Number(node.config?.service_id) || !Number(node.config?.service_revision_id))
      )
        return `部署节点 ${node.name} 未选择服务或 Revision`
    }
    return ''
  }

  const saveWorkflow = async () => {
    if (loading.value || saving.value || !props.editable) return
    if (!workflowDetail.value || !props.project || !workflowName.value.trim())
      return ElMessage.warning('请填写工作流名称')
    const nodes = serializeNodes()
    const configError = validateNodeConfigs(nodes)
    if (configError) return ElMessage.warning(configError)
    if (nodes.some((node) => isDeployNode(node.type)) && !targetEnvironmentId.value)
      return ElMessage.warning('包含部署节点，请先绑定目标环境')
    const definition = {
      delivery_type: props.project.delivery_type as 'k8s_yaml' | 'helm' | 'host',
      nodes
    }
    saving.value = true
    try {
      const validation = await validateWorkflowApi(workflowDetail.value.id, definition)
      validationIssues.value = validation.issues || []
      if (!validation.valid) return ElMessage.warning('工作流编排校验未通过')
      workflowDetail.value = await updateWorkflowApi(workflowDetail.value.id, {
        ...(targetEnvironmentId.value ? { target_environment_id: targetEnvironmentId.value } : {}),
        expected_version: workflowDetail.value.definition_version,
        name: workflowName.value.trim(),
        description: workflowDescription.value.trim(),
        definition
      })
      ElMessage.success('工作流已保存')
      emit('saved')
      dialogVisible.value = false
    } finally {
      saving.value = false
    }
  }

  watch(
    () => [props.visible, props.workflow?.id],
    ([visible]) => {
      if (visible) void loadWorkflow()
      else loadGeneration++
    }
  )
</script>

<style scoped lang="scss">
  .designer-shell {
    max-height: min(630px, calc(100vh - 270px));
    overflow-y: auto;
  }
  .designer-toolbar {
    min-height: 58px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .workflow-fields {
    display: grid;
    grid-template-columns: 220px minmax(260px, 1fr);
    gap: 8px;
    flex: 1;
  }
  .environment-readonly-hint {
    grid-column: 1 / -1;
    color: var(--art-text-gray-500);
  }
  .node-actions {
    display: flex;
    gap: 8px;
  }
  .service-order {
    padding: 12px 0;
    border-bottom: 1px solid var(--art-border-color);
  }
  .service-order-heading {
    display: flex;
    align-items: baseline;
    gap: 12px;
    margin-bottom: 8px;
  }
  .service-order-heading strong {
    font-size: 13px;
  }
  .service-order-heading small {
    color: var(--art-text-gray-500);
  }
  .service-order-list {
    display: flex;
    gap: 8px;
    overflow-x: auto;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .service-order-list li {
    min-width: 190px;
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 4px 7px;
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
  }
  .order-number {
    flex: none;
    width: 20px;
    text-align: center;
    color: var(--el-color-primary);
    font-weight: 600;
  }
  .order-name {
    min-width: 0;
    flex: 1;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font-size: 13px;
  }
  .service-order-list .el-button + .el-button {
    margin-left: 0;
  }
  .designer-body {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 300px;
    min-height: min(500px, calc(94vh - 260px));
  }
  .flow-panel {
    min-width: 0;
    border-right: 1px solid var(--art-border-color);
  }
  .workflow-flow {
    height: clamp(340px, calc(94vh - 260px), 500px);
    background: var(--art-bg-color);
  }
  .workflow-node {
    width: 176px;
    min-height: 58px;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 8px 12px;
    background: var(--art-main-bg-color);
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    box-shadow: 0 3px 10px rgb(15 23 42 / 7%);
  }
  .workflow-node.is-selected {
    border-color: var(--el-color-primary);
    box-shadow: 0 0 0 2px var(--el-color-primary-light-8);
  }
  .workflow-node .node-icon {
    width: 30px;
    height: 30px;
    display: grid;
    place-items: center;
    border-radius: 6px;
    color: #2459a9;
    background: #e9f1ff;
  }
  .workflow-node.is-approval .node-icon {
    color: #8a5b13;
    background: #fff4dc;
  }
  .workflow-node.is-yaml_deploy .node-icon,
  .workflow-node.is-helm_deploy .node-icon,
  .workflow-node.is-host_deploy .node-icon {
    color: #126747;
    background: #e3f7ef;
  }
  .workflow-node > span:last-of-type {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .workflow-node strong,
  .workflow-node small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .workflow-node small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
  }
  .inspector-panel {
    padding: 16px;
  }
  .inspector-heading {
    display: flex;
    justify-content: space-between;
    margin-bottom: 18px;
  }
  .inspector-heading span {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .inspector-heading small {
    margin-top: 4px;
    color: var(--art-text-gray-500);
    overflow-wrap: anywhere;
  }
  .inspector-note {
    color: var(--art-text-gray-500);
    line-height: 1.6;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 10px;
  }
  .validation-panel {
    display: flex;
    flex-direction: column;
    gap: 4px;
    margin-top: 12px;
    padding: 10px 12px;
    border-left: 3px solid var(--el-color-danger);
    background: var(--el-color-danger-light-9);
    color: var(--el-color-danger);
  }
  @media (max-width: 900px) {
    .designer-toolbar {
      align-items: stretch;
      flex-direction: column;
      padding: 10px 0;
    }
    .workflow-fields {
      grid-template-columns: minmax(0, 1fr);
    }
    .node-actions {
      flex-wrap: wrap;
    }
    .designer-body {
      grid-template-columns: minmax(0, 1fr);
    }
    .flow-panel {
      border-right: 0;
      border-bottom: 1px solid var(--art-border-color);
    }
    .inspector-panel {
      min-height: 280px;
    }
    .workflow-flow {
      height: 360px;
    }
  }
</style>
