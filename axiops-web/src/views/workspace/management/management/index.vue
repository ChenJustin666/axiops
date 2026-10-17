<template>
  <div class="workspace-management">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <ElButton :icon="ArrowLeft" @click="handleBack" circle size="small" />

        <div class="workspace-title">
          <div class="workspace-icon">
            <ElIcon :size="28">
              <Box />
            </ElIcon>
          </div>
          <div class="workspace-info">
            <h2>{{ workspaceDetail?.name || '工作空间' }}</h2>
          </div>
        </div>

        <div class="header-actions">
          <ElSpace :size="12">
            <ElButton :icon="Refresh" @click="refreshData" :loading="loading"> 刷新</ElButton>
            <ElButton type="primary" :icon="Setting" @click="handleSettings"> 配置管理</ElButton>
            <ElButton type="danger" :icon="Delete" @click="handleDelete" plain>
              删除工作空间
            </ElButton>
          </ElSpace>
        </div>
      </div>
    </div>

    <!-- 标签页内容 -->
    <ElCard class="content-card" shadow="never">
      <ElTabs v-model="activeTab">
        <ElTabPane label="基本信息" name="basic">
          <BasicInfo :workspace="workspaceDetail" :loading="loading" />
        </ElTabPane>
      </ElTabs>
    </ElCard>

    <!-- 编辑对话框 -->
    <WorkspaceDialog
      v-if="workspaceDetail && projectStore.selectedProject && clusterDetail"
      v-model="dialogVisible"
      mode="edit"
      :workspace="workspaceDetail"
      :project="projectStore.selectedProject"
      :cluster="clusterDetail"
      @success="handleDialogSuccess"
    />
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted } from 'vue'
  import { useRoute, useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox, ElLoading } from 'element-plus'
  import { ArrowLeft, Refresh, Setting, Delete, Box } from '@element-plus/icons-vue'
  import {
    getProjectWorkspaceApi,
    deleteProjectWorkspaceApi,
    getProjectClusterApi,
    type ProjectWorkspace,
    type ProjectCluster
  } from '@/api'
  import { useProjectStore } from '@/store/modules/project'
  import BasicInfo from './basic.vue'
  import WorkspaceDialog from '../subpage/dialog/index.vue'

  defineOptions({ name: 'WorkspaceManagement' })

  const route = useRoute()
  const router = useRouter()
  const projectStore = useProjectStore()

  // 状态
  const workspaceId = ref<number>(Number(route.params.id))
  const workspaceDetail = ref<ProjectWorkspace | null>(null)
  const clusterDetail = ref<ProjectCluster | null>(null)
  const loading = ref(false)
  const activeTab = ref('basic')
  const dialogVisible = ref(false)
  const dialogMode = ref<'create' | 'edit'>('edit')

  // 获取集群详情
  const getClusterDetail = async (clusterId: number) => {
    try {
      const response = await getProjectClusterApi(clusterId)
      clusterDetail.value = response
      console.log('✅ 成功加载集群详情:', response)
    } catch (error) {
      console.error('❌ 获取集群详情失败:', error)
      clusterDetail.value = null
    }
  }

  // 获取工作空间详情
  const getWorkspaceDetail = async () => {
    try {
      loading.value = true
      const response = await getProjectWorkspaceApi(workspaceId.value)
      workspaceDetail.value = response

      console.log('✅ 成功加载工作空间详情:', response)

      // 获取关联的集群详情
      if (response.projectClusterId) {
        await getClusterDetail(response.projectClusterId)
      } else {
        console.warn('⚠️ 工作空间没有关联的集群ID')
      }
    } catch (error) {
      console.error('❌ 获取工作空间详情失败:', error)
    } finally {
      loading.value = false
    }
  }

  // 刷新数据
  const refreshData = () => {
    getWorkspaceDetail()
  }

  // 返回
  const handleBack = () => {
    router.back()
  }

  // 设置 - 打开编辑对话框
  const handleSettings = () => {
    if (!workspaceDetail.value) {
      ElMessage.warning('工作空间信息加载中，请稍候')
      return
    }

    if (!projectStore.selectedProject) {
      ElMessage.warning('请先选择项目')
      return
    }

    // 使用本地的 clusterDetail 而不是 projectStore.selectedCluster
    if (!clusterDetail.value) {
      ElMessage.warning('集群信息加载中，请稍候')
      return
    }

    console.log('🔧 打开配置对话框:', {
      workspace: workspaceDetail.value,
      project: projectStore.selectedProject,
      cluster: clusterDetail.value
    })

    dialogMode.value = 'edit'
    dialogVisible.value = true
  }

  // 删除
  const handleDelete = async () => {
    try {
      await ElMessageBox.confirm(
        `确定要删除工作空间 "${workspaceDetail.value?.name}" 吗？此操作将删除所有相关资源。`,
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'error'
        }
      )

      // 显示全屏 loading
      const loadingInstance = ElLoading.service({
        lock: true,
        text: '正在删除工作空间，请稍候...',
        background: 'rgba(0, 0, 0, 0.7)'
      })

      try {
        await deleteProjectWorkspaceApi(workspaceId.value)
        ElMessage.success('删除成功')
        router.back()
      } finally {
        loadingInstance.close()
      }
    } catch (error) {
      if (error !== 'cancel') {
        console.error('❌ 删除失败:', error)
      }
    }
  }

  // 对话框成功回调
  const handleDialogSuccess = async () => {
    await getWorkspaceDetail()
  }

  // 初始化
  onMounted(() => {
    console.log('📝 工作空间详情页面初始化, ID:', workspaceId.value)
    getWorkspaceDetail()
  })
</script>

<style lang="scss" scoped>
  .workspace-management {
    height: 100%;
    padding: 20px;

    .page-header {
      background: var(--el-bg-color);
      border-radius: 8px;
      padding: 16px 20px;
      margin-bottom: 16px;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);

      .header-content {
        display: flex;
        align-items: center;
        gap: 16px;

        .workspace-title {
          display: flex;
          align-items: center;
          gap: 12px;
          flex: 1;

          .workspace-icon {
            display: flex;
            align-items: center;
            justify-content: center;
            width: 48px;
            height: 48px;
            border-radius: 8px;
            background: var(--el-color-primary-light-9);
            color: var(--el-color-primary);
          }

          .workspace-info {
            flex: 1;

            h2 {
              margin: 0 0 8px;
              font-size: 20px;
              font-weight: 500;
              color: var(--el-text-color-primary);
            }

            .workspace-meta {
              display: flex;
              align-items: center;
              gap: 8px;

              .meta-item {
                display: flex;
                align-items: center;
                gap: 4px;
                font-size: 14px;
                color: var(--el-text-color-regular);

                :deep(.el-icon) {
                  color: var(--el-text-color-secondary);
                }
              }

              :deep(.el-divider--vertical) {
                height: 14px;
                margin: 0 8px;
              }
            }
          }
        }

        .header-actions {
          flex-shrink: 0;
        }
      }
    }

    .content-card {
      border-radius: 8px;

      :deep(.el-card__body) {
        padding: 20px;
      }

      :deep(.el-tabs) {
        .el-tabs__header {
          margin-bottom: 20px;
        }
      }
    }
  }
</style>
