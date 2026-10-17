<template>
  <div class="selection-guide-state">
    <!-- 图标区域 -->
    <div class="guide-icon" :class="iconColorClass">
      <component :is="currentStepIcon" :size="iconSize" :stroke-width="1.2" />
    </div>

    <!-- 文字提示区域 -->
    <div class="guide-text">
      <p class="title">{{ currentStepTitle }}</p>
      <p class="description">{{ currentStepDescription }}</p>
    </div>

    <!-- 选择进度指示器 -->
    <div v-if="showProgress && requiredSteps.length > 1" class="selection-progress">
      <div
        v-for="(step, index) in requiredSteps"
        :key="step.key"
        class="progress-item"
        :class="{
          completed: isStepCompleted(step.key),
          current: isCurrentStep(step.key),
          pending: !isStepCompleted(step.key) && !isCurrentStep(step.key)
        }"
      >
        <div class="step-indicator">
          <Check v-if="isStepCompleted(step.key)" :size="12" />
          <span v-else>{{ index + 1 }}</span>
        </div>
        <span class="step-label">{{ step.label }}</span>
        <ChevronRight v-if="index < requiredSteps.length - 1" :size="14" class="step-arrow" />
      </div>
    </div>

    <!-- 操作按钮区域 -->
    <div class="guide-actions">
      <ElButton
        v-if="currentMissingStep?.key === 'project'"
        type="primary"
        @click="handleOpenProjectSelector"
      >
        <FolderSearch :size="16" class="btn-icon" />
        选择项目
      </ElButton>

      <template v-else-if="currentMissingStep">
        <span class="action-hint">
          请在上方选择器中选择
          <strong>{{ currentMissingStep.label }}</strong>
        </span>
      </template>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { computed, type Component } from 'vue'
  import {
    FolderOpen,
    FolderSearch,
    Server,
    Box,
    Package,
    Check,
    ChevronRight,
    Database
  } from 'lucide-vue-next'
  import { mittBus } from '@/utils/sys'

  defineOptions({ name: 'SelectionGuideState' })

  /**
   * 选择步骤配置
   */
  interface SelectionStep {
    key: 'project' | 'cluster' | 'workspace' | 'application' | 'registry'
    label: string
    icon?: Component
    value?: any
  }

  /**
   * 选择层级预设类型
   */
  type SelectionLevel =
    | 'project'
    | 'project-cluster'
    | 'project-cluster-workspace'
    | 'project-cluster-workspace-app'
    | 'project-cluster-registry'
    | 'custom'

  interface Props {
    /** 选择层级预设 */
    level?: SelectionLevel
    /** 自定义步骤配置 */
    customSteps?: SelectionStep[]
    /** 当前已选择的项目 */
    project?: any
    /** 当前已选择的集群 */
    cluster?: any
    /** 当前已选择的工作空间 */
    workspace?: any
    /** 当前已选择的服务 */
    application?: any
    /** 当前已选择的镜像仓库 */
    registry?: any
    /** 图标大小 */
    iconSize?: number
    /** 是否显示进度指示器 */
    showProgress?: boolean
    /** 目标资源名称 */
    resourceName?: string
  }

  const props = withDefaults(defineProps<Props>(), {
    level: 'project-cluster-workspace',
    customSteps: undefined,
    project: undefined,
    cluster: undefined,
    workspace: undefined,
    application: undefined,
    registry: undefined,
    iconSize: 56,
    showProgress: true,
    resourceName: '资源'
  })

  /** 步骤配置映射 */
  const stepConfigMap: Record<string, SelectionStep> = {
    project: { key: 'project', label: '项目', icon: FolderOpen },
    cluster: { key: 'cluster', label: '集群', icon: Server },
    workspace: { key: 'workspace', label: '工作空间', icon: Box },
    application: { key: 'application', label: '服务', icon: Package },
    registry: { key: 'registry', label: '镜像仓库', icon: Database }
  }

  /** 预设层级对应的步骤 */
  const levelStepsMap: Record<Exclude<SelectionLevel, 'custom'>, string[]> = {
    project: ['project'],
    'project-cluster': ['project', 'cluster'],
    'project-cluster-workspace': ['project', 'cluster', 'workspace'],
    'project-cluster-workspace-app': ['project', 'cluster', 'workspace', 'application'],
    'project-cluster-registry': ['project', 'cluster', 'registry']
  }

  /** 获取当前配置的必要选择步骤 */
  const requiredSteps = computed<SelectionStep[]>(() => {
    if (props.level === 'custom' && props.customSteps) {
      return props.customSteps
    }
    const stepKeys = levelStepsMap[props.level as Exclude<SelectionLevel, 'custom'>] || []
    return stepKeys.map((key) => ({
      ...stepConfigMap[key],
      value: getStepValue(key)
    }))
  })

  /** 根据步骤标识获取对应的当前值 */
  function getStepValue(key: string): any {
    const valueMap: Record<string, any> = {
      project: props.project,
      cluster: props.cluster,
      workspace: props.workspace,
      application: props.application,
      registry: props.registry
    }
    return valueMap[key]
  }

  /** 判断步骤是否已完成 */
  function isStepCompleted(key: string): boolean {
    const value = getStepValue(key)
    return value !== null && value !== undefined && value !== ''
  }

  /** 判断是否为当前待完成的步骤 */
  function isCurrentStep(key: string): boolean {
    return currentMissingStep.value?.key === key
  }

  /** 获取当前缺失的第一个步骤 */
  const currentMissingStep = computed<SelectionStep | null>(() => {
    for (const step of requiredSteps.value) {
      if (!isStepCompleted(step.key)) {
        return step
      }
    }
    return null
  })

  /** 当前步骤的图标 */
  const currentStepIcon = computed<Component>(() => {
    if (!currentMissingStep.value) {
      return FolderOpen
    }
    return (
      currentMissingStep.value.icon ||
      stepConfigMap[currentMissingStep.value.key]?.icon ||
      FolderOpen
    )
  })

  /** 图标颜色样式类 */
  const iconColorClass = computed<string>(() => {
    if (!currentMissingStep.value) {
      return 'color-success'
    }
    const colorMap: Record<string, string> = {
      project: 'color-warning',
      cluster: 'color-primary',
      workspace: 'color-info',
      application: 'color-purple',
      registry: 'color-cyan'
    }
    return colorMap[currentMissingStep.value.key] || 'color-default'
  })

  /** 当前步骤的标题文案 */
  const currentStepTitle = computed<string>(() => {
    if (!currentMissingStep.value) {
      return '准备就绪'
    }
    return `请选择${currentMissingStep.value.label}`
  })

  /** 当前步骤的描述文案 */
  const currentStepDescription = computed<string>(() => {
    if (!currentMissingStep.value) {
      return '所有必要选项已完成'
    }
    const step = currentMissingStep.value
    const resourceText = props.resourceName
    const descriptionMap: Record<string, string> = {
      project: `选择一个项目后，即可查看和管理该项目下的${resourceText}`,
      cluster: `在上方选择器中选择集群，以便管理该集群下的${resourceText}`,
      workspace: `选择工作空间后，即可管理该工作空间中的${resourceText}`,
      application: `选择具体的服务后，可查看和管理其${resourceText}详情`,
      registry: `选择镜像仓库后，即可管理该仓库中的${resourceText}`
    }
    return descriptionMap[step.key] || `请先完成${step.label}的选择`
  })

  /** 打开项目选择器 */
  const handleOpenProjectSelector = () => {
    mittBus.emit('openProjectSelector')
  }
</script>

<style lang="scss" scoped>
  .selection-guide-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 48px 24px;
    min-height: 320px;

    .guide-icon {
      margin-bottom: 20px;
      padding: 16px;
      border-radius: 50%;
      background: #f5f7fa;
      transition: all 0.3s ease;

      &.color-warning {
        background: #fdf6ec;
        color: #e6a23c;
      }

      &.color-primary {
        background: #ecf5ff;
        color: #409eff;
      }

      &.color-info {
        background: #f4f4f5;
        color: #909399;
      }

      &.color-purple {
        background: #f5f0ff;
        color: #9b7fea;
      }

      &.color-cyan {
        background: #e6fffb;
        color: #13c2c2;
      }

      &.color-success {
        background: #f0f9eb;
        color: #67c23a;
      }

      &.color-default {
        background: #f5f7fa;
        color: #c0c4cc;
      }
    }

    .guide-text {
      text-align: center;
      margin-bottom: 20px;

      .title {
        margin: 0 0 8px;
        font-size: 16px;
        font-weight: 500;
        color: #303133;
      }

      .description {
        margin: 0;
        font-size: 13px;
        color: #909399;
        line-height: 1.6;
        max-width: 400px;
      }
    }

    .selection-progress {
      display: flex;
      align-items: center;
      gap: 4px;
      margin-bottom: 24px;
      padding: 12px 20px;
      background: #fafbfc;
      border-radius: 8px;
      border: 1px solid #ebeef5;

      .progress-item {
        display: flex;
        align-items: center;
        gap: 6px;

        .step-indicator {
          width: 20px;
          height: 20px;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          font-size: 11px;
          font-weight: 500;
          transition: all 0.2s ease;
        }

        .step-label {
          font-size: 12px;
          white-space: nowrap;
          transition: color 0.2s ease;
        }

        .step-arrow {
          color: #dcdfe6;
          margin: 0 2px;
        }

        &.completed {
          .step-indicator {
            background: #67c23a;
            color: white;
          }
          .step-label {
            color: #67c23a;
          }
        }

        &.current {
          .step-indicator {
            background: #409eff;
            color: white;
            animation: pulse 1.5s infinite;
          }
          .step-label {
            color: #409eff;
            font-weight: 500;
          }
        }

        &.pending {
          .step-indicator {
            background: #ebeef5;
            color: #c0c4cc;
          }
          .step-label {
            color: #c0c4cc;
          }
        }
      }
    }

    .guide-actions {
      .action-hint {
        font-size: 13px;
        color: #606266;
        padding: 10px 20px;
        background: #f5f7fa;
        border-radius: 6px;
        display: inline-block;

        strong {
          color: #409eff;
        }
      }

      .btn-icon {
        margin-right: 6px;
      }
    }
  }

  @keyframes pulse {
    0%,
    100% {
      box-shadow: 0 0 0 0 rgba(64, 158, 255, 0.4);
    }
    50% {
      box-shadow: 0 0 0 6px rgba(64, 158, 255, 0);
    }
  }
</style>
