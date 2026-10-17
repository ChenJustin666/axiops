<template>
  <ElDialog
    :model-value="visible"
    title="NetworkPolicy 详情"
    width="900px"
    top="5vh"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
  >
    <!-- 加载状态 -->
    <div v-if="loading" class="loading-state">
      <ElSkeleton :rows="8" animated />
    </div>

    <!-- 详情内容 -->
    <div v-else-if="detail" class="detail-content">
      <!-- 基础信息 -->
      <div class="detail-section">
        <div class="section-header">
          <Database :size="16" />
          <span>基础信息</span>
        </div>
        <div class="info-grid">
          <div class="info-item">
            <span class="label">名称</span>
            <span class="value">{{ detail.name }}</span>
          </div>
          <div class="info-item">
            <span class="label">命名空间</span>
            <span class="value">{{ detail.namespace }}</span>
          </div>
          <div class="info-item">
            <span class="label">运行时长</span>
            <span class="value">{{ detail.age }}</span>
          </div>
          <div class="info-item">
            <span class="label">策略类型</span>
            <span class="value">
              <ElTag
                v-for="type in detail.policyTypes"
                :key="type"
                :type="type === 'Ingress' ? 'success' : 'warning'"
                size="small"
                style="margin-right: 4px"
              >
                {{ type }}
              </ElTag>
            </span>
          </div>
        </div>
      </div>

      <!-- Pod 选择器 -->
      <div class="detail-section">
        <div class="section-header">
          <Target :size="16" />
          <span>Pod 选择器</span>
        </div>
        <div class="selector-content">
          <div v-if="Object.keys(detail.podSelector || {}).length > 0" class="labels-display">
            <ElTag
              v-for="[key, value] in Object.entries(detail.podSelector)"
              :key="key"
              type="primary"
              size="small"
            >
              {{ key }}={{ value }}
            </ElTag>
          </div>
          <div
            v-else-if="
              detail.podSelectorMatchExpressions && detail.podSelectorMatchExpressions.length > 0
            "
            class="expressions-display"
          >
            <div
              v-for="(expr, index) in detail.podSelectorMatchExpressions"
              :key="index"
              class="expression-item"
            >
              <ElTag type="info" size="small">
                {{ expr.key }} {{ expr.operator }}
                {{ expr.values?.join(', ') || '' }}
              </ElTag>
            </div>
          </div>
          <div v-else class="empty-selector">
            <ElIcon><Info /></ElIcon>
            <span>未指定选择器，应用于命名空间内所有 Pod</span>
          </div>
        </div>
      </div>

      <!-- 入站规则 -->
      <div v-if="detail.policyTypes?.includes('Ingress')" class="detail-section">
        <div class="section-header">
          <ArrowDown :size="16" />
          <span>入站规则 ({{ detail.ingressRules?.length || 0 }})</span>
        </div>
        <div class="rules-content">
          <div v-if="!detail.ingressRules || detail.ingressRules.length === 0" class="empty-rules">
            <ElIcon><XCircle /></ElIcon>
            <span>拒绝所有入站流量</span>
          </div>
          <div v-else class="rules-list">
            <div v-for="(rule, index) in detail.ingressRules" :key="index" class="rule-item">
              <div class="rule-header">
                <ElTag type="success" size="small">规则 {{ index + 1 }}</ElTag>
              </div>
              <div class="rule-body">
                <!-- 端口 -->
                <div v-if="rule.ports && rule.ports.length > 0" class="rule-section">
                  <span class="rule-section-label">端口:</span>
                  <div class="rule-section-content">
                    <ElTag
                      v-for="(port, pIndex) in rule.ports"
                      :key="pIndex"
                      size="small"
                      type="info"
                    >
                      {{ port.protocol || 'TCP' }}/{{ port.port || port.namedPort || 'All' }}
                      <span v-if="port.endPort">-{{ port.endPort }}</span>
                    </ElTag>
                  </div>
                </div>
                <!-- 来源 -->
                <div v-if="rule.from && rule.from.length > 0" class="rule-section">
                  <span class="rule-section-label">来源:</span>
                  <div class="rule-section-content">
                    <div v-for="(peer, pIndex) in rule.from" :key="pIndex" class="peer-item">
                      <ElTag v-if="peer.ipBlock" type="warning" size="small">
                        IP: {{ peer.ipBlock.cidr }}
                      </ElTag>
                      <div
                        v-else-if="peer.podSelector || peer.podSelectorExprs"
                        class="peer-selector"
                      >
                        <ElTag type="success" size="small">Pod</ElTag>
                        <span v-if="peer.podSelector" class="peer-labels">
                          <ElTag
                            v-for="[key, value] in Object.entries(peer.podSelector)"
                            :key="key"
                            size="small"
                          >
                            {{ key }}={{ value }}
                          </ElTag>
                        </span>
                      </div>
                      <div
                        v-else-if="peer.namespaceSelector || peer.namespaceSelectorExprs"
                        class="peer-selector"
                      >
                        <ElTag type="primary" size="small">Namespace</ElTag>
                        <span v-if="peer.namespaceSelector" class="peer-labels">
                          <ElTag
                            v-for="[key, value] in Object.entries(peer.namespaceSelector)"
                            :key="key"
                            size="small"
                          >
                            {{ key }}={{ value }}
                          </ElTag>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
                <div
                  v-if="
                    (!rule.ports || rule.ports.length === 0) &&
                    (!rule.from || rule.from.length === 0)
                  "
                  class="rule-section"
                >
                  <span class="rule-section-label">允许所有</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 出站规则 -->
      <div v-if="detail.policyTypes?.includes('Egress')" class="detail-section">
        <div class="section-header">
          <ArrowUp :size="16" />
          <span>出站规则 ({{ detail.egressRules?.length || 0 }})</span>
        </div>
        <div class="rules-content">
          <div v-if="!detail.egressRules || detail.egressRules.length === 0" class="empty-rules">
            <ElIcon><XCircle /></ElIcon>
            <span>拒绝所有出站流量</span>
          </div>
          <div v-else class="rules-list">
            <div v-for="(rule, index) in detail.egressRules" :key="index" class="rule-item">
              <div class="rule-header">
                <ElTag type="warning" size="small">规则 {{ index + 1 }}</ElTag>
              </div>
              <div class="rule-body">
                <!-- 端口 -->
                <div v-if="rule.ports && rule.ports.length > 0" class="rule-section">
                  <span class="rule-section-label">端口:</span>
                  <div class="rule-section-content">
                    <ElTag
                      v-for="(port, pIndex) in rule.ports"
                      :key="pIndex"
                      size="small"
                      type="info"
                    >
                      {{ port.protocol || 'TCP' }}/{{ port.port || port.namedPort || 'All' }}
                      <span v-if="port.endPort">-{{ port.endPort }}</span>
                    </ElTag>
                  </div>
                </div>
                <!-- 目标 -->
                <div v-if="rule.to && rule.to.length > 0" class="rule-section">
                  <span class="rule-section-label">目标:</span>
                  <div class="rule-section-content">
                    <div v-for="(peer, pIndex) in rule.to" :key="pIndex" class="peer-item">
                      <ElTag v-if="peer.ipBlock" type="warning" size="small">
                        IP: {{ peer.ipBlock.cidr }}
                      </ElTag>
                      <div
                        v-else-if="peer.podSelector || peer.podSelectorExprs"
                        class="peer-selector"
                      >
                        <ElTag type="success" size="small">Pod</ElTag>
                        <span v-if="peer.podSelector" class="peer-labels">
                          <ElTag
                            v-for="[key, value] in Object.entries(peer.podSelector)"
                            :key="key"
                            size="small"
                          >
                            {{ key }}={{ value }}
                          </ElTag>
                        </span>
                      </div>
                      <div
                        v-else-if="peer.namespaceSelector || peer.namespaceSelectorExprs"
                        class="peer-selector"
                      >
                        <ElTag type="primary" size="small">Namespace</ElTag>
                        <span v-if="peer.namespaceSelector" class="peer-labels">
                          <ElTag
                            v-for="[key, value] in Object.entries(peer.namespaceSelector)"
                            :key="key"
                            size="small"
                          >
                            {{ key }}={{ value }}
                          </ElTag>
                        </span>
                      </div>
                    </div>
                  </div>
                </div>
                <div
                  v-if="
                    (!rule.ports || rule.ports.length === 0) && (!rule.to || rule.to.length === 0)
                  "
                  class="rule-section"
                >
                  <span class="rule-section-label">允许所有</span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- 标签和注解 -->
      <div class="detail-section">
        <div class="section-header">
          <Tag :size="16" />
          <span>标签与注解</span>
        </div>
        <div class="labels-annotations">
          <div v-if="detail.labels && Object.keys(detail.labels).length > 0" class="la-section">
            <span class="la-label">Labels:</span>
            <div class="la-tags">
              <ElTag
                v-for="[key, value] in Object.entries(detail.labels)"
                :key="key"
                type="info"
                size="small"
              >
                {{ key }}={{ value }}
              </ElTag>
            </div>
          </div>
          <div
            v-if="detail.annotations && Object.keys(detail.annotations).length > 0"
            class="la-section"
          >
            <span class="la-label">Annotations:</span>
            <div class="la-tags">
              <ElTag
                v-for="[key, value] in Object.entries(detail.annotations)"
                :key="key"
                type="info"
                size="small"
              >
                {{ key }}={{ value }}
              </ElTag>
            </div>
          </div>
          <div
            v-if="
              (!detail.labels || Object.keys(detail.labels).length === 0) &&
              (!detail.annotations || Object.keys(detail.annotations).length === 0)
            "
            class="empty-la"
          >
            无
          </div>
        </div>
      </div>
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <ElButton @click="handleClose">关 闭</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { Database, Target, ArrowDown, ArrowUp, Tag, Info, XCircle } from 'lucide-vue-next'
  import {
    getNetworkPolicyDetailApi,
    type NetworkPolicyDetail
  } from '../../../../api/workload/networkpolicy'
  import type { ProjectWorkspace, ProjectCluster } from '../../../../api/manager/project'

  interface Props {
    modelValue: boolean
    networkPolicyName: string
    workspace: ProjectWorkspace | null
    cluster: ProjectCluster | null
  }

  const props = defineProps<Props>()

  const emit = defineEmits(['update:modelValue'])

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const loading = ref(false)
  const detail = ref<NetworkPolicyDetail | null>(null)

  const loadDetail = async () => {
    if (!props.cluster || !props.workspace || !props.networkPolicyName) {
      return
    }

    loading.value = true
    try {
      detail.value = await getNetworkPolicyDetailApi({
        clusterUuid: props.cluster.clusterUuid,
        namespace: props.workspace.namespace,
        name: props.networkPolicyName
      })
    } finally {
      loading.value = false
    }
  }

  watch(visible, (val) => {
    if (val) {
      loadDetail()
    } else {
      detail.value = null
    }
  })

  const handleClose = () => {
    visible.value = false
  }
</script>

<style lang="scss" scoped>
  .loading-state {
    padding: 40px;
  }

  .detail-content {
    max-height: 65vh;
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

  .detail-section {
    margin-bottom: 20px;
    padding: 16px;
    background: #fafafa;
    border-radius: 4px;
    border: 1px solid #e4e7ed;

    &:last-child {
      margin-bottom: 0;
    }

    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 12px;
      padding-bottom: 8px;
      border-bottom: 1px solid #e4e7ed;
      font-size: 14px;
      font-weight: 500;
      color: #303133;
    }
  }

  .info-grid {
    display: grid;
    grid-template-columns: repeat(2, 1fr);
    gap: 12px;

    .info-item {
      display: flex;
      align-items: center;
      gap: 8px;

      .label {
        font-size: 13px;
        color: #909399;
        min-width: 80px;
      }

      .value {
        font-size: 13px;
        color: #303133;
        font-weight: 500;
      }
    }
  }

  .selector-content {
    min-height: 40px;
    display: flex;
    align-items: center;

    .labels-display,
    .expressions-display {
      display: flex;
      flex-wrap: wrap;
      gap: 6px;
    }

    .empty-selector {
      display: flex;
      align-items: center;
      gap: 6px;
      color: #909399;
      font-size: 13px;
    }
  }

  .rules-content {
    .empty-rules {
      display: flex;
      align-items: center;
      justify-content: center;
      gap: 8px;
      padding: 20px;
      color: #f56c6c;
      font-size: 13px;
    }

    .rules-list {
      display: flex;
      flex-direction: column;
      gap: 12px;
    }

    .rule-item {
      padding: 12px;
      background: white;
      border-radius: 4px;
      border: 1px solid #dcdfe6;

      .rule-header {
        margin-bottom: 8px;
      }

      .rule-body {
        .rule-section {
          display: flex;
          align-items: flex-start;
          gap: 8px;
          margin-bottom: 8px;

          &:last-child {
            margin-bottom: 0;
          }

          .rule-section-label {
            font-size: 12px;
            color: #909399;
            min-width: 40px;
            padding-top: 2px;
          }

          .rule-section-content {
            flex: 1;
            display: flex;
            flex-wrap: wrap;
            gap: 4px;
          }
        }

        .peer-item {
          display: flex;
          align-items: center;
          gap: 4px;
        }

        .peer-selector {
          display: flex;
          align-items: center;
          gap: 4px;

          .peer-labels {
            display: flex;
            flex-wrap: wrap;
            gap: 2px;
          }
        }
      }
    }
  }

  .labels-annotations {
    .la-section {
      display: flex;
      align-items: flex-start;
      gap: 8px;
      margin-bottom: 8px;

      &:last-child {
        margin-bottom: 0;
      }

      .la-label {
        font-size: 13px;
        color: #909399;
        min-width: 80px;
        padding-top: 2px;
      }

      .la-tags {
        flex: 1;
        display: flex;
        flex-wrap: wrap;
        gap: 4px;
      }
    }

    .empty-la {
      color: #909399;
      font-size: 13px;
    }
  }

  @media (max-width: 768px) {
    .info-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
