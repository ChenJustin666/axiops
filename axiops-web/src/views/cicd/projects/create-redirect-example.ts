// 项目创建后跳转逻辑修改
// 文件: src/views/cicd/projects/index.vue

// 修改 submitCreate 方法
const submitCreate = async () => {
  if (!(await formRef.value?.validate())) return
  
  submitting.value = true
  try {
    // 创建项目
    const result = await createProjectApi(form)
    
    ElMessage.success('项目已创建')
    createVisible.value = false
    
    // 跳转到初始化向导（而不是直接跳转到工作区）
    await router.push(`/cicd/projects/${result.id}?init=1&tab=services`)
    
  } catch (error: any) {
    ElMessage.error(error.message || '创建失败')
  } finally {
    submitting.value = false
  }
}

// 修改 openProject 方法（点击项目卡片时）
const openProject = (id: number) => {
  // 直接跳转到工作区，工作区内部会检查初始化状态
  // 如果未初始化会自动重定向到向导
  router.push(`/cicd/projects/${id}`)
}
