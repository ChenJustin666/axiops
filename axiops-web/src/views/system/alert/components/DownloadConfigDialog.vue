<template>
  <ElDialog
    v-model="visible"
    title="下载监控配置"
    width="900px"
    min-height="800px"
    :close-on-click-modal="false"
    @closed="handleClosed"
  >
    <div class="download-config-container">
      <div class="config-type-section">
        <ElRadioGroup v-model="configType" class="config-type-group">
          <ElRadioButton value="prometheus">
            <span>Prometheus</span>
          </ElRadioButton>
          <ElRadioButton value="alertmanager">
            <span>Alertmanager</span>
          </ElRadioButton>
        </ElRadioGroup>
      </div>

      <ElAlert
        v-if="configType === 'prometheus'"
        type="warning"
        :closable="false"
        class="config-tips-alert"
      >
        <template #title>
          <span class="tips-title">
            请保留 <strong>externalLabels</strong> 外部标签配置，确保告警能正确识别来源集群
          </span>
        </template>
      </ElAlert>

      <ElAlert
        v-if="configType === 'alertmanager'"
        type="warning"
        :closable="false"
        class="config-tips-alert"
      >
        <template #title>
          <span class="tips-title">
            请保留 <strong>Webhook 接收器配置</strong>，确保告警能正常推送到平台
          </span>
        </template>
      </ElAlert>

      <ElForm
        ref="formRef"
        :model="downloadForm"
        :rules="rules"
        label-width="80px"
        label-position="left"
        class="download-form"
      >
        <div class="form-row">
          <ElFormItem label="目标集群" prop="clusterUuid" class="form-item-half">
            <ElSelect
              v-model="downloadForm.clusterUuid"
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
              v-model="downloadForm.namespace"
              placeholder="请选择"
              :loading="namespaceLoading"
              filterable
              clearable
              :disabled="!downloadForm.clusterUuid"
              @change="handleNamespaceChange"
              @visible-change="handleNamespaceDropdownVisible"
            >
              <ElOption v-for="ns in namespaceList" :key="ns" :label="ns" :value="ns" />
            </ElSelect>
          </ElFormItem>
        </div>

        <template v-if="configType === 'alertmanager'">
          <div class="form-row">
            <ElFormItem label="资源类型" prop="crdType" class="form-item-half">
              <ElSelect
                v-model="downloadForm.crdType"
                placeholder="请选择"
                @change="handleCrdTypeChange"
              >
                <ElOption label="Secret" value="secret" />
                <ElOption label="ConfigMap" value="configmap" />
              </ElSelect>
            </ElFormItem>

            <ElFormItem label="资源名称" prop="alertmanagerName" class="form-item-half">
              <ElSelect
                v-model="downloadForm.alertmanagerName"
                placeholder="请选择或输入"
                :loading="resourceLoading"
                filterable
                allow-create
                clearable
                :disabled="!downloadForm.clusterUuid || !downloadForm.namespace"
                @visible-change="handleResourceDropdownVisible"
              >
                <ElOption
                  v-for="name in resourceNameList"
                  :key="name"
                  :label="name"
                  :value="name"
                />
              </ElSelect>
            </ElFormItem>
          </div>
        </template>

        <template v-if="configType === 'prometheus'">
          <div class="form-row">
            <ElFormItem label="资源名称" prop="prometheusName" class="form-item-half">
              <ElInput v-model="downloadForm.prometheusName" placeholder="默认 k8s" clearable />
            </ElFormItem>
          </div>
        </template>
      </ElForm>

      <div class="config-preview">
        <div class="preview-header">
          <span class="preview-title">配置预览</span>
          <ElButton
            size="small"
            type="primary"
            :loading="previewLoading"
            :disabled="!canPreview"
            @click="handlePreview"
          >
            加载预览
          </ElButton>
        </div>
        <div class="preview-content">
          <YamlEditorPro
            v-model="previewConfig"
            filename="config.yaml"
            :readonly="true"
            :show-toolbar="true"
            :show-line-numbers="true"
            :show-status-bar="true"
            :enable-highlight="true"
            :validate-on-change="false"
            height="600px"
            :placeholder="previewPlaceholder"
          />
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="visible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="downloadLoading"
          @click="handleDownload"
          :disabled="!canDownload"
        >
          下载
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, watch } from 'vue'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import YamlEditorPro from '@/components/yaml-editor-pro/index.vue'
  import { getAlertmanagerConfigApi, getPrometheusConfigApi } from '@/api/manager/monitor'
  import { searchClusterApi, getClusterNamespaceListApi, type Cluster } from '@/api/manager/cluster'
  import { getNamespaceConfigMapApi, getNamespaceSecretMapApi } from '@/api/workload/core'

  interface Props {
    modelValue: boolean
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const formRef = ref<FormInstance>()
  const clusterLoading = ref(false)
  const namespaceLoading = ref(false)
  const resourceLoading = ref(false)
  const previewLoading = ref(false)
  const downloadLoading = ref(false)

  const configType = ref<'prometheus' | 'alertmanager'>('prometheus')

  const clusterList = ref<Cluster[]>([])
  const clusterPagination = reactive({ current: 1, size: 100, total: 0 })
  const namespaceList = ref<string[]>([])
  const resourceNameList = ref<string[]>([])
  const previewConfig = ref('')
  const previewPlaceholder = '# 点击【加载预览】查看配置内容'

  const downloadForm = reactive({
    clusterUuid: '',
    namespace: 'monitoring',
    crdType: 'secret' as 'configmap' | 'secret',
    alertmanagerName: 'alertmanager-main',
    prometheusName: 'k8s'
  })

  const rules: FormRules = {
    clusterUuid: [{ required: true, message: '请选择集群', trigger: 'change' }],
    namespace: [{ required: true, message: '请选择命名空间', trigger: 'change' }]
  }

  const hasMoreClusters = computed(() => clusterList.value.length < clusterPagination.total)
  const canPreview = computed(() => downloadForm.clusterUuid && downloadForm.namespace)
  const canDownload = computed(() => downloadForm.clusterUuid && downloadForm.namespace)

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
    if (visible && downloadForm.clusterUuid && namespaceList.value.length === 0) {
      loadNamespaceList(downloadForm.clusterUuid)
    }
  }

  const loadResourceNameList = async () => {
    if (!downloadForm.clusterUuid || !downloadForm.namespace) return
    try {
      resourceLoading.value = true
      const res =
        downloadForm.crdType === 'secret'
          ? await getNamespaceSecretMapApi(downloadForm.clusterUuid, downloadForm.namespace)
          : await getNamespaceConfigMapApi(downloadForm.clusterUuid, downloadForm.namespace)
      resourceNameList.value = res || []
    } finally {
      resourceLoading.value = false
    }
  }

  const handleResourceDropdownVisible = (visible: boolean) => {
    if (visible) loadResourceNameList()
  }

  const handleClusterChange = async (uuid: string) => {
    downloadForm.namespace = 'monitoring'
    namespaceList.value = []
    resourceNameList.value = []
    previewConfig.value = ''
    if (uuid) await loadNamespaceList(uuid)
  }

  const handleNamespaceChange = () => {
    resourceNameList.value = []
    previewConfig.value = ''
  }

  const handleCrdTypeChange = () => {
    resourceNameList.value = []
  }

  const handlePreview = async () => {
    if (!formRef.value) return
    try {
      await formRef.value.validate()
      previewLoading.value = true
      previewConfig.value = ''

      const res =
        configType.value === 'prometheus'
          ? await getPrometheusConfigApi({
              clusterUuid: downloadForm.clusterUuid,
              namespace: downloadForm.namespace,
              name: downloadForm.prometheusName || undefined
            })
          : await getAlertmanagerConfigApi({
              clusterUuid: downloadForm.clusterUuid,
              namespace: downloadForm.namespace,
              crdType: downloadForm.crdType,
              name: downloadForm.alertmanagerName || undefined
            })

      previewConfig.value = res.config
      ElMessage.success('配置加载成功')
    } finally {
      previewLoading.value = false
    }
  }

  const handleDownload = async () => {
    if (!formRef.value) return
    try {
      await formRef.value.validate()
      downloadLoading.value = true

      let res, filename: string
      if (configType.value === 'prometheus') {
        res = await getPrometheusConfigApi({
          clusterUuid: downloadForm.clusterUuid,
          namespace: downloadForm.namespace,
          name: downloadForm.prometheusName || undefined
        })
        filename = `prometheus-${downloadForm.prometheusName || 'k8s'}-config`
      } else {
        res = await getAlertmanagerConfigApi({
          clusterUuid: downloadForm.clusterUuid,
          namespace: downloadForm.namespace,
          crdType: downloadForm.crdType,
          name: downloadForm.alertmanagerName || undefined
        })
        filename = `alertmanager-${downloadForm.alertmanagerName || 'alertmanager-main'}-config`
      }

      const blob = new Blob([res.config], { type: 'text/yaml' })
      const url = URL.createObjectURL(blob)
      const a = document.createElement('a')
      a.href = url
      a.download = `${filename}_${Date.now()}.yaml`
      a.click()
      URL.revokeObjectURL(url)
      ElMessage.success('配置下载成功')
    } finally {
      downloadLoading.value = false
    }
  }

  watch(configType, () => {
    previewConfig.value = ''
  })

  const handleClosed = () => {
    downloadForm.clusterUuid = ''
    downloadForm.namespace = 'monitoring'
    downloadForm.crdType = 'secret'
    downloadForm.alertmanagerName = 'alertmanager-main'
    downloadForm.prometheusName = 'k8s'
    configType.value = 'prometheus'
    previewConfig.value = ''
    clusterList.value = []
    namespaceList.value = []
    resourceNameList.value = []
    clusterPagination.current = 1
    clusterPagination.total = 0
    formRef.value?.clearValidate()
  }
</script>

<style scoped lang="scss">
  .download-config-container {
    display: flex;
    flex-direction: column;
    gap: 14px;

    .config-type-section {
      .config-type-group {
        width: 100%;

        :deep(.el-radio-button) {
          width: 50%;

          .el-radio-button__inner {
            width: 100%;
            padding: 10px 16px;
          }
        }
      }
    }

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

    .download-form {
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
    }

    .config-preview {
      border: 1px solid var(--el-border-color);
      border-radius: 8px;
      overflow: hidden;

      .preview-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 10px 14px;
        background: var(--el-fill-color-light);
        border-bottom: 1px solid var(--el-border-color-lighter);

        .preview-title {
          font-size: 14px;
          font-weight: 500;
          color: var(--el-text-color-primary);
        }
      }

      .preview-content {
        border-radius: 0 0 8px 8px;
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
