<template>
  <ElDialog
    v-model="visible"
    title="🚀 部署告警规则到集群"
    width="1200px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div class="deploy-container">
      <!-- 部署提示 -->
      <ElAlert type="info" :closable="false" show-icon class="deploy-alert">
        <template #title>
          <div class="deploy-tips">
            <strong>📌 部署说明：</strong>
            选择要部署的分组和规则到 Kubernetes 集群，系统会自动生成 PrometheusRule 资源并应用。
          </div>
        </template>
      </ElAlert>

      <div class="deploy-main">
        <!-- 左侧：分组和规则选择 -->
        <div class="selection-section">
          <div class="section-header">
            <div class="header-content">
              <div class="header-left">
                <ElIcon class="header-icon"><Bell /></ElIcon>
                <span class="header-title">选择分组和规则</span>
              </div>
              <div class="header-actions">
                <ElButton size="small" type="primary" @click="handleSelectAll">
                  {{ isAllSelected ? '取消全选' : '全选' }}
                </ElButton>
              </div>
            </div>
          </div>

          <div class="selection-content" v-loading="treeLoading">
            <ElTree
              ref="treeRef"
              :data="treeData"
              :props="treeProps"
              show-checkbox
              node-key="nodeKey"
              :default-expand-all="true"
              @check="handleTreeCheck"
            >
              <template #default="{ node, data }">
                <div class="tree-node">
                  <ElIcon class="node-icon" :class="data.type">
                    <Folder v-if="data.type === 'group'" />
                    <Bell v-else />
                  </ElIcon>
                  <span class="node-label">{{ data.name }}</span>
                  <ElTag
                    v-if="data.type === 'rule'"
                    size="small"
                    :type="getSeverityType(data.severity)"
                  >
                    {{ getSeverityLabel(data.severity) }}
                  </ElTag>
                  <ElTag v-if="data.type === 'group'" size="small" type="info">
                    {{ data.children?.length || 0 }} 条规则
                  </ElTag>
                </div>
              </template>
            </ElTree>

            <ElEmpty
              v-if="!treeData.length && !treeLoading"
              description="暂无分组和规则"
              :image-size="80"
            />
          </div>

          <div class="selection-summary">
            <span>已选择：</span>
            <ElTag type="primary">{{ selectedGroupIds.length }} 个分组</ElTag>
            <ElTag type="success">{{ selectedRuleIds.length }} 条规则</ElTag>
          </div>
        </div>

        <!-- 右侧：部署配置 -->
        <div class="config-section">
          <div class="section-header">
            <div class="header-content">
              <div class="header-left">
                <ElIcon class="header-icon"><Setting /></ElIcon>
                <span class="header-title">部署配置</span>
              </div>
            </div>
          </div>

          <div class="config-content">
            <ElForm ref="formRef" :model="deployForm" :rules="rules" label-position="top">
              <!-- 目标集群 -->
              <ElFormItem label="目标集群" prop="clusterUuid">
                <ElSelect
                  v-model="deployForm.clusterUuid"
                  placeholder="请选择集群"
                  style="width: 100%"
                  :loading="clusterLoading"
                  filterable
                  clearable
                  @change="handleClusterChange"
                  @visible-change="handleClusterDropdownVisible"
                >
                  <ElOption
                    v-for="cluster in clusterList"
                    :key="cluster.uuid"
                    :label="`${cluster.name} (${cluster.environment})`"
                    :value="cluster.uuid"
                  >
                    <div class="cluster-option">
                      <div class="cluster-header">
                        <span class="cluster-name">{{ cluster.name }}</span>
                        <ElTag size="small" :type="getEnvironmentType(cluster.environment)">
                          {{ cluster.environment }}
                        </ElTag>
                      </div>
                      <div class="cluster-tags">
                        <ElTag size="small" type="info">{{ cluster.version }}</ElTag>
                        <ElTag size="small" :type="getHealthStatusType(cluster.healthStatus)">
                          {{ getHealthStatusLabel(cluster.healthStatus) }}
                        </ElTag>
                      </div>
                    </div>
                  </ElOption>

                  <template #footer>
                    <div v-if="hasMoreClusters" class="select-footer">
                      <ElButton
                        size="small"
                        text
                        :loading="clusterLoading"
                        @click.stop="loadMoreClusters"
                      >
                        {{ clusterLoading ? '加载中...' : '加载更多' }}
                      </ElButton>
                    </div>
                  </template>
                </ElSelect>
              </ElFormItem>

              <!-- 命名空间 -->
              <ElFormItem label="命名空间" prop="namespace">
                <ElSelect
                  v-model="deployForm.namespace"
                  placeholder="请选择命名空间"
                  style="width: 100%"
                  :loading="namespaceLoading"
                  filterable
                  clearable
                  :disabled="!deployForm.clusterUuid"
                  @visible-change="handleNamespaceDropdownVisible"
                >
                  <ElOption v-for="ns in namespaceList" :key="ns" :label="ns" :value="ns">
                    <div class="namespace-option">
                      <ElIcon><FolderOpened /></ElIcon>
                      <span>{{ ns }}</span>
                    </div>
                  </ElOption>
                </ElSelect>
              </ElFormItem>

              <!-- YAML 预览 -->
              <div class="preview-section">
                <div class="preview-header">
                  <span class="preview-title">
                    <ElIcon><View /></ElIcon>
                    YAML 预览
                  </span>
                  <ElButton
                    size="small"
                    type="primary"
                    text
                    @click="handlePreview"
                    :loading="previewLoading"
                  >
                    刷新预览
                  </ElButton>
                </div>
                <div class="preview-content" v-loading="previewLoading">
                  <pre v-if="previewYaml" class="yaml-preview">{{ previewYaml }}</pre>
                  <ElEmpty v-else description="点击'刷新预览'查看生成的 YAML" :image-size="60" />
                </div>
              </div>
            </ElForm>

            <!-- 部署结果 -->
            <div v-if="deployResult" class="deploy-result">
              <ElAlert
                :type="deployResult.success ? 'success' : 'error'"
                :title="deployResult.success ? '✅ 部署成功' : '❌ 部署失败'"
                :closable="true"
                show-icon
                @close="deployResult = null"
              >
                <div v-if="deployResult.success" class="result-content">
                  <div
                    >集群：<strong>{{ deployResult.clusterName }}</strong></div
                  >
                  <div
                    >命名空间：<strong>{{ deployResult.namespace }}</strong></div
                  >
                </div>
                <div v-else>{{ deployResult.message }}</div>
              </ElAlert>
            </div>
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="visible = false" size="large">
          <ElIcon><Close /></ElIcon>
          取消
        </ElButton>
        <ElButton
          type="primary"
          size="large"
          :loading="loading"
          @click="handleDeploy"
          :disabled="!canDeploy"
        >
          <ElIcon><Position /></ElIcon>
          {{ loading ? '部署中...' : '确认部署' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch } from 'vue'
  import { ElMessage, ElTree, type FormInstance, type FormRules } from 'element-plus'
  import {
    Close,
    Position,
    FolderOpened,
    Bell,
    Setting,
    View,
    Folder
  } from '@element-plus/icons-vue'
  import {
    deployAlertRulesApi,
    previewAlertRulesYamlApi,
    getAlertRuleTreeApi,
    type AlertRuleTreeNode
  } from '@/api/manager/alert'
  import { searchClusterApi, getClusterNamespaceListApi, type Cluster } from '@/api/manager/cluster'

  interface Props {
    modelValue: boolean
    fileId?: number
  }

  interface DeployResult {
    success: boolean
    clusterName?: string
    namespace?: string
    message?: string
  }

  interface TreeNode {
    nodeKey: string
    id: number
    code: string
    name: string
    type: 'group' | 'rule'
    isEnabled: boolean
    severity?: string
    children?: TreeNode[]
  }

  const props = defineProps<Props>()

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    success: []
  }>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const treeRef = ref<InstanceType<typeof ElTree>>()
  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const treeLoading = ref(false)
  const clusterLoading = ref(false)
  const namespaceLoading = ref(false)
  const previewLoading = ref(false)

  const treeData = ref<TreeNode[]>([])
  const selectedGroupIds = ref<number[]>([])
  const selectedRuleIds = ref<number[]>([])

  const clusterList = ref<Cluster[]>([])
  const clusterPagination = reactive({
    current: 1,
    size: 20,
    total: 0
  })

  const namespaceList = ref<string[]>([])
  const previewYaml = ref('')
  const deployResult = ref<DeployResult | null>(null)

  const deployForm = reactive({
    clusterUuid: '',
    namespace: 'monitoring'
  })

  const rules: FormRules = {
    clusterUuid: [{ required: true, message: '请选择目标集群', trigger: 'change' }],
    namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
  }

  const treeProps = {
    label: 'name',
    children: 'children'
  }

  const canDeploy = computed(() => {
    return (
      deployForm.clusterUuid &&
      deployForm.namespace &&
      (selectedGroupIds.value.length > 0 || selectedRuleIds.value.length > 0)
    )
  })

  const hasMoreClusters = computed(() => {
    return clusterList.value.length < clusterPagination.total
  })

  const isAllSelected = computed(() => {
    if (treeData.value.length === 0) return false
    const allKeys = getAllNodeKeys(treeData.value)
    const checkedKeys = treeRef.value?.getCheckedKeys() || []
    return allKeys.every((key) => checkedKeys.includes(key))
  })

  const getAllNodeKeys = (nodes: TreeNode[]): string[] => {
    const keys: string[] = []
    nodes.forEach((node) => {
      keys.push(node.nodeKey)
      if (node.children) {
        keys.push(...getAllNodeKeys(node.children))
      }
    })
    return keys
  }

  const transformTreeData = (nodes: AlertRuleTreeNode[]): TreeNode[] => {
    return nodes.map((node) => ({
      nodeKey: `${node.type}-${node.id}`,
      id: node.id,
      code: node.code,
      name: node.name,
      type: node.type as 'group' | 'rule',
      isEnabled: node.isEnabled,
      severity: node.severity,
      children: node.children ? transformTreeData(node.children) : undefined
    }))
  }

  const loadTreeData = async () => {
    if (!props.fileId) return

    try {
      treeLoading.value = true
      const res = await getAlertRuleTreeApi({ fileId: props.fileId })
      treeData.value = transformTreeData(res.data || [])
    } finally {
      treeLoading.value = false
    }
  }

  const handleSelectAll = () => {
    if (!treeRef.value) return

    if (isAllSelected.value) {
      treeRef.value.setCheckedKeys([])
      selectedGroupIds.value = []
      selectedRuleIds.value = []
    } else {
      const allKeys = getAllNodeKeys(treeData.value)
      treeRef.value.setCheckedKeys(allKeys)
      updateSelectedIds()
    }
  }

  const handleTreeCheck = () => {
    updateSelectedIds()
  }

  const updateSelectedIds = () => {
    if (!treeRef.value) return

    const checkedNodes = treeRef.value.getCheckedNodes() as TreeNode[]
    selectedGroupIds.value = checkedNodes.filter((n) => n.type === 'group').map((n) => n.id)
    selectedRuleIds.value = checkedNodes.filter((n) => n.type === 'rule').map((n) => n.id)
  }

  watch(
    () => visible.value,
    async (newVisible) => {
      if (newVisible && props.fileId) {
        await loadTreeData()
      }
    }
  )

  const loadClusterList = async (append = false) => {
    try {
      clusterLoading.value = true
      const res = await searchClusterApi({
        page: clusterPagination.current,
        pageSize: clusterPagination.size,
        orderStr: 'id',
        isAsc: false
      })

      if (append) {
        clusterList.value = [...clusterList.value, ...res.items]
      } else {
        clusterList.value = res.items
      }

      clusterPagination.total = res.total
    } finally {
      clusterLoading.value = false
    }
  }

  const loadMoreClusters = async () => {
    clusterPagination.current++
    await loadClusterList(true)
  }

  const handleClusterDropdownVisible = (visible: boolean) => {
    if (visible && clusterList.value.length === 0) {
      clusterPagination.current = 1
      loadClusterList()
    }
  }

  const handleClusterChange = async (uuid: string) => {
    deployResult.value = null
    deployForm.namespace = 'monitoring'
    namespaceList.value = []
    previewYaml.value = ''

    if (uuid) {
      await loadNamespaceList(uuid)
    }
  }

  const loadNamespaceList = async (clusterUuid: string) => {
    try {
      namespaceLoading.value = true
      const res = await getClusterNamespaceListApi(clusterUuid)
      namespaceList.value = res || []
    } finally {
      namespaceLoading.value = false
    }
  }

  const handleNamespaceDropdownVisible = (visible: boolean) => {
    if (!visible) {
      // 清理操作
    }
  }

  const handlePreview = async () => {
    if (!props.fileId || !deployForm.namespace) {
      ElMessage.warning('请先选择命名空间')
      return
    }

    try {
      previewLoading.value = true
      const res = await previewAlertRulesYamlApi({
        fileId: props.fileId,
        namespace: deployForm.namespace,
        groupIds: selectedGroupIds.value,
        ruleIds: selectedRuleIds.value
      })
      previewYaml.value = res.yamlStr
    } finally {
      previewLoading.value = false
    }
  }

  const getEnvironmentType = (env: string) => {
    const typeMap: Record<string, any> = {
      production: 'danger',
      staging: 'warning',
      development: 'info',
      test: 'info'
    }
    return typeMap[env] || 'info'
  }

  const getHealthStatusType = (status: number) => {
    const typeMap: Record<number, any> = {
      1: 'success',
      2: 'danger',
      3: 'warning',
      4: 'info'
    }
    return typeMap[status] || 'info'
  }

  const getHealthStatusLabel = (status: number) => {
    const labelMap: Record<number, string> = {
      1: '健康',
      2: '不健康',
      3: '降级',
      4: '未知'
    }
    return labelMap[status] || '未知'
  }

  const getSeverityType = (severity?: string) => {
    const typeMap: Record<string, any> = {
      critical: 'danger',
      warning: 'warning',
      info: 'info'
    }
    return typeMap[severity || ''] || 'info'
  }

  const getSeverityLabel = (severity?: string) => {
    const labelMap: Record<string, string> = {
      critical: '严重',
      warning: '警告',
      info: '提示'
    }
    return labelMap[severity || ''] || severity || ''
  }

  const handleDeploy = async () => {
    if (!formRef.value) return
    if (!props.fileId) {
      ElMessage.error('缺少文件ID')
      return
    }

    if (selectedGroupIds.value.length === 0 && selectedRuleIds.value.length === 0) {
      ElMessage.warning('请至少选择一个分组或规则')
      return
    }

    try {
      await formRef.value.validate()

      loading.value = true
      deployResult.value = null

      await deployAlertRulesApi({
        fileId: props.fileId,
        clusterUuid: deployForm.clusterUuid,
        namespace: deployForm.namespace,
        groupIds: selectedGroupIds.value,
        ruleIds: selectedRuleIds.value
      })

      const cluster = clusterList.value.find((c) => c.uuid === deployForm.clusterUuid)

      deployResult.value = {
        success: true,
        clusterName: cluster?.name || deployForm.clusterUuid,
        namespace: deployForm.namespace
      }

      ElMessage.success('部署成功！')

      setTimeout(() => {
        emit('success')
        visible.value = false
      }, 2000)
    } catch (error: any) {
    } finally {
      loading.value = false
    }
  }

  const handleClosed = () => {
    deployForm.clusterUuid = ''
    deployForm.namespace = 'monitoring'
    selectedGroupIds.value = []
    selectedRuleIds.value = []
    deployResult.value = null
    clusterList.value = []
    namespaceList.value = []
    treeData.value = []
    previewYaml.value = ''
    clusterPagination.current = 1
    clusterPagination.total = 0
    formRef.value?.clearValidate()
  }
</script>

<style scoped lang="scss">
  .deploy-container {
    display: flex;
    flex-direction: column;
    gap: 16px;

    .deploy-alert {
      .deploy-tips {
        line-height: 1.8;

        strong {
          margin-right: 8px;
        }
      }
    }

    .deploy-main {
      display: flex;
      gap: 20px;
      height: 550px;

      .selection-section {
        flex: 1;
        display: flex;
        flex-direction: column;
        border: 1px solid var(--el-border-color);
        border-radius: 8px;
        overflow: hidden;
        background: var(--el-bg-color);

        .section-header {
          padding: 14px 16px;
          background: linear-gradient(
            135deg,
            var(--el-color-primary-light-9),
            var(--el-fill-color)
          );
          border-bottom: 1px solid var(--el-border-color-light);

          .header-content {
            display: flex;
            justify-content: space-between;
            align-items: center;

            .header-left {
              display: flex;
              align-items: center;
              gap: 10px;

              .header-icon {
                font-size: 18px;
                color: var(--el-color-primary);
              }

              .header-title {
                font-size: 15px;
                font-weight: 600;
                color: var(--el-text-color-primary);
              }
            }
          }
        }

        .selection-content {
          flex: 1;
          overflow-y: auto;
          padding: 12px;

          .tree-node {
            display: flex;
            align-items: center;
            gap: 8px;
            padding: 4px 0;

            .node-icon {
              font-size: 16px;

              &.group {
                color: var(--el-color-warning);
              }

              &.rule {
                color: var(--el-color-primary);
              }
            }

            .node-label {
              flex: 1;
              font-size: 13px;
            }
          }
        }

        .selection-summary {
          padding: 12px 16px;
          border-top: 1px solid var(--el-border-color-light);
          background: var(--el-fill-color-light);
          display: flex;
          align-items: center;
          gap: 12px;
          font-size: 13px;
          color: var(--el-text-color-secondary);
        }
      }

      .config-section {
        width: 450px;
        flex-shrink: 0;
        display: flex;
        flex-direction: column;
        border: 1px solid var(--el-border-color);
        border-radius: 8px;
        overflow: hidden;
        background: var(--el-bg-color);

        .section-header {
          padding: 14px 16px;
          background: linear-gradient(
            135deg,
            var(--el-color-success-light-9),
            var(--el-fill-color)
          );
          border-bottom: 1px solid var(--el-border-color-light);

          .header-content {
            .header-left {
              display: flex;
              align-items: center;
              gap: 10px;

              .header-icon {
                font-size: 18px;
                color: var(--el-color-success);
              }

              .header-title {
                font-size: 15px;
                font-weight: 600;
                color: var(--el-text-color-primary);
              }
            }
          }
        }

        .config-content {
          flex: 1;
          overflow-y: auto;
          padding: 16px;

          .cluster-option {
            display: flex;
            flex-direction: column;
            gap: 6px;
            padding: 4px 0;

            .cluster-header {
              display: flex;
              justify-content: space-between;
              align-items: center;
              gap: 8px;

              .cluster-name {
                flex: 1;
                font-weight: 500;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              }
            }

            .cluster-tags {
              display: flex;
              gap: 8px;
            }
          }

          .namespace-option {
            display: flex;
            align-items: center;
            gap: 8px;
          }

          .select-footer {
            padding: 8px;
            text-align: center;
            border-top: 1px solid var(--el-border-color-lighter);
          }

          .preview-section {
            margin-top: 16px;
            border: 1px solid var(--el-border-color-light);
            border-radius: 8px;
            overflow: hidden;

            .preview-header {
              display: flex;
              justify-content: space-between;
              align-items: center;
              padding: 10px 12px;
              background: var(--el-fill-color-light);
              border-bottom: 1px solid var(--el-border-color-lighter);

              .preview-title {
                display: flex;
                align-items: center;
                gap: 6px;
                font-size: 13px;
                font-weight: 500;
                color: var(--el-text-color-primary);
              }
            }

            .preview-content {
              height: 180px;
              overflow-y: auto;

              .yaml-preview {
                margin: 0;
                padding: 12px;
                font-family: var(--ax-font-mono);
                font-size: 12px;
                line-height: 1.5;
                white-space: pre-wrap;
                word-break: break-all;
                background: #f8f9fa;
              }
            }
          }

          .deploy-result {
            margin-top: 16px;
            animation: slideIn 0.3s ease-out;

            .result-content {
              div {
                margin: 4px 0;

                strong {
                  color: var(--el-color-primary);
                }
              }
            }
          }
        }
      }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 12px;
  }

  @keyframes slideIn {
    from {
      opacity: 0;
      transform: translateY(-10px);
    }
    to {
      opacity: 1;
      transform: translateY(0);
    }
  }
</style>
