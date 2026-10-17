<template>
  <div class="project-detail-page art-full-height">
    <!-- 页面头部 -->
    <div class="page-header">
      <div class="header-content">
        <ElBreadcrumb separator="/">
          <ElBreadcrumbItem :to="{ path: '/project/management' }">项目管理</ElBreadcrumbItem>
          <ElBreadcrumbItem>项目详情</ElBreadcrumbItem>
        </ElBreadcrumb>

        <div class="project-title" v-loading="loading">
          <h2>{{ projectInfo.name }}</h2>
          <ElTag v-if="projectInfo.isSystem" type="info" size="small">系统项目</ElTag>
        </div>
      </div>

      <div class="header-actions">
        <ElButton @click="handleBack">
          <ArrowLeft :size="16" />
          返回
        </ElButton>
        <ElButton type="primary" @click="handleRefresh" :loading="loading">
          <RefreshCw :size="16" />
          刷新
        </ElButton>
      </div>
    </div>

    <!-- 详情内容 -->
    <div class="detail-content">
      <ElTabs v-model="activeTab" class="detail-tabs">
        <!-- 基本信息 -->
        <ElTabPane label="基本信息" name="info">
          <ElCard shadow="never" v-loading="loading">
            <ElDescriptions :column="2" border>
              <ElDescriptionsItem label="项目名称">
                {{ projectInfo.name }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="项目标识">
                <ElTag type="primary" effect="plain" size="small">{{ projectInfo.uuid }}</ElTag>
              </ElDescriptionsItem>
              <ElDescriptionsItem label="项目描述" :span="2">
                {{ projectInfo.description || '暂无描述' }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="系统项目">
                <ElTag :type="projectInfo.isSystem ? 'warning' : 'success'" size="small">
                  {{ projectInfo.isSystem ? '是' : '否' }}
                </ElTag>
              </ElDescriptionsItem>
              <ElDescriptionsItem label="创建人">
                {{ projectInfo.createdBy }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="创建时间">
                {{ formatDate(projectInfo.createdAt) }}
              </ElDescriptionsItem>
              <ElDescriptionsItem label="更新时间">
                {{ formatDate(projectInfo.updatedAt) }}
              </ElDescriptionsItem>
            </ElDescriptions>
          </ElCard>
        </ElTabPane>

        <!-- 项目成员 -->
        <ElTabPane label="项目成员" name="members">
          <ElCard shadow="never">
            <template #header>
              <div class="card-header">
                <span>成员列表</span>
                <ElButton
                  type="primary"
                  size="small"
                  @click="handleAddMember"
                  :loading="memberLoading"
                >
                  <Plus :size="16" />
                  添加成员
                </ElButton>
              </div>
            </template>

            <ElTable :data="memberList" v-loading="memberLoading" style="width: 100%">
              <ElTableColumn prop="username" label="用户名" width="150" />
              <ElTableColumn prop="nickname" label="昵称" width="150" />
              <ElTableColumn prop="email" label="邮箱" min-width="200" />
              <ElTableColumn prop="phone" label="手机号" width="150" />
              <ElTableColumn label="加入时间" width="180">
                <template #default="{ row }">
                  {{ formatDate(row.createdAt) }}
                </template>
              </ElTableColumn>
              <ElTableColumn label="操作" width="120" fixed="right">
                <template #default="{ row }">
                  <ElButton
                    link
                    type="danger"
                    size="small"
                    @click="handleRemoveMember(row.userId)"
                    :loading="removingMembers[row.userId]"
                  >
                    移除
                  </ElButton>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElCard>
        </ElTabPane>

        <!-- 资源配额 -->
        <ElTabPane label="资源配额" name="resources">
          <ElCard shadow="never" v-loading="resourceLoading">
            <template #header>
              <div class="card-header">
                <span>集群资源配额</span>
                <ElButton type="primary" size="small" @click="handleAddResource">
                  <Plus :size="16" />
                  分配资源
                </ElButton>
              </div>
            </template>

            <div class="resource-list">
              <div v-if="resourceList.length === 0" class="empty-state">
                <ElEmpty description="暂未分配集群资源" />
              </div>

              <div v-else class="resource-grid">
                <div v-for="resource in resourceList" :key="resource.id" class="resource-card">
                  <div class="resource-header">
                    <div class="resource-title">
                      <Monitor :size="16" class="cluster-icon" />
                      <ElTooltip
                        :content="resource.clusterName"
                        placement="top"
                        :disabled="resource.clusterName?.length < 30"
                      >
                        <span class="cluster-name">{{ resource.clusterName }}</span>
                      </ElTooltip>
                    </div>
                    <ElDropdown @command="(cmd) => handleResourceCommand(cmd, resource)">
                      <ElButton link type="primary" size="small" class="more-btn">
                        <MoreHorizontal :size="14" />
                      </ElButton>
                      <template #dropdown>
                        <ElDropdownMenu>
                          <ElDropdownItem command="edit">
                            <Edit :size="12" style="margin-right: 6px" />
                            编辑配额
                          </ElDropdownItem>
                          <ElDropdownItem command="sync">
                            <RefreshCw :size="12" style="margin-right: 6px" />
                            同步状态
                          </ElDropdownItem>
                          <ElDropdownItem command="delete" divided>
                            <Trash2 :size="12" style="margin-right: 6px" />
                            删除配额
                          </ElDropdownItem>
                        </ElDropdownMenu>
                      </template>
                    </ElDropdown>
                  </div>

                  <div class="resource-content">
                    <div class="resource-item">
                      <div class="label-row">
                        <span class="label">CPU</span>
                        <span class="value">
                          {{ formatCpuSize(resource.cpuAllocated) }} /
                          {{ formatCpuSize(resource.cpuLimit) }}
                        </span>
                      </div>
                      <div class="progress-row">
                        <ElProgress
                          :percentage="getUsagePercentage(resource.cpuAllocated, resource.cpuLimit)"
                          :stroke-width="6"
                          :show-text="false"
                          :color="getProgressColor(resource.cpuAllocated, resource.cpuLimit)"
                        />
                        <span class="percentage"
                          >{{ getUsagePercentage(resource.cpuAllocated, resource.cpuLimit) }}%</span
                        >
                      </div>
                      <div class="resource-detail" v-if="resource.cpuOvercommitRatio > 1">
                        <span class="overcommit">超分: {{ resource.cpuOvercommitRatio }}x</span>
                        <span class="capacity"
                          >容量: {{ formatCpuSize(resource.cpuCapacity) }}</span
                        >
                      </div>
                    </div>

                    <div class="resource-item">
                      <div class="label-row">
                        <span class="label">内存</span>
                        <span class="value">
                          {{ formatMemorySize(resource.memAllocated) }} /
                          {{ formatMemorySize(resource.memLimit) }}
                        </span>
                      </div>
                      <div class="progress-row">
                        <ElProgress
                          :percentage="getUsagePercentage(resource.memAllocated, resource.memLimit)"
                          :stroke-width="6"
                          :show-text="false"
                          :color="getProgressColor(resource.memAllocated, resource.memLimit)"
                        />
                        <span class="percentage"
                          >{{ getUsagePercentage(resource.memAllocated, resource.memLimit) }}%</span
                        >
                      </div>
                      <div class="resource-detail" v-if="resource.memOvercommitRatio > 1">
                        <span class="overcommit">超分: {{ resource.memOvercommitRatio }}x</span>
                        <span class="capacity"
                          >容量: {{ formatMemorySize(resource.memCapacity) }}</span
                        >
                      </div>
                    </div>

                    <div class="resource-item">
                      <div class="label-row">
                        <span class="label">存储</span>
                        <span class="value">
                          {{ formatStorageSize(resource.storageAllocated) }} /
                          {{ formatStorageSize(resource.storageLimit) }}
                        </span>
                      </div>
                      <div class="progress-row">
                        <ElProgress
                          :percentage="
                            getUsagePercentage(resource.storageAllocated, resource.storageLimit)
                          "
                          :stroke-width="6"
                          :show-text="false"
                          :color="
                            getProgressColor(resource.storageAllocated, resource.storageLimit)
                          "
                        />
                        <span class="percentage"
                          >{{
                            getUsagePercentage(resource.storageAllocated, resource.storageLimit)
                          }}%</span
                        >
                      </div>
                    </div>

                    <div
                      class="resource-item"
                      v-if="parseResourceString(resource.gpuLimit || '0') > 0"
                    >
                      <div class="label-row">
                        <span class="label">GPU</span>
                        <span class="value">
                          {{ parseResourceString(resource.gpuAllocated || '0') }} /
                          {{ parseResourceString(resource.gpuLimit || '0') }} 个
                        </span>
                      </div>
                      <div class="progress-row">
                        <ElProgress
                          :percentage="getUsagePercentage(resource.gpuAllocated, resource.gpuLimit)"
                          :stroke-width="6"
                          :show-text="false"
                          :color="getProgressColor(resource.gpuAllocated, resource.gpuLimit)"
                        />
                        <span class="percentage"
                          >{{ getUsagePercentage(resource.gpuAllocated, resource.gpuLimit) }}%</span
                        >
                      </div>
                      <div class="resource-detail" v-if="resource.gpuOvercommitRatio > 1">
                        <span class="overcommit">超分: {{ resource.gpuOvercommitRatio }}x</span>
                        <span class="capacity"
                          >容量: {{ parseResourceString(resource.gpuCapacity || '0') }} 个</span
                        >
                      </div>
                    </div>

                    <div class="resource-item">
                      <div class="label-row">
                        <span class="label">Pods</span>
                        <span class="value"
                          >{{ resource.podsAllocated }} / {{ resource.podsLimit }} 个</span
                        >
                      </div>
                      <div class="progress-row">
                        <ElProgress
                          :percentage="
                            getUsagePercentage(resource.podsAllocated, resource.podsLimit)
                          "
                          :stroke-width="6"
                          :show-text="false"
                          :color="getProgressColor(resource.podsAllocated, resource.podsLimit)"
                        />
                        <span class="percentage"
                          >{{
                            getUsagePercentage(resource.podsAllocated, resource.podsLimit)
                          }}%</span
                        >
                      </div>
                    </div>
                  </div>

                  <div class="resource-footer">
                    <Clock :size="12" />
                    <span>{{ formatDate(resource.updatedAt) }}</span>
                  </div>
                </div>
              </div>
            </div>
          </ElCard>
        </ElTabPane>

        <!-- 工作空间 -->
        <ElTabPane label="工作空间" name="workspaces">
          <ElCard shadow="never" v-loading="workspaceLoading">
            <template #header>
              <div class="card-header">
                <span>工作空间列表</span>
              </div>
            </template>

            <ElTable :data="workspaceList" style="width: 100%">
              <ElTableColumn prop="name" label="名称" min-width="150" />
              <ElTableColumn prop="namespace" label="命名空间" width="150">
                <template #default="{ row }">
                  <ElTag size="small">{{ row.namespace }}</ElTag>
                </template>
              </ElTableColumn>
              <ElTableColumn prop="clusterName" label="所属集群" width="150" />
              <ElTableColumn label="资源配额" min-width="300">
                <template #default="{ row }">
                  <div class="resource-summary">
                    <span>CPU: {{ formatCpuSize(row.cpuAllocated) }}</span>
                    <ElDivider direction="vertical" />
                    <span>内存: {{ formatMemorySize(row.memAllocated) }}</span>
                    <ElDivider direction="vertical" />
                    <span>存储: {{ formatStorageSize(row.storageAllocated) }}</span>
                  </div>
                </template>
              </ElTableColumn>
              <ElTableColumn label="状态" width="100">
                <template #default="{ row }">
                  <ElTag :type="row.status === 1 ? 'success' : 'danger'" size="small">
                    {{ row.status === 1 ? '正常' : '异常' }}
                  </ElTag>
                </template>
              </ElTableColumn>
            </ElTable>
          </ElCard>
        </ElTabPane>
      </ElTabs>
    </div>

    <!-- 成员分配弹窗 -->
    <MemberDialog
      v-model:visible="memberDialogVisible"
      :projectId="projectId"
      :submitLoading="memberSubmitLoading"
      @submit="handleMemberSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import { useRoute, useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import {
    ArrowLeft,
    RefreshCw,
    Plus,
    Edit,
    Trash2,
    MoreHorizontal,
    Monitor,
    Clock
  } from 'lucide-vue-next'
  import MemberDialog from '../modules/member-dialog.vue'
  import {
    getProjectApi,
    getProjectAdminsApi,
    searchProjectClusterApi,
    searchProjectWorkspaceApi,
    addProjectAdminApi,
    deleteProjectClusterApi,
    syncProjectApi,
    parseResourceString,
    formatCpuSize,
    formatMemorySize,
    formatStorageSize
  } from '@/api/manager'
  import { searchUserApi } from '@/api/portal/user'

  defineOptions({ name: 'ProjectDetail' })

  const route = useRoute()
  const router = useRouter()

  // 项目ID
  const projectId = computed(() => Number(route.params.id))

  // 状态管理
  const loading = ref(false)
  const memberLoading = ref(false)
  const resourceLoading = ref(false)
  const workspaceLoading = ref(false)
  const memberSubmitLoading = ref(false)
  const removingMembers = ref<Record<number, boolean>>({})
  const activeTab = ref('info')
  const memberDialogVisible = ref(false)

  // 数据
  const projectInfo = ref<any>({})
  const memberList = ref<any[]>([])
  const resourceList = ref<any[]>([])
  const workspaceList = ref<any[]>([])

  // 格式化日期
  const formatDate = (timestamp: number) => {
    if (!timestamp) return ''
    return new Date(timestamp * 1000).toLocaleString('zh-CN')
  }

  // 计算使用率百分比
  const getUsagePercentage = (used: string | number, total: string | number) => {
    const usedNum = typeof used === 'string' ? parseResourceString(used) : used
    const totalNum = typeof total === 'string' ? parseResourceString(total) : total

    if (totalNum === 0) return 0
    return Math.min(100, Math.round((usedNum / totalNum) * 100))
  }

  // 获取进度条颜色
  const getProgressColor = (used: string | number, total: string | number) => {
    const percentage = getUsagePercentage(used, total)
    if (percentage < 70) return '#67c23a'
    if (percentage < 90) return '#e6a23c'
    return '#f56c6c'
  }

  // 获取项目详情
  const getProjectInfo = async () => {
    loading.value = true
    try {
      const response = await getProjectApi(projectId.value)
      projectInfo.value = response
    } catch (error) {
    } finally {
      loading.value = false
    }
  }

  // 获取项目成员
  const getProjectMembers = async () => {
    memberLoading.value = true
    try {
      const response = await getProjectAdminsApi({
        projectId: projectId.value
      })

      const adminIds = response.map((admin) => admin.userId) || []

      if (adminIds.length > 0) {
        const userResponse = await searchUserApi({
          page: 1,
          pageSize: 200
        })

        memberList.value = (userResponse.items || [])
          .filter((user) => adminIds.includes(user.id))
          .map((user) => {
            const admin = response.find((a) => a.userId === user.id)
            return {
              ...user,
              userId: user.id,
              createdAt: admin?.createdAt || 0
            }
          })
      } else {
        memberList.value = []
      }
    } catch (error) {
      memberList.value = []
    } finally {
      memberLoading.value = false
    }
  }

  // 获取资源配额
  const getProjectResources = async () => {
    resourceLoading.value = true
    try {
      const response = await searchProjectClusterApi({
        projectId: projectId.value
      })
      resourceList.value = response || []
    } catch (error) {
      resourceList.value = []
    } finally {
      resourceLoading.value = false
    }
  }

  // 获取工作空间
  const getProjectWorkspaces = async () => {
    workspaceLoading.value = true
    try {
      const clusterResponse = await searchProjectClusterApi({
        projectId: projectId.value
      })

      const allWorkspaces: any[] = []

      for (const cluster of clusterResponse || []) {
        const workspaceResponse = await searchProjectWorkspaceApi({
          projectClusterId: cluster.id
        })
        allWorkspaces.push(...(workspaceResponse || []))
      }

      workspaceList.value = allWorkspaces
    } catch (error) {
      workspaceList.value = []
    } finally {
      workspaceLoading.value = false
    }
  }

  // 事件处理
  const handleBack = () => {
    router.push('/project/management')
  }

  const handleRefresh = () => {
    getProjectInfo()
    if (activeTab.value === 'members') {
      getProjectMembers()
    } else if (activeTab.value === 'resources') {
      getProjectResources()
    } else if (activeTab.value === 'workspaces') {
      getProjectWorkspaces()
    }
  }

  const handleAddMember = () => {
    memberDialogVisible.value = true
  }

  const handleMemberSubmit = async (userIds: number[]) => {
    memberSubmitLoading.value = true
    try {
      await addProjectAdminApi({
        projectId: projectId.value,
        userIds
      })
      ElMessage.success('成员分配成功')
      memberDialogVisible.value = false
      getProjectMembers()
    } catch (error: any) {
    } finally {
      memberSubmitLoading.value = false
    }
  }

  const handleRemoveMember = async (userId: number) => {
    try {
      await ElMessageBox.confirm('确定要移除该成员吗？', '提示', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      removingMembers.value[userId] = true

      const response = await getProjectAdminsApi({
        projectId: projectId.value
      })

      const remainingUserIds = (response || [])
        .map((admin) => admin.userId)
        .filter((id) => id !== userId)

      await addProjectAdminApi({
        projectId: projectId.value,
        userIds: remainingUserIds
      })

      ElMessage.success('移除成功')
      getProjectMembers()
    } catch (error) {
      if (error !== 'cancel') {
      }
    } finally {
      removingMembers.value[userId] = false
    }
  }

  const handleAddResource = () => {
    router.push('/project/resource-pool')
  }

  const handleResourceCommand = async (command: string, resource: any) => {
    switch (command) {
      case 'edit':
        router.push('/project/resource-pool')
        break
      case 'sync':
        await handleSyncResource(resource)
        break
      case 'delete':
        await handleDeleteResource(resource)
        break
    }
  }

  const handleSyncResource = async (resource: any) => {
    try {
      resourceLoading.value = true
      await syncProjectApi(resource.id)
      ElMessage.success('同步成功')
      getProjectResources()
    } catch (error: any) {
    } finally {
      resourceLoading.value = false
    }
  }

  const handleDeleteResource = async (resource: any) => {
    try {
      await ElMessageBox.confirm(
        `确定要删除集群 "${resource.clusterName}" 的资源配额吗？`,
        '提示',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      await deleteProjectClusterApi(resource.id)
      ElMessage.success('删除成功')
      getProjectResources()
    } catch (error) {
      if (error !== 'cancel') {
      }
    }
  }

  // 监听Tab切换
  watch(activeTab, (val) => {
    if (val === 'members') {
      getProjectMembers()
    } else if (val === 'resources') {
      getProjectResources()
    } else if (val === 'workspaces') {
      getProjectWorkspaces()
    }
  })

  // 生命周期
  onMounted(() => {
    getProjectInfo()
  })
</script>

<style lang="scss" scoped>
  .project-detail-page {
    padding: 20px;

    .page-header {
      display: flex;
      justify-content: space-between;
      align-items: flex-start;
      margin-bottom: 20px;

      .header-content {
        .project-title {
          display: flex;
          align-items: center;
          gap: 12px;
          margin-top: 12px;

          h2 {
            margin: 0;
            font-size: 20px;
            font-weight: 600;
          }
        }
      }

      .header-actions {
        display: flex;
        gap: 12px;
      }
    }

    .detail-content {
      .detail-tabs {
        :deep(.el-tabs__header) {
          background-color: var(--el-fill-color-blank);
          padding: 0 20px;
          margin-bottom: 0;
        }

        .card-header {
          display: flex;
          justify-content: space-between;
          align-items: center;
        }

        .resource-list {
          .empty-state {
            padding: 40px 0;
          }

          .resource-grid {
            display: grid;
            grid-template-columns: repeat(3, 1fr);
            gap: 16px;

            @media (max-width: 1400px) {
              grid-template-columns: repeat(2, 1fr);
            }

            @media (max-width: 900px) {
              grid-template-columns: 1fr;
            }
          }

          .resource-card {
            border: 1px solid var(--el-border-color-lighter);
            border-radius: 8px;
            padding: 14px 16px;
            background: #fff;
            transition: all 0.2s ease;

            &:hover {
              box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
              border-color: var(--el-color-primary-light-5);
            }

            .resource-header {
              display: flex;
              justify-content: space-between;
              align-items: center;
              margin-bottom: 14px;
              padding-bottom: 10px;
              border-bottom: 1px solid var(--el-border-color-lighter);

              .resource-title {
                display: flex;
                align-items: center;
                gap: 8px;
                min-width: 0;
                flex: 1;
                overflow: hidden;

                .cluster-icon {
                  color: var(--el-color-primary);
                  flex-shrink: 0;
                }

                .cluster-name {
                  font-size: 14px;
                  font-weight: 600;
                  color: var(--el-text-color-primary);
                  overflow: hidden;
                  text-overflow: ellipsis;
                  white-space: nowrap;
                }
              }

              .more-btn {
                flex-shrink: 0;
                padding: 4px;
                margin-left: 8px;
              }
            }

            .resource-content {
              display: flex;
              flex-direction: column;
              gap: 10px;

              .resource-item {
                .label-row {
                  display: flex;
                  justify-content: space-between;
                  align-items: center;
                  margin-bottom: 4px;

                  .label {
                    font-size: 12px;
                    color: var(--el-text-color-regular);
                    font-weight: 500;
                  }

                  .value {
                    font-size: 12px;
                    color: var(--el-text-color-secondary);
                    font-family: var(--ax-font-mono);
                  }
                }

                .progress-row {
                  display: flex;
                  align-items: center;
                  gap: 8px;

                  .el-progress {
                    flex: 1;

                    :deep(.el-progress-bar__outer) {
                      border-radius: 3px;
                    }

                    :deep(.el-progress-bar__inner) {
                      border-radius: 3px;
                    }
                  }

                  .percentage {
                    font-size: 11px;
                    color: var(--el-text-color-secondary);
                    min-width: 32px;
                    text-align: right;
                    font-family: var(--ax-font-mono);
                  }
                }

                .resource-detail {
                  display: flex;
                  justify-content: space-between;
                  margin-top: 2px;
                  font-size: 11px;

                  .overcommit {
                    color: var(--el-color-warning);
                  }

                  .capacity {
                    color: var(--el-text-color-placeholder);
                  }
                }
              }
            }

            .resource-footer {
              margin-top: 12px;
              padding-top: 10px;
              border-top: 1px dashed var(--el-border-color-lighter);
              display: flex;
              align-items: center;
              gap: 4px;
              font-size: 11px;
              color: var(--el-text-color-placeholder);
            }
          }
        }

        .resource-summary {
          display: flex;
          align-items: center;
          font-size: 12px;
          color: var(--el-text-color-regular);
        }
      }
    }
  }
</style>
