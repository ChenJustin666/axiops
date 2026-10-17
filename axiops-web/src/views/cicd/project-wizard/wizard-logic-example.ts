// 向导页面核心逻辑示例
// 文件: src/views/cicd/project-wizard/index.vue

import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import {
  getProjectApi,
  updateProjectApi,
  listAllServicesApi,
  listEnvironmentsApi,
  listWorkflowsApi,
  type AxiOpsProject
} from '@/api/axiops'

const route = useRoute()
const router = useRouter()
const projectId = Number(route.params.projectId)

const loading = ref(false)
const completing = ref(false)
const currentStep = ref(1)
const project = ref<AxiOpsProject | null>(null)
const services = ref([])
const environments = ref([])
const workflows = ref([])

const steps = [
  { value: 1, title: '新建服务', description: '核心配置' },
  { value: 2, title: '创建环境', description: '部署目标' },
  { value: 3, title: '执行工作流', description: '流水线触发' }
]

// 能否进入下一步
const canProceed = computed(() => {
  if (currentStep.value === 1) return services.value.length > 0
  if (currentStep.value === 2) return environments.value.length > 0
  return false
})

// 加载项目信息
const loadProject = async () => {
  loading.value = true
  try {
    project.value = await getProjectApi(projectId)
    
    // 如果已完成初始化，跳转到工作区
    if (project.value.initialization_status === 'completed') {
      ElMessage.warning('项目已完成初始化，跳转到工作区')
      await router.replace(`/cicd/projects/${projectId}`)
      return
    }
    
    // 恢复到上次的步骤
    currentStep.value = project.value.initialization_step || 1
  } catch (error) {
    ElMessage.error('加载项目失败')
    await router.push('/cicd/projects')
  } finally {
    loading.value = false
  }
}

// 下一步
const nextStep = async () => {
  if (!canProceed.value) {
    ElMessage.warning('请先完成当前步骤的配置')
    return
  }

  try {
    // 保存当前步骤到后端
    await updateProjectApi(projectId, {
      initialization_step: currentStep.value + 1
    })
    currentStep.value++

    // 加载下一步需要的数据
    if (currentStep.value === 3) {
      await loadWorkflows()
    }
  } catch (error) {
    ElMessage.error('保存进度失败')
  }
}

// 上一步
const previousStep = async () => {
  if (currentStep.value > 1) {
    try {
      await updateProjectApi(projectId, {
        initialization_step: currentStep.value - 1
      })
      currentStep.value--
    } catch (error) {
      ElMessage.error('保存进度失败')
    }
  }
}

// 完成初始化
const completeWizard = async () => {
  if (environments.value.length === 0) {
    ElMessage.warning('至少需要创建一个环境才能完成初始化')
    return
  }

  try {
    await ElMessageBox.confirm(
      '完成初始化后将进入项目工作区，您可以继续配置服务、环境和工作流。确认完成初始化吗？',
      '完成初始化',
      { confirmButtonText: '确认完成', cancelButtonText: '取消', type: 'success' }
    )
  } catch {
    return
  }

  completing.value = true
  try {
    // 标记为已完成
    await updateProjectApi(projectId, {
      initialization_status: 'completed',
      initialization_step: 3
    })
    
    ElMessage.success('项目初始化完成！')
    await router.push(`/cicd/projects/${projectId}`)
  } catch (error) {
    ElMessage.error('完成初始化失败')
  } finally {
    completing.value = false
  }
}

// 退出向导
const exitWizard = async () => {
  try {
    await ElMessageBox.confirm(
      '退出向导后可以从项目列表重新进入继续配置。确认退出吗？',
      '退出向导',
      { confirmButtonText: '确认退出', cancelButtonText: '取消', type: 'warning' }
    )
    await router.push('/cicd/projects')
  } catch {
    // 用户取消
  }
}

// 页面加载时
onMounted(async () => {
  await loadProject()
  if (project.value) {
    await Promise.all([
      listAllServicesApi({ project_id: projectId }).then(res => services.value = res.list || []),
      listEnvironmentsApi({ project_id: projectId }).then(res => environments.value = res || [])
    ])
  }
})
