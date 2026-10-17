<template>
  <div v-loading="loading" class="build-settings-page">
    <header class="page-header">
      <div>
        <span class="eyebrow">CI RESOURCE POOL</span>
        <h1>构建资源</h1>
        <p>平台统一管理 CI 集群、构建器、缓存和并发。服务配置只需选择或继承这里的默认值。</p>
      </div>
      <div class="header-actions">
        <ElButton @click="openRegistries">管理镜像仓库</ElButton>
        <ElButton type="primary" :loading="saving" :disabled="loading || !!loadError" @click="save"
          >保存设置</ElButton
        >
      </div>
    </header>

    <ElAlert
      title="代码源与镜像仓库分开管理"
      description="代码源页面只维护 Git 连接；镜像仓库在平台镜像仓库中维护凭据和地址，构建配置页面只选择已授权的仓库。"
      type="info"
      :closable="false"
      show-icon
    />

    <ElAlert v-if="loadError" :title="loadError" type="error" :closable="false" show-icon>
      <ElButton link type="primary" @click="load">重新读取</ElButton>
    </ElAlert>
    <ElForm label-position="top" class="settings-form" :disabled="loading || saving || !!loadError">
      <ElCard shadow="never" class="settings-card">
        <template #header
          ><strong>CI 执行器</strong><span class="card-hint">一次配置，所有服务继承</span></template
        >
        <div class="form-grid">
          <ElFormItem label="启用平台 CI">
            <ElSwitch v-model="form.enabled" active-text="启用" inactive-text="停用" />
          </ElFormItem>
          <ElFormItem label="CI 集群">
            <ElSelect
              v-model="form.cluster_id"
              filterable
              style="width: 100%"
              placeholder="选择专用构建集群"
            >
              <ElOption
                v-for="cluster in clusters"
                :key="cluster.id"
                :label="`${cluster.name} · ${cluster.status}`"
                :value="cluster.id"
              />
            </ElSelect>
          </ElFormItem>
          <ElFormItem label="CI Namespace">
            <ElInput v-model="form.namespace" placeholder="axiops-cicd" />
          </ElFormItem>
          <ElFormItem label="构建器镜像">
            <ElInput
              v-model="form.builder_image"
              placeholder="registry.example.com/axiops/builder:stable"
            />
          </ElFormItem>
        </div>
        <ElFormItem label="默认构建命令">
          <ElInput
            v-model="form.build_command"
            type="textarea"
            :rows="4"
            placeholder="构建器需要输出 image_digest 或 artifact_sha256"
          />
        </ElFormItem>
        <div class="form-grid">
          <ElFormItem label="构建节点标签（可选，每行 key=value）">
            <ElInput
              v-model="nodeSelectorText"
              type="textarea"
              :rows="2"
              placeholder="axiops.io/ci-pool=ci"
            />
          </ElFormItem>
          <ElFormItem label="每个节点最多同时构建数">
            <ElInputNumber v-model="maxBuildsPerNode" :min="0" :max="32" />
            <span class="field-note">0 表示仅按资源调度</span>
          </ElFormItem>
        </div>
        <p class="field-note"
          >填写节点标签后，构建只进入匹配的节点池；管理员需先为这些节点添加标签并完成构建器环境检查。留空兼容现有集群调度。更换集群、Namespace
          或节点策略前，需等待正在执行的任务完成并清理。</p
        >
        <ElCollapse class="advanced-scheduling">
          <ElCollapseItem title="专用节点污点容忍（可选）" name="tolerations">
            <div v-for="(item, index) in tolerations" :key="index" class="form-grid four">
              <ElInput v-model="item.key" placeholder="污点键，例如 dedicated" />
              <ElInput v-model="item.value" placeholder="污点值，例如 ci" />
              <ElSelect v-model="item.effect"
                ><ElOption
                  v-for="effect in ['NoSchedule', 'NoExecute', 'PreferNoSchedule']"
                  :key="effect"
                  :label="effect"
                  :value="effect"
              /></ElSelect>
              <ElButton @click="tolerations.splice(index, 1)">移除</ElButton>
            </div>
            <ElButton @click="tolerations.push({ key: '', value: '', effect: 'NoSchedule' })"
              >添加容忍</ElButton
            >
          </ElCollapseItem>
        </ElCollapse>
      </ElCard>

      <ElCard shadow="never" class="settings-card">
        <template #header
          ><strong>默认资源与缓存</strong><span class="card-hint">服务留空时继承</span></template
        >
        <div class="form-grid four">
          <ElFormItem label="CPU（可选）"
            ><ElInput
              v-model="form.resource_cpu"
              clearable
              placeholder="留空不限制；例如 500m 或 1"
          /></ElFormItem>
          <ElFormItem label="内存（可选）"
            ><ElInput v-model="form.resource_memory" clearable placeholder="留空不限制；例如 2Gi"
          /></ElFormItem>
          <ElFormItem label="构建超时（秒）"
            ><ElInputNumber
              v-model="form.timeout_seconds"
              :min="60"
              :max="86400"
              style="width: 100%"
          /></ElFormItem>
          <ElFormItem label="额外保留 Job 供排查（秒，可选）"
            ><ElInputNumber
              v-model="form.job_retention_seconds"
              :min="0"
              :max="604800"
              style="width: 100%"
          /></ElFormItem>
        </div>
        <p class="field-note"
          >默认 0：成功、失败、超时的构建均在结果与日志保存后立即安排清理 Job 和 Pod。
          页面中的构建记录和日志继续保留。仅在需要额外检查 Pod
          时填写保留时长；取消任务会先保存可读取的输出再停止执行。</p
        >
        <p class="field-note"
          >CPU、内存可分别留空。填写时按该值预留并限制构建资源；留空时平台不预留、不设上限，按需共享节点资源，仍受集群配额和默认策略约束。服务配置中的非空值优先生效。</p
        >
        <div class="form-grid">
          <ElFormItem label="单次构建临时磁盘上限（可选）">
            <ElInput v-model="workspaceSize" clearable placeholder="例如 10Gi，留空不限制" />
          </ElFormItem>
          <ElFormItem label="构建日志保留天数">
            <ElInputNumber v-model="logRetentionDays" :min="0" :max="3650" />
            <span class="field-note">0 表示不自动清理；设定期限后也适用于历史日志</span>
          </ElFormItem>
          <ElFormItem label="依赖缓存 PVC（可选）"
            ><ElInput
              v-model="form.cache_pvc_name"
              placeholder="预先创建在 CI Namespace 的 RWX PVC"
          /></ElFormItem>
          <ElFormItem label="缓存目录"
            ><ElInput v-model="form.cache_dir" placeholder="/workspace/.axiops-cache"
          /></ElFormItem>
        </div>
        <p class="field-note"
          >源码使用独立临时目录。PVC
          按项目和服务隔离，同一服务使用持久缓存时串行构建，避免并发写坏缓存；PVC
          需单独设置容量和维护保留策略。</p
        >
      </ElCard>

      <ElCard shadow="never" class="settings-card">
        <template #header
          ><strong>并发与排队</strong
          ><span class="card-hint">平台默认上限，项目可单独覆盖</span></template
        >
        <div class="form-grid">
          <ElFormItem label="平台最大同时执行任务数"
            ><ElInputNumber
              v-model="concurrency.max_running"
              :min="1"
              :max="1000"
              style="width: 100%"
          /></ElFormItem>
          <ElFormItem label="排队超时（秒）"
            ><ElInputNumber
              v-model="concurrency.queue_timeout_seconds"
              :min="60"
              :max="604800"
              style="width: 100%"
          /></ElFormItem>
        </div>
        <p class="field-note"
          >单台 4 核 8 GB 可先尝试并发 2 个、每个 1 CPU /
          2Gi，再按实际峰值调整。平台和项目上限控制派发数量，节点资源不足时 Pod
          等待调度；排队超时用于未派发任务，构建超时包含 Pod
          等待时间。单节点并发上限采用固定槽位，部分任务可能等待同槽位释放。</p
        >
      </ElCard>
    </ElForm>
  </div>
</template>

<script setup lang="ts">
  import { onMounted, reactive, ref } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage } from 'element-plus'
  import {
    getConcurrencyPolicyApi,
    getPlatformBuildSettingsApi,
    listClustersApi,
    saveConcurrencyPolicyApi,
    savePlatformBuildSettingsApi,
    type AxiOpsCluster,
    type AxiOpsPlatformBuildSettings
  } from '@/api/axiops'

  const router = useRouter()
  const loading = ref(false)
  const saving = ref(false)
  const loadError = ref('')
  const nodeSelectorText = ref('')
  const maxBuildsPerNode = ref(0)
  const workspaceSize = ref('')
  const logRetentionDays = ref(0)
  const tolerations = ref<Array<{ key: string; value: string; effect: string }>>([])
  const clusters = ref<AxiOpsCluster[]>([])
  const defaultBuildCommand = 'exec /usr/local/bin/entrypoint.sh'
  const form = reactive<AxiOpsPlatformBuildSettings>({
    enabled: false,
    cluster_id: 0,
    namespace: 'axiops-cicd',
    builder_image: '',
    build_command: defaultBuildCommand,
    timeout_seconds: 1800,
    resource_cpu: '',
    resource_memory: '',
    cache_pvc_name: '',
    cache_dir: '/workspace/.axiops-cache',
    job_retention_seconds: 0
  })
  const concurrency = reactive({ max_running: 50, queue_timeout_seconds: 7200 })

  const load = async () => {
    loading.value = true
    loadError.value = ''
    try {
      const [settings, policy, clusterList] = await Promise.all([
        getPlatformBuildSettingsApi(),
        getConcurrencyPolicyApi(),
        listClustersApi()
      ])
      if (settings) Object.assign(form, settings)
      if (!form.build_command.trim()) form.build_command = defaultBuildCommand
      nodeSelectorText.value = Object.entries(settings?.execution_policy?.node_selector || {})
        .map(([key, value]) => `${key}=${value}`)
        .join('\n')
      maxBuildsPerNode.value = settings?.execution_policy?.max_builds_per_node || 0
      workspaceSize.value = settings?.execution_policy?.workspace_size || ''
      logRetentionDays.value = settings?.execution_policy?.log_retention_days || 0
      tolerations.value = (settings?.execution_policy?.tolerations || []).map((item) => ({
        ...item
      }))
      if (policy) Object.assign(concurrency, policy)
      clusters.value = clusterList || []
    } catch {
      loadError.value = '构建资源读取失败，重新读取成功后才能保存，避免覆盖已有设置。'
    } finally {
      loading.value = false
    }
  }
  const save = async () => {
    if (loading.value || saving.value || loadError.value) return
    if (!form.build_command.trim()) form.build_command = defaultBuildCommand
    const nodeSelector: Record<string, string> = Object.create(null)
    for (const line of nodeSelectorText.value
      .split('\n')
      .map((value) => value.trim())
      .filter(Boolean)) {
      const separator = line.indexOf('=')
      const key = line.slice(0, separator).trim()
      if (separator < 1 || Object.prototype.hasOwnProperty.call(nodeSelector, key))
        return void ElMessage.warning('节点标签请每行填写 key=value，键不能重复')
      nodeSelector[key] = line.slice(separator + 1).trim()
    }
    if (
      form.enabled &&
      (!form.cluster_id ||
        !form.namespace.trim() ||
        !form.builder_image.trim() ||
        !form.build_command.trim())
    ) {
      return void ElMessage.warning('启用 CI 前请完整填写集群、Namespace、构建器镜像和构建命令')
    }
    if (
      form.resource_memory.trim() &&
      !/^(?:\d+(?:\.\d+)?|\.\d+)(?:Ki|Mi|Gi|Ti|Pi|Ei|k|M|G|T|P|E)$/.test(
        form.resource_memory.trim()
      )
    )
      return void ElMessage.warning('内存请填写带单位的值，例如 2Gi；也可以留空不限制')
    saving.value = true
    try {
      // 仅提交后端接受的字段。load() 会把 updated_by/updated_at 合并进 form，
      // 而后端启用了 DisallowUnknownFields，多余字段会导致"平台 CI 配置格式错误"。
      await savePlatformBuildSettingsApi({
        enabled: form.enabled,
        cluster_id: form.cluster_id,
        namespace: form.namespace.trim(),
        builder_image: form.builder_image.trim(),
        build_command: form.build_command,
        timeout_seconds: form.timeout_seconds,
        resource_cpu: form.resource_cpu.trim(),
        resource_memory: form.resource_memory.trim(),
        cache_pvc_name: form.cache_pvc_name.trim(),
        cache_dir: form.cache_dir.trim(),
        job_retention_seconds: form.job_retention_seconds,
        execution_policy: {
          log_retention_days: logRetentionDays.value,
          node_selector: nodeSelector,
          max_builds_per_node: maxBuildsPerNode.value,
          workspace_size: workspaceSize.value.trim(),
          tolerations: tolerations.value.map((item) => ({
            key: item.key.trim(),
            value: item.value.trim(),
            effect: item.effect
          }))
        }
      })
      try {
        await saveConcurrencyPolicyApi({
          project_id: 0,
          max_running: concurrency.max_running,
          queue_timeout_seconds: concurrency.queue_timeout_seconds
        })
      } catch {
        return void ElMessage.error('执行器设置已保存，但并发设置保存失败；请重试保存。')
      }
      ElMessage.success('平台构建资源已保存，重启 workflow-worker 后生效')
    } finally {
      saving.value = false
    }
  }
  const openRegistries = () => void router.push('/cluster/repositories')
  onMounted(load)
</script>

<style scoped lang="scss">
  .build-settings-page {
    min-height: 100%;
    padding: 24px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .page-header,
  .header-actions,
  :deep(.el-card__header) {
    display: flex;
    align-items: center;
  }
  .page-header {
    justify-content: space-between;
    gap: 24px;
    margin-bottom: 18px;
  }
  h1 {
    margin: 4px 0 6px;
    font-size: 24px;
  }
  p {
    margin: 0;
    color: var(--art-text-gray-500);
  }
  .eyebrow {
    color: var(--el-color-primary);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: 0.12em;
  }
  .header-actions {
    gap: 10px;
  }
  .settings-form {
    max-width: 1100px;
    margin-top: 18px;
  }
  .settings-card {
    margin-top: 14px;
    border: 1px solid var(--el-border-color-lighter);
  }
  :deep(.el-card__header) {
    justify-content: space-between;
  }
  .card-hint {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .form-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 0 18px;
  }
  .form-grid.four {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
  .field-note {
    font-size: 12px;
  }
  @media (max-width: 800px) {
    .page-header,
    .form-grid,
    .form-grid.four {
      grid-template-columns: 1fr;
      display: grid;
    }
    .header-actions {
      justify-content: flex-start;
    }
  }
</style>
