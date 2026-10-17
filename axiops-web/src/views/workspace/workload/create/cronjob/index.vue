<template>
  <div class="cronjob-create-wizard">
    <!-- 步骤卡片 -->
    <WorkloadStepsCard
      :steps="steps"
      :current-step="currentStep"
      :mode="mode"
      :namespace="currentNamespace"
      resource-type="CronJob"
      v-model:edit-mode="editMode"
      :is-valid="isValid"
      :creating="creating"
      @back="handleBack"
      @submit="handleSave"
      @step-click="goToStep"
    />

    <!-- 主内容卡片 -->
    <ElCard class="wizard-content-card" shadow="hover">
      <template v-if="editMode === 'form'">
        <!-- 步骤内容 -->
        <Transition name="step" mode="out-in">
          <component
            :is="currentStepComponent"
            :key="`${currentStep}-${componentKey}`"
            :mode="mode"
            ref="currentStepRef"
            @validate="handleValidation"
          />
        </Transition>
      </template>

      <!-- YAML模式 -->
      <template v-else>
        <YamlEditor
          v-model="yamlContent"
          :filename="`${metadataStore.metadata.nameEn || 'cronjob'}.yaml`"
          :readonly="false"
          height="600px"
          @change="handleYamlChange"
          @save="handleYamlSave"
        />
      </template>
    </ElCard>
  </div>
</template>

<script setup lang="ts">
  import {
    ref,
    computed,
    watch,
    provide,
    onMounted,
    onBeforeUnmount,
    shallowRef,
    nextTick
  } from 'vue'
  import { useRouter, useRoute } from 'vue-router'
  import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
  import YamlEditor from '@/components/yaml-editor-pro/index.vue'
  import yaml from 'js-yaml'
  import WorkloadStepsCard from '../common/components/WorkloadStepsCard.vue'

  // 导入 Store
  import {
    useMetadataStore,
    // useContainersStore,
    // useVolumesStore,
    // useSchedulingStore,
    // useAdvancedStore,
    resetAllWorkloadStores,
    generateCronJobFromStores,
    updateStoresFromCronJob,
    validateAllStoresWithCronJob,
    useContainersStore
  } from '@/store/workload'

  // 导入 API
  import { ResourceType } from '@/api/workload'

  // 导入工具函数
  import {
    validateRouteParams,
    initWorkloadByMode,
    isFieldEditable,
    ensureNamespace,
    ensureRequiredLabelsInResource,
    type RouteParams
  } from '../common/utils/modeHandler'

  // 导入统一提交处理
  import { submitWorkload, submitWorkloadFromYaml } from '../common/utils/submitHandler'

  // 步骤组件
  import MetadataStep from '../common/components/MetadataStep.vue'
  import CronJobConfigStep from './components/CronJobConfigStep.vue'
  import ContainersStep from '../common/components/ContainersStep.vue'
  import VolumesStep from '../common/components/VolumesStep.vue'
  import SchedulingStep from '../common/components/SchedulingStep.vue'
  import AdvancedStep from '../common/components/AdvancedStep.vue'
  import ReviewStep from './components/ReviewStep.vue'

  // 初始化 Store
  const metadataStore = useMetadataStore()
  // const containersStore = useContainersStore()
  // const volumesStore = useVolumesStore()
  // const schedulingStore = useSchedulingStore()
  // const advancedStore = useAdvancedStore()
  // const cronJobStore = useCronJobStore()

  const router = useRouter()
  const route = useRoute()

  // 路由参数
  const routeParams = ref<RouteParams | null>(null)
  const mode = computed(() => routeParams.value?.mode || 'createApp')
  const resourceClusterId = computed(() => routeParams.value?.resourceClusterId || 0)
  const clusterUuid = computed(() => routeParams.value?.clusterUuid || '')
  const workspaceId = computed(() => routeParams.value?.workspaceId || 0)
  const currentNamespace = computed(() => routeParams.value?.namespace || 'default')
  const applicationId = computed(() => routeParams.value?.applicationId)
  const applicationVersionId = computed(() => routeParams.value?.applicationVersionId)

  // 编辑模式
  const editMode = ref<'form' | 'yaml'>('form')

  // 当前步骤
  const currentStep = ref(0)
  const currentStepRef = ref<any>(null)

  const containersStore = useContainersStore()

  // 创建状态
  const creating = ref(false)

  // 数据是否已修改
  const hasUnsavedChanges = ref(false)

  // 组件key，用于强制刷新
  const componentKey = ref(0)

  // 原始 YAML 对象（用于编辑模式下保留未被表单管理的字段）
  const originalYamlObj = ref<any>(null)

  // 步骤配置接口
  interface StepConfig {
    id: string
    title: string
    description: string
    component: any
    hasError: boolean
    validated: boolean
    touched: boolean
    required: boolean
  }

  // 验证结果接口
  interface ValidationResult {
    valid: boolean
    errors: string[]
    warnings?: string[]
  }

  // ⭐ 步骤配置 - 存储配置在容器配置之前
  const steps = shallowRef<StepConfig[]>([
    {
      id: 'metadata',
      title: '元数据配置',
      description: 'CronJob名称和标签',
      component: MetadataStep,
      hasError: true,
      validated: false,
      touched: false,
      required: true
    },
    {
      id: 'cronjob',
      title: 'CronJob配置',
      description: '定时任务调度配置',
      component: CronJobConfigStep,
      hasError: true,
      validated: false,
      touched: false,
      required: true
    },
    {
      id: 'volumes',
      title: '存储配置',
      description: '配置存储卷（供容器挂载）',
      component: VolumesStep,
      hasError: false,
      validated: true,
      touched: false,
      required: false
    },
    {
      id: 'containers',
      title: '容器配置',
      description: '配置容器镜像和资源',
      component: ContainersStep,
      hasError: true,
      validated: false,
      touched: false,
      required: true
    },
    {
      id: 'scheduling',
      title: '调度策略',
      description: '节点选择和亲和性',
      component: SchedulingStep,
      hasError: false,
      validated: true,
      touched: false,
      required: false
    },
    {
      id: 'advanced',
      title: '高级配置',
      description: '安全上下文和网络',
      component: AdvancedStep,
      hasError: false,
      validated: true,
      touched: false,
      required: false
    },
    {
      id: 'review',
      title: '确认配置',
      description: '检查并确认',
      component: ReviewStep,
      hasError: false,
      validated: true,
      touched: false,
      required: false
    }
  ])

  // 当前步骤组件
  const currentStepComponent = computed(() => {
    return steps.value[currentStep.value].component
  })

  // 整体是否有效
  const isValid = computed(() => {
    return steps.value
      .filter((step) => step.required)
      .every((step) => step.validated && !step.hasError)
  })

  // YAML内容
  const yamlContent = ref('')

  // 字段是否可编辑
  const canEditField = (field: 'nameCn' | 'nameEn' | 'version' | 'resourceName' | 'desc') => {
    return isFieldEditable(mode.value, field)
  }

  // 提供给子组件
  provide('namespace', currentNamespace)
  provide('resourceClusterId', resourceClusterId)
  provide('clusterUuid', clusterUuid)
  provide('workspaceId', workspaceId)
  provide('mode', mode)
  provide('canEditField', canEditField)

  // ==================== 验证逻辑 ====================

  // 处理验证
  const handleValidation = (result: ValidationResult) => {
    const currentSteps = [...steps.value]
    const step = currentSteps[currentStep.value]

    step.validated = result.valid
    step.hasError = result.errors.length > 0
    step.touched = true

    steps.value = currentSteps

    if (step.touched) {
      hasUnsavedChanges.value = true
    }
  }

  // 验证当前步骤
  const validateCurrentStep = async (): Promise<boolean> => {
    const step = steps.value[currentStep.value]

    if (!currentStepRef.value?.validate) {
      if (!step.required) {
        const currentSteps = [...steps.value]
        currentSteps[currentStep.value].validated = true
        currentSteps[currentStep.value].hasError = false
        steps.value = currentSteps
        return true
      }

      return false
    }

    try {
      const result = await currentStepRef.value.validate()

      if (typeof result === 'boolean') {
        const validationResult = {
          valid: result,
          errors: result ? [] : ['验证失败']
        }
        handleValidation(validationResult)
        return result
      }

      handleValidation(result)
      return result.valid
    } catch (error) {
      console.error(`❌ 步骤 ${step.title} 验证异常:`, error)
      handleValidation({ valid: false, errors: ['验证过程发生错误'] })
      return false
    }
  }

  // ==================== 步骤导航 ====================

  // 跳转到指定步骤
  const goToStep = async (index: number) => {
    // 验证当前步骤
    if (currentStep.value !== index) {
      await validateCurrentStep()
    }

    const currentSteps = [...steps.value]
    currentSteps[currentStep.value].touched = true
    steps.value = currentSteps

    currentStep.value = index
  }

  // ==================== 保存逻辑 ====================

  /**
   * 智能合并原始 YAML 和表单数据
   * 策略：表单管理的字段使用表单数据，其他字段保留原始值
   * CronJob 结构：spec.jobTemplate.spec.template.spec
   */
  const mergeWithOriginalYaml = (formData: any): any => {
    if (!originalYamlObj.value || mode.value !== 'editAppVersion') {
      return formData
    }

    const original = originalYamlObj.value

    // 深度合并 metadata
    const mergedMetadata = {
      ...original.metadata,
      ...formData.metadata,
      name: formData.metadata.name,
      namespace: formData.metadata.namespace,
      labels: formData.metadata.labels,
      annotations: formData.metadata.annotations
    }
    delete mergedMetadata.uid
    delete mergedMetadata.resourceVersion
    delete mergedMetadata.generation
    delete mergedMetadata.creationTimestamp
    delete mergedMetadata.managedFields

    // CronJob 的 Pod Spec 路径：spec.jobTemplate.spec.template.spec
    const originalPodSpec = original.spec?.jobTemplate?.spec?.template?.spec || {}
    const formPodSpec = formData.spec?.jobTemplate?.spec?.template?.spec || {}

    // 合并容器：基于 name 匹配
    const mergedContainers =
      formPodSpec.containers?.map((formContainer: any) => {
        const originalContainer = originalPodSpec.containers?.find(
          (c: any) => c.name === formContainer.name
        )
        if (originalContainer) {
          return { ...originalContainer, ...formContainer }
        }
        return formContainer
      }) || []

    // 合并 initContainers
    const mergedInitContainers = formPodSpec.initContainers?.map((formContainer: any) => {
      const originalContainer = originalPodSpec.initContainers?.find(
        (c: any) => c.name === formContainer.name
      )
      if (originalContainer) {
        return { ...originalContainer, ...formContainer }
      }
      return formContainer
    })

    // 合并 Pod Spec
    const mergedPodSpec = {
      ...originalPodSpec,
      ...formPodSpec,
      containers: mergedContainers
    }
    if (mergedInitContainers && mergedInitContainers.length > 0) {
      mergedPodSpec.initContainers = mergedInitContainers
    }

    // 合并 jobTemplate
    const mergedJobTemplate = {
      ...original.spec?.jobTemplate,
      ...formData.spec?.jobTemplate,
      spec: {
        ...original.spec?.jobTemplate?.spec,
        ...formData.spec?.jobTemplate?.spec,
        template: {
          ...original.spec?.jobTemplate?.spec?.template,
          ...formData.spec?.jobTemplate?.spec?.template,
          metadata: formData.spec?.jobTemplate?.spec?.template?.metadata,
          spec: mergedPodSpec
        }
      }
    }

    // 合并 spec
    const mergedSpec = {
      ...original.spec,
      ...formData.spec,
      jobTemplate: mergedJobTemplate
    }

    return {
      apiVersion: formData.apiVersion,
      kind: formData.kind,
      metadata: mergedMetadata,
      spec: mergedSpec
    }
  }

  // 表单模式保存
  const handleSave = async () => {
    // 使用统一的验证函数验证所有配置
    const validationResult = validateAllStoresWithCronJob()

    if (!validationResult.valid) {
      console.error('❌ 配置验证失败:', validationResult.errors)

      // 显示具体错误信息
      if (validationResult.errors.length > 0) {
        const errorMsg = validationResult.errors.slice(0, 3).join('\n')
        ElMessage.error({
          message: errorMsg,
          duration: 5000,
          showClose: true
        })
      }
      return
    }

    // 显示警告信息（如果有）
    if (validationResult.warnings && validationResult.warnings.length > 0) {
      console.warn('⚠️ 配置警告:', validationResult.warnings)
    }

    creating.value = true

    try {
      // createNewService 模式下，获取用户选择的目标位置
      const targetLocation = metadataStore.getTargetLocation()
      const finalClusterId =
        mode.value === 'createNewService' && targetLocation
          ? targetLocation.clusterId
          : resourceClusterId.value
      const finalClusterUuid =
        mode.value === 'createNewService' && targetLocation
          ? targetLocation.clusterUuid
          : clusterUuid.value
      const finalWorkspaceId =
        mode.value === 'createNewService' && targetLocation
          ? targetLocation.workspaceId
          : workspaceId.value
      const finalNamespace =
        mode.value === 'createNewService' && targetLocation
          ? targetLocation.namespace
          : currentNamespace.value

      // 使用统一的生成函数生成 CronJob 对象
      let cronJob = generateCronJobFromStores()

      // 编辑模式下与原始 YAML 合并，保留未被表单管理的字段
      cronJob = mergeWithOriginalYaml(cronJob)

      // 确保 namespace 和标签正确
      ensureNamespace(cronJob, finalNamespace)
      ensureRequiredLabelsInResource(
        cronJob,
        metadataStore.metadata.nameEn,
        metadataStore.metadata.version
      )

      // 生成YAML字符串
      const yamlStr = yaml.dump(cronJob, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })

      // 调用统一提交函数
      const result = await submitWorkload({
        mode: mode.value,
        resourceType: ResourceType.CRONJOB,
        resourceClusterId: finalClusterId,
        clusterUuid: finalClusterUuid,
        workspaceId: finalWorkspaceId,
        namespace: finalNamespace,
        nameCn: metadataStore.metadata.nameCn,
        resourceName: metadataStore.metadata.resourceName,
        nameEn: metadataStore.metadata.nameEn,
        version: metadataStore.metadata.version,
        description: metadataStore.metadata.desc,
        resourceYamlStr: yamlStr,
        applicationId: applicationId.value,
        applicationVersionId: applicationVersionId.value
      })

      if (result.success) {
        clearAllData()
        navigateToList()
      }
    } catch (error) {
      console.error('💥 保存 CronJob 失败:', error)
    } finally {
      creating.value = false
    }
  }

  // YAML模式保存
  const handleYamlSave = async () => {
    creating.value = true

    try {
      // ✅ 编辑模式下的无损更新策略
      let finalYamlStr = yamlContent.value
      let resourceNameFromYaml = ''

      if (mode.value === 'editAppVersion') {
        // 编辑模式：需要合并原始 YAML 的字段（如 status、managedFields 等）
        try {
          const currentYamlObj = yaml.load(yamlContent.value) as any
          resourceNameFromYaml = currentYamlObj?.metadata?.name || ''

          // 重新生成表单数据作为基础
          const formDataObj = generateCronJobFromStores()

          // ✅ 深度合并：保留原始 YAML 中表单没有的字段（如 status、managedFields 等）
          // 策略：formData 覆盖 metadata 和 spec，其他字段保留
          const mergedObj: any = {
            apiVersion: currentYamlObj.apiVersion || formDataObj.apiVersion,
            kind: currentYamlObj.kind || formDataObj.kind,
            metadata: {
              ...currentYamlObj.metadata, // 保留原始 metadata 中的所有字段
              ...formDataObj.metadata, // 表单数据覆盖
              name: formDataObj.metadata.name, // 确保资源名正确
              namespace: currentNamespace.value
            },
            spec: formDataObj.spec // spec 完全由表单生成
          }

          // 保留原始 YAML 中 spec 之外的其他字段（如 status）
          if (currentYamlObj.status) {
            mergedObj.status = currentYamlObj.status
          }

          // 重新生成 YAML
          finalYamlStr = yaml.dump(mergedObj, {
            indent: 2,
            lineWidth: -1,
            noRefs: true,
            sortKeys: false
          })
        } catch (mergeError) {
          console.error('合并 YAML 失败，使用原始内容:', mergeError)
          // 合并失败时使用原始 YAML
          const parsedCronJob = yaml.load(yamlContent.value) as any
          resourceNameFromYaml = parsedCronJob?.metadata?.name || ''
        }
      } else {
        // 非编辑模式：直接解析 YAML 获取 resourceName
        try {
          const parsedCronJob = yaml.load(yamlContent.value) as any
          resourceNameFromYaml = parsedCronJob?.metadata?.name || ''
        } catch (error) {
          console.error('解析 YAML 失败:', error)
        }
      }

      const result = await submitWorkloadFromYaml(
        {
          mode: mode.value,
          resourceType: ResourceType.CRONJOB,
          resourceClusterId: resourceClusterId.value,
          clusterUuid: clusterUuid.value,
          workspaceId: workspaceId.value,
          namespace: currentNamespace.value,
          applicationId: applicationId.value,
          applicationVersionId: applicationVersionId.value
        },
        finalYamlStr,
        {
          nameCn: metadataStore.metadata.nameCn,
          resourceName: resourceNameFromYaml || metadataStore.metadata.resourceName,
          nameEn: metadataStore.metadata.nameEn,
          version: metadataStore.metadata.version,
          desc: metadataStore.metadata.desc
        }
      )

      if (result.success) {
        clearAllData()
        navigateToList()
      }
    } catch (error) {
      console.error('💥 从 YAML 保存失败:', error)
    } finally {
      creating.value = false
    }
  }

  // YAML变化处理
  const handleYamlChange = (value: string) => {
    yamlContent.value = value
    hasUnsavedChanges.value = true
  }

  // ==================== 页面导航 ====================

  // 返回
  const handleBack = async () => {
    await confirmExit(() => {
      router.back()
    })
  }

  // 导航到列表页
  const navigateToList = () => {
    router.push({
      name: 'WorkspaceApp',
      query: {
        resourceClusterId: resourceClusterId.value,
        clusterUuid: clusterUuid.value,
        workspaceId: workspaceId.value,
        namespace: currentNamespace.value
      }
    })
  }

  // 退出确认
  const confirmExit = async (callback: () => void) => {
    const message = hasUnsavedChanges.value
      ? '您有未保存的更改，确定要离开吗？所有未保存的配置将丢失。'
      : '确定要离开当前页面吗？'

    try {
      await ElMessageBox.confirm(message, '确认离开', {
        confirmButtonText: '确定离开',
        cancelButtonText: '继续编辑',
        type: 'warning',
        distinguishCancelAndClose: true,
        beforeClose: (action, instance, done) => {
          if (action === 'confirm') {
            clearAllData()
            done()
          } else {
            done()
          }
        }
      })
      window.removeEventListener('beforeunload', handleBeforeUnload)
      callback()
    } catch {
      // 用户取消
    }
  }

  // 清理所有数据
  const clearAllData = () => {
    resetAllWorkloadStores()

    hasUnsavedChanges.value = false
    currentStep.value = 0
    editMode.value = 'form'
    yamlContent.value = ''
    originalYamlObj.value = null

    steps.value.forEach((step) => {
      step.touched = false
      if (step.required) {
        step.hasError = true
        step.validated = false
      } else {
        step.hasError = false
        step.validated = true
      }
    })
  }

  // 处理页面离开前的确认
  const handleBeforeUnload = (e: BeforeUnloadEvent) => {
    if (hasUnsavedChanges.value) {
      e.preventDefault()
      e.returnValue = '您有未保存的更改，确定要离开吗？'
    }
  }

  // ==================== YAML 配置加载 ====================

  /**
   * 加载 YAML 配置并填充到各个 store
   * @param versionId 版本ID
   * @param mode 工作模式
   */
  const loadYamlConfig = async (versionId: number, mode: string) => {
    // ✅ 添加全局 loading 状态
    const loadingInstance = ElLoading.service({
      lock: true,
      text: '正在加载配置...',
      background: 'rgba(0, 0, 0, 0.7)'
    })

    try {
      // ✅ 编辑模式和复制模式下，从 API 获取的元数据优先（不允许修改 ns、nameCn、nameEn、version）
      const savedMetadata = {
        nameCn: metadataStore.metadata.nameCn,
        nameEn: metadataStore.metadata.nameEn,
        version: metadataStore.metadata.version,
        resourceName: metadataStore.metadata.resourceName,
        desc: metadataStore.metadata.desc,
        namespace: currentNamespace.value
      }

      const { loadWorkloadYamlForEdit } = await import('../common/utils/modeHandler')
      const yamlStr = await loadWorkloadYamlForEdit(versionId)

      if (!yamlStr) {
        throw new Error('YAML 配置为空')
      }

      const documents = yaml.loadAll(yamlStr)

      if (documents.length === 0) {
        throw new Error('YAML 内容为空')
      }

      if (documents.length > 1) {
        console.warn(`检测到 ${documents.length} 个 YAML 文档，将使用第一个 CronJob`)
      }

      const parsedCronJob = documents[0] as any

      if (!parsedCronJob || parsedCronJob.kind !== 'CronJob') {
        throw new Error(`期望 CronJob 类型，但得到: ${parsedCronJob?.kind || '未知'}`)
      }

      // 编辑模式下保存原始 YAML 对象（深拷贝）
      if (mode === 'editAppVersion') {
        originalYamlObj.value = JSON.parse(JSON.stringify(parsedCronJob))
      }

      // 从 YAML 中提取原始 resourceName
      const parsedResourceName = parsedCronJob.metadata?.name || ''

      // ✅ 确保 namespace 正确（编辑模式和复制模式下使用当前 namespace）
      if (parsedCronJob.metadata) {
        parsedCronJob.metadata.namespace = currentNamespace.value
      }

      updateStoresFromCronJob(parsedCronJob)

      // ✅ 根据模式决定最终的元数据（关键：编辑模式和复制模式下 ns、nameCn、nameEn、version 不允许修改）
      let finalResourceName = savedMetadata.resourceName
      if (mode === 'editAppVersion') {
        // 编辑模式：使用 YAML 中的原始 resourceName，不允许修改
        finalResourceName = parsedResourceName
      } else if (mode === 'copyAppVersion') {
        // 复制模式：生成新的 resourceName
        finalResourceName = `${savedMetadata.nameEn}-${savedMetadata.version}`
      } else if (mode === 'createNewService') {
        // 创建新服务模式：所有字段可编辑，resourceName 自动生成
        finalResourceName = `${savedMetadata.nameEn}-${savedMetadata.version}`
      }

      // ✅ 更新元数据（编辑模式和复制模式下使用 API 返回的数据，不允许用户修改）
      // createNewService 模式下所有字段可编辑，使用 API 返回的数据作为默认值
      metadataStore.updateMetadata({
        nameCn: savedMetadata.nameCn, // createNewService 模式下可编辑
        nameEn: savedMetadata.nameEn, // createNewService 模式下可编辑
        version: savedMetadata.version, // createNewService 模式下可编辑
        resourceName: finalResourceName,
        desc: savedMetadata.desc
      })

      // 确保 namespace 正确
      metadataStore.setNamespace(savedMetadata.namespace)

      // 强制刷新组件
      componentKey.value++

      // 等待组件更新完成
      await nextTick()

      ElMessage.success('配置加载完成')
    } catch (error) {
      console.error('加载完整配置失败:', error)
      ElMessage.error({
        message: `加载配置失败：${error instanceof Error ? error.message : '未知错误'}`,
        duration: 5000,
        showClose: true
      })
    } finally {
      loadingInstance.close()
    }
  }

  // ==================== 生命周期 ====================

  onMounted(async () => {
    const params = validateRouteParams(route.query)
    if (!params) {
      setTimeout(() => {
        router.back()
      }, 1500)
      return
    }

    routeParams.value = params

    // 初始化基础元数据
    const success = await initWorkloadByMode(params)

    if (!success) {
      setTimeout(() => {
        router.back()
      }, 1500)
      return
    }

    // 如果是编辑、复制或创建新服务模式，加载完整的 YAML 配置
    if (
      (params.mode === 'editAppVersion' ||
        params.mode === 'copyAppVersion' ||
        params.mode === 'createNewService') &&
      params.applicationVersionId
    ) {
      await loadYamlConfig(params.applicationVersionId, params.mode)
    }

    hasUnsavedChanges.value = false
    window.addEventListener('beforeunload', handleBeforeUnload)
  })

  // 监听编辑模式切换
  watch(editMode, async (newMode, oldMode) => {
    if (newMode === 'yaml') {
      // 切换到 YAML 模式：从 store 生成 YAML
      let cronJob = generateCronJobFromStores()
      // 编辑模式下与原始 YAML 合并，保留未被表单管理的字段
      cronJob = mergeWithOriginalYaml(cronJob)
      yamlContent.value = yaml.dump(cronJob, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })
    } else if (newMode === 'form' && oldMode === 'yaml') {
      // 从 YAML 模式切换回表单模式
      if (yamlContent.value && yamlContent.value.trim()) {
        try {
          const documents = yaml.loadAll(yamlContent.value)

          if (documents.length === 0) {
            ElMessage.error('YAML 内容为空')
            editMode.value = 'yaml'
            return
          }

          if (documents.length > 1) {
            ElMessage.error({
              message: `检测到 ${documents.length} 个 YAML 文档，CronJob 页面只能包含一个 CronJob 类型的 YAML`,
              duration: 5000,
              showClose: true
            })
            editMode.value = 'yaml'
            return
          }

          const parsedCronJob = documents[0] as any

          if (!parsedCronJob || !parsedCronJob.kind) {
            ElMessage.error('YAML 格式不正确')
            editMode.value = 'yaml'
            return
          }

          if (parsedCronJob.kind !== 'CronJob') {
            ElMessage.error({
              message: `当前页面只能处理 CronJob 类型的 YAML，但检测到 kind: ${parsedCronJob.kind}`,
              duration: 5000,
              showClose: true
            })
            editMode.value = 'yaml'
            return
          }

          // 从 YAML 中提取元数据
          const yamlMetadata = parsedCronJob.metadata || {}
          const yamlLabels = yamlMetadata.labels || {}
          const yamlAnnotations = yamlMetadata.annotations || {}
          const yamlResourceName = parsedCronJob.metadata?.name || ''

          if (!yamlResourceName) {
            ElMessage.error({
              message: 'YAML 缺少 metadata.name 字段',
              duration: 5000,
              showClose: true
            })
            editMode.value = 'yaml'
            return
          }

          // 从 YAML 中解析字段
          const parsedNameEn = yamlLabels.app || ''
          const parsedVersion = yamlLabels.version || ''
          const parsedDesc =
            yamlAnnotations.description || yamlAnnotations['ikubeops.com/description'] || ''
          const parsedNameCn = yamlAnnotations['ikubeops.com/service-name'] || ''

          // 先取消选中容器
          containersStore.selectContainer(null)

          // 切换到第一步
          currentStep.value = 0

          // 等待组件卸载完成
          await nextTick()

          // 更新 namespace
          if (parsedCronJob.metadata) {
            parsedCronJob.metadata.namespace = currentNamespace.value
          }

          // 加载 YAML 到 stores
          updateStoresFromCronJob(parsedCronJob)

          // 使用从 YAML 解析的数据更新元数据
          metadataStore.updateMetadata({
            nameCn: parsedNameCn,
            nameEn: parsedNameEn,
            version: parsedVersion,
            resourceName: yamlResourceName,
            desc: parsedDesc
          })

          // 手动标记 resourceName
          metadataStore.setResourceName(yamlResourceName)

          // 设置命名空间
          metadataStore.setNamespace(currentNamespace.value)

          // 强制刷新组件
          componentKey.value++

          // 等待渲染完成
          await nextTick()
          await nextTick()

          // 更新步骤验证状态
          const currentSteps = [...steps.value]

          // 元数据配置
          const metadataValid = parsedNameCn && parsedNameEn && parsedVersion && yamlResourceName
          currentSteps[0].validated = !!metadataValid
          currentSteps[0].hasError = !metadataValid
          currentSteps[0].touched = true

          // CronJob 配置 - 默认通过
          currentSteps[1].validated = true
          currentSteps[1].hasError = false
          currentSteps[1].touched = true

          // 容器配置
          const containersResult = containersStore.validate()
          currentSteps[3].validated = containersResult.valid
          currentSteps[3].hasError = !containersResult.valid
          currentSteps[3].touched = true

          // 其他非必填步骤默认通过
          currentSteps[2].validated = true // 存储配置
          currentSteps[2].hasError = false
          currentSteps[4].validated = true // 调度策略
          currentSteps[4].hasError = false
          currentSteps[5].validated = true // 高级配置
          currentSteps[5].hasError = false
          currentSteps[6].validated = true // 确认配置
          currentSteps[6].hasError = false

          steps.value = currentSteps

          // 根据中文名是否为空给出不同提示
          if (!parsedNameCn) {
            ElMessage.warning({
              message: '已从 YAML 同步数据，请补充填写中文名',
              duration: 4000,
              showClose: true
            })
          } else {
            ElMessage.success('已从 YAML 同步数据')
          }
        } catch (error) {
          console.error('解析 YAML 失败:', error)
          ElMessage.error({
            message: `YAML 格式错误：${error instanceof Error ? error.message : '未知错误'}`,
            duration: 5000,
            showClose: true
          })
          editMode.value = 'yaml'
        }
      }
    }
  })

  // 监听路由变化，重新加载配置
  watch(
    () => route.query,
    async (newQuery, oldQuery) => {
      const newVersionId = Number(newQuery.applicationVersionId)
      const oldVersionId = Number(oldQuery?.applicationVersionId)
      const newMode = newQuery.mode as string

      // 如果是编辑、复制或创建新服务模式，且版本ID变化了
      if (
        (newMode === 'editAppVersion' ||
          newMode === 'copyAppVersion' ||
          newMode === 'createNewService') &&
        newVersionId !== oldVersionId &&
        newVersionId > 0
      ) {
        const params = validateRouteParams(newQuery)
        if (params) {
          routeParams.value = params
          await initWorkloadByMode(params)
          await loadYamlConfig(newVersionId, newMode)
        }
      }
    },
    { deep: true }
  )

  // 组件卸载时清理
  onBeforeUnmount(() => {
    window.removeEventListener('beforeunload', handleBeforeUnload)
  })
</script>

<style lang="scss" scoped>
  .cronjob-create-wizard {
    .wizard-content-card {
      min-height: 420px;

      ::v-deep(.el-card__body) {
        padding: 14px;
      }

      .step-enter-active,
      .step-leave-active {
        transition: all 0.3s ease;
      }

      .step-enter-from {
        opacity: 0;
        transform: translateX(20px);
      }

      .step-leave-to {
        opacity: 0;
        transform: translateX(-20px);
      }
    }
  }
</style>
