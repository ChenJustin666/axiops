<template>
  <div class="deployment-create-wizard">
    <!-- 步骤卡片 -->
    <WorkloadStepsCard
      :steps="steps"
      :current-step="currentStep"
      :mode="mode"
      :namespace="currentNamespace"
      resource-type="Deployment"
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
          :filename="`${metadataStore.metadata.nameEn || 'deployment'}.yaml`"
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
    useContainersStore,
    useVolumesStore,
    useSchedulingStore,
    useAdvancedStore,
    resetAllWorkloadStores
  } from '@/store/workload'
  import { useDeploymentStore } from '@/store/workload/deployment'

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
  import ReplicasStep from './components/ReplicasStep.vue'
  import ContainersStep from '../common/components/ContainersStep.vue'
  import VolumesStep from '../common/components/VolumesStep.vue'
  import SchedulingStep from '../common/components/SchedulingStep.vue'
  import AdvancedStep from '../common/components/AdvancedStep.vue'
  import ReviewStep from './components/ReviewStep.vue'

  // 初始化 Store
  const metadataStore = useMetadataStore()
  const containersStore = useContainersStore()
  const volumesStore = useVolumesStore()
  const schedulingStore = useSchedulingStore()
  const advancedStore = useAdvancedStore()
  const deploymentStore = useDeploymentStore()

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
  const appProjectId = computed(() => routeParams.value?.appProjectId || 0)
  // 编辑模式
  const editMode = ref<'form' | 'yaml'>('form')

  // 当前步骤
  const currentStep = ref(0)
  const currentStepRef = ref<any>(null)

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

  const steps = shallowRef<StepConfig[]>([
    {
      id: 'metadata',
      title: '元数据配置',
      description: 'Deployment名称和标签',
      component: MetadataStep,
      hasError: true,
      validated: false,
      touched: false,
      required: true
    },
    {
      id: 'replicas',
      title: '副本配置',
      description: '副本数和更新策略',
      component: ReplicasStep,
      hasError: false,
      validated: true,
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
  const canEditField = (field: 'nameCn' | 'nameEn' | 'version' | 'desc') => {
    return isFieldEditable(mode.value, field)
  }

  // 提供给子组件
  provide('namespace', currentNamespace)
  provide('resourceClusterId', resourceClusterId)
  provide('clusterUuid', clusterUuid)
  provide('workspaceId', workspaceId)
  provide('mode', mode)
  provide('canEditField', canEditField)
  provide('appProjectId', appProjectId)
  // ==================== 验证逻辑 ====================

  // ⭐ 新增：YAML 导入后验证所有步骤
  const validateAllStepsAfterYamlImport = async () => {
    const currentSteps = [...steps.value]
    let allValid = true
    const validationErrors: string[] = []

    // 1. 验证元数据步骤
    try {
      const metadataValid =
        metadataStore.metadata.nameCn &&
        metadataStore.metadata.nameEn &&
        metadataStore.metadata.version &&
        metadataStore.metadata.resourceName
      currentSteps[0].validated = metadataValid
      currentSteps[0].hasError = !metadataValid
      currentSteps[0].touched = true
      if (!metadataValid) {
        validationErrors.push('元数据配置不完整')
        allValid = false
      }
    } catch (error) {
      console.error('元数据验证失败:', error)
    }

    // 2. 验证副本配置
    try {
      const replicasValid = deploymentStore.replicas > 0
      currentSteps[1].validated = replicasValid
      currentSteps[1].hasError = !replicasValid
      currentSteps[1].touched = true
      if (!replicasValid) {
        validationErrors.push('副本数必须大于0')
        allValid = false
      }
    } catch (error) {
      console.error('副本验证失败:', error)
    }

    // 3. 验证容器配置（最关键）
    try {
      const containersResult = containersStore.validate()
      currentSteps[2].validated = containersResult.valid
      currentSteps[2].hasError = !containersResult.valid
      currentSteps[2].touched = true

      if (!containersResult.valid) {
        validationErrors.push(...containersResult.errors)
        allValid = false
      }
    } catch (error) {
      console.error('容器验证失败:', error)
      currentSteps[2].validated = false
      currentSteps[2].hasError = true
      validationErrors.push('容器配置验证异常')
      allValid = false
    }

    // 4. 其他非必填步骤默认通过
    currentSteps[3].validated = true // 存储卷
    currentSteps[3].hasError = false
    currentSteps[4].validated = true // 调度策略
    currentSteps[4].hasError = false
    currentSteps[5].validated = true // 高级配置
    currentSteps[5].hasError = false
    currentSteps[6].validated = true // 确认配置
    currentSteps[6].hasError = false

    steps.value = currentSteps

    // 显示验证结果
    if (!allValid) {
      // 显示第一个错误
      if (validationErrors.length > 0) {
        ElMessage.error({
          message: `配置验证：${validationErrors[0]}`,
          duration: 4000,
          showClose: true
        })
      }
    }

    return allValid
  }

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

  // ==================== 数据生成 ====================

  // 生成完整的 Deployment 对象
  const generateDeploymentObject = () => {
    const { metadata, namespace } = metadataStore
    const { nameEn, version, resourceName, nameCn, desc, labels, annotations } = metadata

    // 生成完整的标签
    const finalLabels = {
      app: nameEn,
      version: version,
      ...labels
    }

    // 生成完整的注解
    const finalAnnotations = {
      'ikubeops.com/service-name': nameCn,
      'created-by': 'kube-nova',
      ...(desc ? { description: desc } : {}),
      ...annotations
    }

    // 获取容器配置
    const containerConfig = containersStore.toK8sFormat()

    // 获取调度配置
    const schedulingSpec = schedulingStore.toK8sFormat()

    // 获取高级配置
    const advancedSpec = advancedStore.toK8sFormat()

    // 获取存储卷
    const volumes = volumesStore.volumes.length > 0 ? volumesStore.volumes : undefined

    // 获取 Deployment 配置
    const deploymentSpec = deploymentStore.toK8sFormat()

    // 构建 Pod Spec
    const podSpec: any = {
      containers: containerConfig.containers,
      ...(containerConfig.initContainers && { initContainers: containerConfig.initContainers }),
      ...(volumes && volumes.length > 0 && { volumes }),
      ...schedulingSpec,
      ...advancedSpec
    }

    // 构建 Deployment
    const deployment: any = {
      apiVersion: 'apps/v1',
      kind: 'Deployment',
      metadata: {
        name: resourceName,
        namespace: namespace,
        labels: finalLabels,
        annotations: finalAnnotations
      },
      spec: {
        ...deploymentSpec,
        selector: {
          matchLabels: {
            app: nameEn,
            version: version
          }
        },
        template: {
          metadata: {
            labels: finalLabels,
            annotations: finalAnnotations
          },
          spec: podSpec
        }
      }
    }

    return deployment
  }

  // ==================== 保存逻辑 ====================

  /**
   * 智能合并原始 YAML 和表单数据
   * 策略：表单管理的字段使用表单数据，其他字段保留原始值
   */
  const mergeWithOriginalYaml = (formData: any): any => {
    if (!originalYamlObj.value || mode.value !== 'editAppVersion') {
      return formData
    }

    const original = originalYamlObj.value

    // 深度合并 metadata（保留原始中表单没有的字段）
    const mergedMetadata = {
      ...original.metadata,
      ...formData.metadata,
      // 确保关键字段使用表单数据
      name: formData.metadata.name,
      namespace: formData.metadata.namespace,
      labels: formData.metadata.labels,
      annotations: formData.metadata.annotations
    }
    // 删除不需要的运行时字段
    delete mergedMetadata.uid
    delete mergedMetadata.resourceVersion
    delete mergedMetadata.generation
    delete mergedMetadata.creationTimestamp
    delete mergedMetadata.managedFields

    // 深度合并 spec.template.spec（Pod 模板）
    const originalPodSpec = original.spec?.template?.spec || {}
    const formPodSpec = formData.spec?.template?.spec || {}

    // 合并容器：基于 name 匹配
    const mergedContainers =
      formPodSpec.containers?.map((formContainer: any) => {
        const originalContainer = originalPodSpec.containers?.find(
          (c: any) => c.name === formContainer.name
        )
        if (originalContainer) {
          // 同名容器：保留原始中表单没有的字段
          return {
            ...originalContainer,
            ...formContainer
          }
        }
        return formContainer
      }) || []

    // 合并 initContainers
    const mergedInitContainers = formPodSpec.initContainers?.map((formContainer: any) => {
      const originalContainer = originalPodSpec.initContainers?.find(
        (c: any) => c.name === formContainer.name
      )
      if (originalContainer) {
        return {
          ...originalContainer,
          ...formContainer
        }
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

    // 合并 spec（保留原始中表单没有的字段）
    const mergedSpec = {
      ...original.spec,
      ...formData.spec,
      template: {
        ...original.spec?.template,
        ...formData.spec?.template,
        metadata: formData.spec?.template?.metadata,
        spec: mergedPodSpec
      }
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
    // 验证所有必填步骤
    const requiredSteps = steps.value.filter((s) => s.required)
    const allValid = requiredSteps.every((s) => s.validated && !s.hasError)

    if (!allValid) {
      return
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

      // 生成 Deployment 对象
      let deployment = generateDeploymentObject()

      // 编辑模式下与原始 YAML 合并，保留未被表单管理的字段
      deployment = mergeWithOriginalYaml(deployment)

      // 确保 namespace 和标签正确
      ensureNamespace(deployment, finalNamespace)
      ensureRequiredLabelsInResource(
        deployment,
        metadataStore.metadata.nameEn,
        metadataStore.metadata.version
      )

      // 确保 metadata.name 与 resourceName 一致
      deployment.metadata.name = metadataStore.metadata.resourceName

      // 生成YAML字符串
      const yamlStr = yaml.dump(deployment, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })

      // 调用统一提交函数
      const result = await submitWorkload({
        mode: mode.value,
        resourceType: ResourceType.DEPLOYMENT,
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
          const formDataObj = generateDeploymentObject()

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
          const parsedDeployment = yaml.load(yamlContent.value) as any
          resourceNameFromYaml = parsedDeployment?.metadata?.name || ''
        }
      } else {
        // 非编辑模式：直接解析 YAML 获取 resourceName
        try {
          const parsedDeployment = yaml.load(yamlContent.value) as any
          resourceNameFromYaml = parsedDeployment?.metadata?.name || ''
        } catch (error) {
          console.error('解析 YAML 失败:', error)
        }
      }

      const result = await submitWorkloadFromYaml(
        {
          mode: mode.value,
          resourceType: ResourceType.DEPLOYMENT,
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
    deploymentStore.reset()

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

  // ==================== 监听器 ====================

  // ========== 修复后的 YAML 切换逻辑 ==========
  // 替换 index.vue 中 watch(editMode, ...) 的 else if 分支

  watch(editMode, async (newMode, oldMode) => {
    if (newMode === 'yaml') {
      // 切换到 YAML 模式：从 store 生成 YAML
      let deployment = generateDeploymentObject()
      // 编辑模式下与原始 YAML 合并，保留未被表单管理的字段
      deployment = mergeWithOriginalYaml(deployment)
      yamlContent.value = yaml.dump(deployment, {
        indent: 2,
        lineWidth: -1,
        noRefs: true,
        sortKeys: false
      })
    } else if (newMode === 'form' && oldMode === 'yaml') {
      // 从 YAML 模式切换回表单模式
      if (yamlContent.value && yamlContent.value.trim()) {
        try {
          // ⭐ 验证 YAML 内容
          const documents = yaml.loadAll(yamlContent.value)

          if (documents.length === 0) {
            ElMessage.error('YAML 内容为空')
            editMode.value = 'yaml'
            return
          }

          if (documents.length > 1) {
            ElMessage.error({
              message: `检测到 ${documents.length} 个 YAML 文档，Deployment 页面只能包含一个 Deployment 类型的 YAML`,
              duration: 5000,
              showClose: true
            })
            editMode.value = 'yaml'
            return
          }

          const parsedDeployment = documents[0] as any

          if (!parsedDeployment || !parsedDeployment.kind) {
            ElMessage.error('YAML 格式不正确')
            editMode.value = 'yaml'
            return
          }

          if (parsedDeployment.kind !== 'Deployment') {
            ElMessage.error({
              message: `当前页面只能处理 Deployment 类型的 YAML，但检测到 kind: ${parsedDeployment.kind}`,
              duration: 5000,
              showClose: true
            })
            editMode.value = 'yaml'
            return
          }

          // ⭐⭐⭐ 关键修复：先从 YAML 中提取元数据 ⭐⭐⭐
          const yamlMetadata = parsedDeployment.metadata || {}
          const yamlLabels = yamlMetadata.labels || {}
          const yamlAnnotations = yamlMetadata.annotations || {}
          const yamlResourceName = parsedDeployment.metadata?.name || ''
          // 检查 resourceName 是否为空
          if (!yamlResourceName) {
            ElMessage.error({
              message: '后端返回的 YAML 缺少 metadata.name 字段，请检查后端代码',
              duration: 5000,
              showClose: true
            })
            setTimeout(() => {
              router.back()
            }, 1500)
            return
          }
          // 从 YAML 中解析字段
          const parsedNameEn = yamlLabels.app || '' // 英文名从 labels.app 获取
          const parsedVersion = yamlLabels.version || '' // 版本从 labels.version 获取
          const parsedResourceName = yamlMetadata.name || '' // 资源名称从 metadata.name 获取
          const parsedDesc =
            yamlAnnotations.description || yamlAnnotations['ikubeops.com/description'] || ''
          const parsedNameCn = yamlAnnotations['ikubeops.com/service-name'] || '' // 尝试从注解获取中文名

          console.log('📝 从 YAML 解析的元数据:', {
            nameEn: parsedNameEn,
            version: parsedVersion,
            resourceName: parsedResourceName,
            nameCn: parsedNameCn,
            desc: parsedDesc
          })

          // 先取消选中容器
          containersStore.selectContainer(null)

          // 切换到第一步
          currentStep.value = 0

          // 等待组件卸载完成
          await nextTick()

          // 更新 namespace
          if (parsedDeployment.metadata) {
            parsedDeployment.metadata.namespace = currentNamespace.value
          }

          // 加载 YAML 到 stores（容器、存储卷等）
          const { updateStoresFromDeployment } = await import('@/store/workload')
          updateStoresFromDeployment(parsedDeployment)

          if (parsedDeployment.spec) {
            deploymentStore.loadFromK8s(parsedDeployment.spec)
          }

          // ：使用从 YAML 解析的数据更新元数据
          metadataStore.updateMetadata({
            nameCn: parsedNameCn, // 中文名（可能为空，需要用户填写）
            nameEn: parsedNameEn, // 英文名
            version: parsedVersion, // 版本
            resourceName: yamlResourceName,
            desc: parsedDesc // 描述
          })
          // 手动标记
          metadataStore.setResourceName(yamlResourceName)
          // 设置命名空间
          metadataStore.setNamespace(currentNamespace.value)

          console.log('metadataStore:', JSON.stringify(metadataStore.metadata))

          // 强制刷新组件
          componentKey.value++

          // 等待渲染完成
          await nextTick()
          await nextTick()

          // 验证所有步骤
          await validateAllStepsAfterYamlImport()

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
          console.error('解析YAML失败:', error)
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

  // ==================== 生命周期 ====================

  // 组件挂载时
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

      const parsedDeployment = documents[0] as any

      if (!parsedDeployment || parsedDeployment.kind !== 'Deployment') {
        throw new Error(`期望 Deployment 类型，但得到: ${parsedDeployment?.kind || '未知'}`)
      }

      // 编辑模式下保存原始 YAML 对象（深拷贝）
      if (mode === 'editAppVersion') {
        originalYamlObj.value = JSON.parse(JSON.stringify(parsedDeployment))
      }

      // 从 YAML 中提取原始 resourceName
      const yamlResourceName = parsedDeployment.metadata?.name || ''

      // ✅ 确保 namespace 正确（编辑模式和复制模式下使用当前 namespace）
      if (parsedDeployment.metadata) {
        parsedDeployment.metadata.namespace = currentNamespace.value
      }

      const { updateStoresFromDeployment } = await import('@/store/workload')
      updateStoresFromDeployment(parsedDeployment)

      if (parsedDeployment.spec) {
        deploymentStore.loadFromK8s(parsedDeployment.spec)
      }

      // ✅ 根据模式决定最终的元数据（关键：编辑模式和复制模式下 ns、nameCn、nameEn、version 不允许修改）
      let finalResourceName = savedMetadata.resourceName
      if (mode === 'editAppVersion') {
        // 编辑模式：使用 YAML 中的原始 resourceName，不允许修改
        finalResourceName = yamlResourceName || savedMetadata.resourceName
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

      // 等待组件更新完成后
      await nextTick()
      await nextTick()

      // 手动设置步骤验证状态
      const currentSteps = [...steps.value]

      // 步骤1：元数据配置
      const metadataValid =
        savedMetadata.nameCn && savedMetadata.nameEn && savedMetadata.version && finalResourceName
      currentSteps[0].validated = metadataValid
      currentSteps[0].hasError = !metadataValid
      currentSteps[0].touched = true

      // 步骤2：副本配置
      const replicasValid = deploymentStore.replicas > 0
      currentSteps[1].validated = replicasValid
      currentSteps[1].hasError = !replicasValid
      currentSteps[1].touched = true

      // 步骤3：容器配置
      const containersResult = containersStore.validate()
      currentSteps[2].validated = containersResult.valid
      currentSteps[2].hasError = !containersResult.valid
      currentSteps[2].touched = true

      // 步骤4-7：非必填步骤，默认通过
      for (let i = 3; i < currentSteps.length; i++) {
        currentSteps[i].validated = true
        currentSteps[i].hasError = false
        currentSteps[i].touched = true
      }

      steps.value = currentSteps

      ElMessage.success('配置加载完成')
    } catch (error) {
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
  .deployment-create-wizard {
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
