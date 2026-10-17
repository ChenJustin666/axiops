// 工作区页面添加初始化状态检查
// 文件: src/views/cicd/project-workspace/index.vue

// 在现有的 onMounted 中添加检查逻辑

const loadWorkspace = async () => {
  loading.value = true
  loadError.value = ''
  
  try {
    // 1. 加载项目信息
    project.value = await getProjectApi(projectId)
    
    // 2. 检查初始化状态
    if (project.value.initialization_status !== 'completed') {
      ElMessage.info('项目尚未完成初始化，跳转到初始化向导')
      await router.replace(`/cicd/projects/${projectId}/wizard`)
      return
    }
    
    // 3. 已初始化，加载工作区数据
    await Promise.all([
      loadServices(),
      loadEnvironments(),
      loadClusters(),
      loadWorkflows(),
      loadCodeSources()
    ])
    
    // 4. 默认选中第一个服务
    if (services.value.length > 0) {
      selectedService.value = services.value[0]
      expandedServices.value.add(services.value[0].id)
    }
    
  } catch (error: any) {
    loadError.value = error.message || '加载项目失败'
    console.error('加载工作区失败:', error)
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await access.load()
  await loadWorkspace()
})
