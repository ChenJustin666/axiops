<template>
  <div class="job-review-step">
    <!-- 验证结果横幅 -->
    <div v-if="!isValid || warnings.length > 0" class="validation-banner">
      <ElAlert v-if="!isValid" type="error" :closable="false" show-icon>
        <template #title>
          <div class="alert-title">发现 {{ errors.length }} 个配置错误</div>
        </template>
        <ul class="error-list">
          <li v-for="(error, index) in errors" :key="index">{{ error }}</li>
        </ul>
      </ElAlert>

      <ElAlert
        v-if="warnings.length > 0"
        type="warning"
        :closable="false"
        show-icon
        style="margin-top: 12px"
      >
        <template #title>
          <div class="alert-title">发现 {{ warnings.length }} 个配置警告</div>
        </template>
        <ul class="warning-list">
          <li v-for="(warning, index) in warnings" :key="index">{{ warning }}</li>
        </ul>
      </ElAlert>
    </div>

    <!-- 配置成功横幅 -->
    <ElAlert v-else type="success" :closable="false" show-icon style="margin-bottom: 16px">
      <template #title>
        <CheckCircle :size="16" style="margin-right: 4px" />
        配置验证通过，可以提交创建
      </template>
    </ElAlert>

    <ElRow :gutter="16">
      <!-- 左列 -->
      <ElCol :xs="24" :lg="12">
        <!-- 基本信息 -->
        <ElCard class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <FileText :size="18" />
              基本信息
            </span>
          </template>

          <ElDescriptions :column="1" border size="default">
            <ElDescriptionsItem label="中文名称" label-class-name="desc-label">
              {{ metadataStore.metadata.nameCn || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="英文名称" label-class-name="desc-label">
              {{ metadataStore.metadata.nameEn || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="版本" label-class-name="desc-label">
              {{ metadataStore.metadata.version || '-' }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="资源名称" label-class-name="desc-label">
              <code>{{ metadataStore.metadata.resourceName || '-' }}</code>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="命名空间" label-class-name="desc-label">
              <ElTag size="small">{{ metadataStore.namespace }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem
              v-if="metadataStore.metadata.desc"
              label="描述"
              label-class-name="desc-label"
            >
              {{ metadataStore.metadata.desc }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>

        <!-- Job 配置 -->
        <ElCard class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <Settings :size="18" />
              Job 配置
            </span>
          </template>

          <ElDescriptions :column="2" border size="default">
            <ElDescriptionsItem label="完成次数" label-class-name="desc-label">
              <ElTag type="primary" size="small">{{ jobStore.completions }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="并行度" label-class-name="desc-label">
              <ElTag type="success" size="small">{{ jobStore.parallelism }}</ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="重试次数" label-class-name="desc-label">
              <ElTag :type="jobStore.backoffLimit === 0 ? 'danger' : 'info'" size="small">
                {{ jobStore.backoffLimit }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="完成模式" label-class-name="desc-label">
              <ElTag
                :type="jobStore.completionMode === 'Indexed' ? 'warning' : 'info'"
                size="small"
              >
                {{ jobStore.completionMode }}
              </ElTag>
            </ElDescriptionsItem>
            <ElDescriptionsItem label="运行时限" label-class-name="desc-label">
              {{
                jobStore.activeDeadlineSeconds
                  ? formatDuration(jobStore.activeDeadlineSeconds)
                  : '无限制'
              }}
            </ElDescriptionsItem>
            <ElDescriptionsItem label="完成后 TTL" label-class-name="desc-label">
              {{
                jobStore.ttlSecondsAfterFinished !== undefined
                  ? formatDuration(jobStore.ttlSecondsAfterFinished)
                  : '不自动清理'
              }}
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>

        <!-- 容器配置 -->
        <ElCard class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <Box :size="18" />
              容器配置
            </span>
          </template>

          <div class="container-summary">
            <div class="summary-item">
              <span class="label">初始化容器:</span>
              <ElTag size="small">{{ containersStore.initContainers.length }} 个</ElTag>
            </div>
            <div class="summary-item">
              <span class="label">主容器:</span>
              <ElTag type="primary" size="small"
                >{{ containersStore.mainContainers.length }} 个</ElTag
              >
            </div>
          </div>

          <ElDivider style="margin: 12px 0" />

          <div
            v-for="container in containersStore.allContainers"
            :key="container.id"
            class="container-item"
          >
            <div class="container-header">
              <ElTag :type="container.type === 'init' ? 'info' : 'primary'" size="small">
                {{ container.type === 'init' ? 'Init' : 'Main' }}
              </ElTag>
              <span class="container-name">{{ container.name }}</span>
            </div>
            <div class="container-body">
              <div class="info-row">
                <span class="label">镜像:</span>
                <code>{{ container.image || '-' }}</code>
              </div>
              <div v-if="container.ports && container.ports.length > 0" class="info-row">
                <span class="label">端口:</span>
                <span>{{ container.ports.map((p) => p.containerPort).join(', ') }}</span>
              </div>
            </div>
          </div>
        </ElCard>
      </ElCol>

      <!-- 右列 -->
      <ElCol :xs="24" :lg="12">
        <!-- 执行预估 -->
        <ElCard class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <TrendingUp :size="18" />
              执行预估
            </span>
          </template>

          <div class="stats-grid">
            <div class="stat-card primary">
              <div class="stat-label">预计总 Pod 数</div>
              <div class="stat-value">{{ estimatedTotalPods }}</div>
              <div class="stat-desc">最多创建的 Pod 数量</div>
            </div>
            <div class="stat-card success">
              <div class="stat-label">预计执行轮次</div>
              <div class="stat-value">{{ estimatedRounds }}</div>
              <div class="stat-desc">理想情况下的轮次</div>
            </div>
            <div class="stat-card warning">
              <div class="stat-label">并发效率</div>
              <div class="stat-value">{{ concurrencyEfficiency }}%</div>
              <div class="stat-desc">并行度利用率</div>
            </div>
          </div>
        </ElCard>

        <!-- 配置建议 -->
        <ElCard v-if="suggestions.length > 0" class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <Lightbulb :size="18" />
              配置建议
            </span>
          </template>

          <div class="suggestions-list">
            <ElAlert
              v-for="(suggestion, index) in suggestions"
              :key="index"
              type="info"
              :closable="false"
              show-icon
            >
              {{ suggestion }}
            </ElAlert>
          </div>
        </ElCard>

        <!-- 存储配置 -->
        <ElCard v-if="volumesStore.volumes.length > 0" class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <HardDrive :size="18" />
              存储配置
            </span>
          </template>

          <div class="volume-list">
            <div v-for="(volume, index) in volumesStore.volumes" :key="index" class="volume-item">
              <ElTag size="small">{{ volume.name }}</ElTag>
              <span class="volume-type">
                {{ getVolumeType(volume) }}
              </span>
            </div>
          </div>
        </ElCard>

        <!-- 调度配置 -->
        <ElCard v-if="hasSchedulingConfig" class="review-card" shadow="never">
          <template #header>
            <span class="card-title">
              <Target :size="18" />
              调度配置
            </span>
          </template>

          <ElDescriptions :column="1" border size="small">
            <ElDescriptionsItem
              v-if="schedulingStore.nodeName"
              label="指定节点"
              label-class-name="desc-label"
            >
              {{ schedulingStore.nodeName }}
            </ElDescriptionsItem>
            <ElDescriptionsItem
              v-if="Object.keys(schedulingStore.nodeSelector).length > 0"
              label="节点选择器"
              label-class-name="desc-label"
            >
              <div
                v-for="(value, key) in schedulingStore.nodeSelector"
                :key="key"
                class="label-item"
              >
                <code>{{ key }}</code> = <code>{{ value }}</code>
              </div>
            </ElDescriptionsItem>
            <ElDescriptionsItem
              v-if="schedulingStore.tolerations.length > 0"
              label="污点容忍"
              label-class-name="desc-label"
            >
              {{ schedulingStore.tolerations.length }} 条规则
            </ElDescriptionsItem>
          </ElDescriptions>
        </ElCard>
      </ElCol>
    </ElRow>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import {
    Settings,
    Box,
    TrendingUp,
    Lightbulb,
    HardDrive,
    Target,
    CheckCircle
  } from 'lucide-vue-next'
  import {
    useMetadataStore,
    useContainersStore,
    useVolumesStore,
    useSchedulingStore,
    validateAllStoresWithJob
  } from '@/store/workload'
  import { useJobStore } from '@/store/workload/job'
  import type { V1Volume } from '@kubernetes/client-node'

  // Props
  interface Props {
    mode?: string
  }

  defineProps<Props>()

  // Stores
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const jobStore = useJobStore()

  // 验证结果
  const validationResult = computed(() => validateAllStoresWithJob())
  const isValid = computed(() => validationResult.value.valid)
  const errors = computed(() => validationResult.value.errors)
  const warnings = computed(() => validationResult.value.warnings || [])

  // 预计总 Pod 数
  const estimatedTotalPods = computed(() => {
    const completions = jobStore.completions || 1
    const backoffLimit = jobStore.backoffLimit || 0
    return completions * (1 + backoffLimit)
  })

  // 预计轮次
  const estimatedRounds = computed(() => {
    const completions = jobStore.completions || 1
    const parallelism = jobStore.parallelism || 1
    return Math.ceil(completions / parallelism)
  })

  // 并发效率
  const concurrencyEfficiency = computed(() => {
    const completions = jobStore.completions || 1
    const parallelism = jobStore.parallelism || 1
    if (parallelism >= completions) {
      return 100
    }
    return Math.round((completions / (estimatedRounds.value * parallelism)) * 100)
  })

  // 是否有调度配置
  const hasSchedulingConfig = computed(() => {
    return !!(
      schedulingStore.nodeName ||
      Object.keys(schedulingStore.nodeSelector).length > 0 ||
      schedulingStore.tolerations.length > 0
    )
  })

  // 配置建议
  const suggestions = computed(() => {
    const result: string[] = []

    if (jobStore.parallelism === 1 && jobStore.completions > 1) {
      result.push('💡 建议增加并行度以提高任务执行效率')
    }

    if (jobStore.backoffLimit > 10) {
      result.push('⚠️ 重试次数过多可能导致失败任务占用过多资源')
    }

    if (jobStore.activeDeadlineSeconds === undefined) {
      result.push('💡 建议设置运行时限，避免任务无限期运行')
    }

    if (jobStore.ttlSecondsAfterFinished === undefined) {
      result.push('💡 建议设置完成后 TTL，自动清理已完成的 Job')
    }

    if (jobStore.completionMode === 'Indexed' && jobStore.completions > 10) {
      if (jobStore.maxFailedIndexes === undefined) {
        result.push('💡 索引模式下建议设置最大失败索引数，提前终止失败的 Job')
      }
    }

    return result
  })

  // 格式化时长
  function formatDuration(seconds: number): string {
    if (seconds === 0) return '0秒'
    if (seconds < 60) {
      return `${seconds}秒`
    } else if (seconds < 3600) {
      const minutes = Math.floor(seconds / 60)
      const remainingSeconds = seconds % 60
      return remainingSeconds > 0 ? `${minutes}分${remainingSeconds}秒` : `${minutes}分钟`
    } else if (seconds < 86400) {
      const hours = Math.floor(seconds / 3600)
      const remainingMinutes = Math.floor((seconds % 3600) / 60)
      return remainingMinutes > 0 ? `${hours}小时${remainingMinutes}分钟` : `${hours}小时`
    } else {
      const days = Math.floor(seconds / 86400)
      const remainingHours = Math.floor((seconds % 86400) / 3600)
      return remainingHours > 0 ? `${days}天${remainingHours}小时` : `${days}天`
    }
  }

  // 获取存储类型
  function getVolumeType(volume: V1Volume): string {
    if (volume.emptyDir) return 'EmptyDir'
    if (volume.hostPath) return 'HostPath'
    if (volume.persistentVolumeClaim) return 'PVC'
    if (volume.configMap) return 'ConfigMap'
    if (volume.secret) return 'Secret'
    if (volume.nfs) return 'NFS'
    return '未知类型'
  }

  // 验证方法
  function validate() {
    return isValid.value
  }

  // 导出
  defineExpose({
    validate
  })
</script>

<style lang="scss" scoped>
  .job-review-step {
    .validation-banner {
      margin-bottom: 16px;

      .alert-title {
        font-weight: 600;
      }

      .error-list,
      .warning-list {
        margin: 8px 0 0 0;
        padding-left: 20px;

        li {
          margin: 4px 0;
          font-size: 13px;
          line-height: 1.6;
        }
      }
    }

    .review-card {
      margin-bottom: 16px;
      border: 1px solid #e4e7ed;

      &:last-child {
        margin-bottom: 0;
      }

      ::v-deep(.el-card__header) {
        padding: 12px 16px;
        background: #fafbfc;
        border-bottom: 1px solid #e4e7ed;
      }

      ::v-deep(.el-card__body) {
        padding: 16px;
      }

      .card-title {
        display: flex;
        align-items: center;
        gap: 8px;
        font-weight: 600;
        font-size: 14px;
        color: #303133;
      }
    }

    ::v-deep(.desc-label) {
      width: 100px;
      font-weight: 500;
      background-color: #fafbfc;
    }

    .container-summary {
      display: flex;
      gap: 16px;

      .summary-item {
        display: flex;
        align-items: center;
        gap: 8px;

        .label {
          font-size: 13px;
          color: #606266;
        }
      }
    }

    .container-item {
      padding: 12px;
      background: #fafbfc;
      border-radius: 4px;
      margin-bottom: 12px;

      &:last-child {
        margin-bottom: 0;
      }

      .container-header {
        display: flex;
        align-items: center;
        gap: 8px;
        margin-bottom: 8px;

        .container-name {
          font-weight: 500;
          font-size: 14px;
        }
      }

      .container-body {
        .info-row {
          display: flex;
          align-items: center;
          gap: 8px;
          margin: 4px 0;
          font-size: 13px;

          .label {
            color: #909399;
            min-width: 40px;
          }

          code {
            font-size: 12px;
            padding: 2px 6px;
            background: white;
            border-radius: 3px;
          }
        }
      }
    }

    .stats-grid {
      display: grid;
      grid-template-columns: repeat(3, 1fr);
      gap: 12px;

      .stat-card {
        padding: 20px;
        text-align: center;
        border-radius: 8px;
        color: white;

        &.primary {
          background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
        }

        &.success {
          background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
        }

        &.warning {
          background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
        }

        .stat-label {
          font-size: 13px;
          opacity: 0.9;
          margin-bottom: 8px;
        }

        .stat-value {
          font-size: 28px;
          font-weight: 700;
          margin-bottom: 8px;
        }

        .stat-desc {
          font-size: 12px;
          opacity: 0.8;
        }
      }
    }

    .suggestions-list {
      ::v-deep(.el-alert) {
        margin-bottom: 12px;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }

    .volume-list {
      .volume-item {
        display: flex;
        align-items: center;
        gap: 8px;
        padding: 8px 12px;
        background: #fafbfc;
        border-radius: 4px;
        margin-bottom: 8px;

        &:last-child {
          margin-bottom: 0;
        }

        .volume-type {
          font-size: 12px;
          color: #909399;
        }
      }
    }

    .label-item {
      display: flex;
      align-items: center;
      gap: 4px;
      margin: 4px 0;

      code {
        font-size: 12px;
        padding: 2px 6px;
        background: #f5f7fa;
        border-radius: 3px;
      }
    }

    @media (max-width: 1024px) {
      .stats-grid {
        grid-template-columns: 1fr;
      }

      ::v-deep(.el-col) {
        margin-bottom: 16px;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }
  }
</style>
