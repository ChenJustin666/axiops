<template>
  <ElDialog
    v-model="visible"
    title="配置 Alertmanager"
    width="900px"
    min-height="800px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div class="config-alert-container">
      <ElAlert type="warning" :closable="false" class="config-tips-alert">
        <template #title>
          <span class="tips-title">
            建议先下载当前配置修改，请保留 <strong>Webhook 接收器配置</strong> 确保告警正常推送
          </span>
        </template>
      </ElAlert>

      <ElForm
        ref="formRef"
        :model="configForm"
        :rules="rules"
        label-width="80px"
        label-position="left"
        class="config-form"
      >
        <div class="form-row">
          <ElFormItem label="目标集群" prop="clusterUuid" class="form-item-half">
            <ElSelect
              v-model="configForm.clusterUuid"
              placeholder="请选择"
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
              />
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

          <ElFormItem label="命名空间" prop="namespace" class="form-item-half">
            <ElSelect
              v-model="configForm.namespace"
              placeholder="请选择"
              :loading="namespaceLoading"
              filterable
              clearable
              :disabled="!configForm.clusterUuid"
              @change="handleNamespaceChange"
              @visible-change="handleNamespaceDropdownVisible"
            >
              <ElOption v-for="ns in namespaceList" :key="ns" :label="ns" :value="ns" />
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="form-row">
          <ElFormItem label="资源类型" prop="crdType" class="form-item-half">
            <ElSelect
              v-model="configForm.crdType"
              placeholder="请选择"
              @change="handleCrdTypeChange"
            >
              <ElOption label="Secret" value="secret" />
              <ElOption label="ConfigMap" value="configmap" />
            </ElSelect>
          </ElFormItem>

          <ElFormItem label="资源名称" prop="name" class="form-item-half">
            <ElSelect
              v-model="configForm.name"
              placeholder="请选择或输入"
              :loading="resourceLoading"
              filterable
              allow-create
              clearable
              :disabled="!configForm.clusterUuid || !configForm.namespace"
              @visible-change="handleResourceDropdownVisible"
            >
              <ElOption v-for="name in resourceNameList" :key="name" :label="name" :value="name" />
            </ElSelect>
          </ElFormItem>
        </div>

        <div class="action-buttons">
          <ElButton
            type="primary"
            plain
            size="small"
            :loading="loadConfigLoading"
            :disabled="!canLoadConfig"
            @click="handleLoadCurrentConfig"
          >
            加载当前配置
          </ElButton>
          <ElButton plain size="small" @click="handleLoadExample"> 加载示例配置 </ElButton>
        </div>

        <div class="editor-wrapper">
          <YamlEditorPro
            v-model="configForm.config"
            filename="alertmanager.yaml"
            :show-toolbar="true"
            :show-line-numbers="true"
            :show-status-bar="true"
            :enable-highlight="true"
            :validate-on-change="true"
            height="500px"
            placeholder="# 请输入 Alertmanager 配置（YAML 格式）..."
            @valid-change="handleValidChange"
          />
        </div>
      </ElForm>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="visible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="submitLoading"
          @click="handleSubmit"
          :disabled="!canSubmit"
        >
          {{ submitLoading ? '提交中...' : '提交配置' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed } from 'vue'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import { getAlertmanagerConfigApi, setAlertmanagerConfigApi } from '@/api/manager/monitor'
  import { searchClusterApi, getClusterNamespaceListApi, type Cluster } from '@/api/manager/cluster'
  import { getNamespaceConfigMapApi, getNamespaceSecretMapApi } from '@/api/workload/core'

  interface Props {
    modelValue: boolean
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ 'update:modelValue': [value: boolean]; success: [] }>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const formRef = ref<FormInstance>()
  const clusterLoading = ref(false)
  const namespaceLoading = ref(false)
  const resourceLoading = ref(false)
  const loadConfigLoading = ref(false)
  const submitLoading = ref(false)
  const isYamlValid = ref(false)

  const clusterList = ref<Cluster[]>([])
  const clusterPagination = reactive({ current: 1, size: 100, total: 0 })
  const namespaceList = ref<string[]>([])
  const resourceNameList = ref<string[]>([])

  const configForm = reactive({
    clusterUuid: '',
    namespace: 'monitoring',
    crdType: 'secret' as 'configmap' | 'secret',
    name: 'alertmanager-main',
    config: ''
  })

  const rules: FormRules = {
    clusterUuid: [{ required: true, message: '请选择集群', trigger: 'change' }],
    namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
  }

  const hasMoreClusters = computed(() => clusterList.value.length < clusterPagination.total)
  const canLoadConfig = computed(() => configForm.clusterUuid && configForm.namespace)
  const canSubmit = computed(
    () =>
      configForm.clusterUuid &&
      configForm.namespace &&
      configForm.config.trim() &&
      isYamlValid.value
  )

  const handleValidChange = (valid: boolean) => {
    isYamlValid.value = valid
  }

  const loadClusterList = async (append = false) => {
    try {
      clusterLoading.value = true
      const res = await searchClusterApi({
        page: clusterPagination.current,
        pageSize: clusterPagination.size,
        orderStr: 'id',
        isAsc: false
      })
      clusterList.value = append ? [...clusterList.value, ...res.items] : res.items
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
    if (visible && configForm.clusterUuid && namespaceList.value.length === 0) {
      loadNamespaceList(configForm.clusterUuid)
    }
  }

  const loadResourceNameList = async () => {
    if (!configForm.clusterUuid || !configForm.namespace) return
    try {
      resourceLoading.value = true
      const res =
        configForm.crdType === 'secret'
          ? await getNamespaceSecretMapApi(configForm.clusterUuid, configForm.namespace)
          : await getNamespaceConfigMapApi(configForm.clusterUuid, configForm.namespace)
      resourceNameList.value = res || []
    } finally {
      resourceLoading.value = false
    }
  }

  const handleResourceDropdownVisible = (visible: boolean) => {
    if (visible) loadResourceNameList()
  }

  const handleClusterChange = async (uuid: string) => {
    configForm.namespace = 'monitoring'
    namespaceList.value = []
    resourceNameList.value = []
    if (uuid) await loadNamespaceList(uuid)
  }

  const handleNamespaceChange = () => {
    resourceNameList.value = []
  }

  const handleCrdTypeChange = () => {
    resourceNameList.value = []
  }

  const handleLoadCurrentConfig = async () => {
    try {
      loadConfigLoading.value = true
      const res = await getAlertmanagerConfigApi({
        clusterUuid: configForm.clusterUuid,
        namespace: configForm.namespace,
        crdType: configForm.crdType,
        name: configForm.name || undefined
      })
      configForm.config = res.config
      ElMessage.success('当前配置加载成功')
    } finally {
      loadConfigLoading.value = false
    }
  }

  const handleLoadExample = () => {
    configForm.config = `global:
  resolve_timeout: 5m

route:
  group_by: ['alertname', 'cluster', 'service']
  group_wait: 30s
  group_interval: 5m
  repeat_interval: 4h
  receiver: 'webhook'
  routes:
    - match:
        severity: critical
      receiver: 'webhook'
      continue: true

receivers:
  - name: 'webhook'
    webhook_configs:
      - url: 'http://10.8.8.6:8811/manager/v1/webhook/alerts'
        send_resolved: true
        max_alerts: 0
        http_config:
          authorization:
            credentials: 'your-secret-token-here-change-me-in-production'

inhibit_rules:
  - source_match:
      severity: 'critical'
    target_match:
      severity: 'warning'
    equal: ['alertname', 'cluster', 'service']`
    ElMessage.success('示例配置已加载')
  }

  const handleSubmit = async () => {
    if (!formRef.value) return

    if (!isYamlValid.value) {
      ElMessage.warning('YAML 语法错误，请先修正')
      return
    }

    try {
      await formRef.value.validate()
      await ElMessageBox.confirm(
        '确定要提交此配置吗？配置将立即生效，请确保已保留 Webhook 接收器配置。',
        '提交确认',
        {
          type: 'warning',
          confirmButtonText: '确定提交',
          cancelButtonText: '取消'
        }
      )

      submitLoading.value = true

      await setAlertmanagerConfigApi({
        clusterUuid: configForm.clusterUuid,
        namespace: configForm.namespace,
        crdType: configForm.crdType,
        name: configForm.name || undefined,
        config: configForm.config
      })

      ElMessage.success('配置提交成功')
      emit('success')
      visible.value = false
    } catch (error: any) {
      if (error === 'cancel') return
    } finally {
      submitLoading.value = false
    }
  }

  const handleClosed = () => {
    configForm.clusterUuid = ''
    configForm.namespace = 'monitoring'
    configForm.crdType = 'secret'
    configForm.name = 'alertmanager-main'
    configForm.config = ''
    isYamlValid.value = false
    clusterList.value = []
    namespaceList.value = []
    resourceNameList.value = []
    clusterPagination.current = 1
    clusterPagination.total = 0
    formRef.value?.clearValidate()
  }
</script>

<style scoped lang="scss">
  .config-alert-container {
    display: flex;
    flex-direction: column;
    gap: 14px;

    .config-tips-alert {
      :deep(.el-alert__content) {
        padding: 0;
      }

      .tips-title {
        font-size: 13px;
        line-height: 1.5;

        strong {
          color: var(--el-color-warning-dark-2);
        }
      }
    }

    .config-form {
      .form-row {
        display: flex;
        gap: 16px;

        .form-item-half {
          flex: 1;
          margin-bottom: 14px;

          :deep(.el-select) {
            width: 100%;
          }
        }
      }

      .select-footer {
        padding: 6px;
        text-align: center;
        border-top: 1px solid var(--el-border-color-lighter);
      }

      .action-buttons {
        display: flex;
        gap: 10px;
        margin-bottom: 14px;
      }

      .editor-wrapper {
        border-radius: 8px;
        overflow: hidden;
      }
    }
  }

  .dialog-footer {
    display: flex;
    justify-content: flex-end;
    gap: 10px;
  }
</style>
