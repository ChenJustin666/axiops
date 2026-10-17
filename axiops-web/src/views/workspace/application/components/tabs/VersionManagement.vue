<template>
  <div class="version-management">
    <!-- 版本卡片网格 -->
    <div class="versions-grid">
      <div class="version-card add-card" @click="handleAddVersion">
        <Plus :size="24" />
        <span>发布新版本</span>
      </div>

      <!-- 删除服务按钮 - 仅在没有版本时显示 -->
      <div
        v-if="versions.length === 0 && !loading"
        class="version-card delete-card"
        @click="handleDeleteApplication"
      >
        <Trash2 :size="24" />
        <span>删除服务</span>
      </div>

      <div
        v-for="version in versions"
        :key="version.id"
        class="version-card"
        :class="{
          active: selectedVersion?.id === version.id,
          abnormal: !version.status || version.status === 0,
          'role-stable': version.versionRole === 'stable',
          'role-primary': version.versionRole === 'primary',
          'role-canary': version.versionRole === 'canary',
          'role-blue': version.versionRole === 'blue',
          'role-green': version.versionRole === 'green'
        }"
        @click="handleSelectVersion(version)"
      >
        <!-- 第一行：版本号 + 操作按钮 -->
        <div class="card-header">
          <div class="version-title">
            <Tag :size="16" />
            <ElTooltip
              :content="version.version"
              placement="top"
              :disabled="version.version.length <= 28"
              :show-after="300"
            >
              <span class="version-text">{{ version.version }}</span>
            </ElTooltip>
          </div>
          <!-- 修复：移除 v-if 条件，让所有版本都显示下拉菜单 -->
          <ElDropdown trigger="click" @command="(cmd) => handleCommand(cmd, version)">
            <span class="more-btn-wrapper" @click.stop>
              <ElButton class="more-btn" size="small" text circle>
                <MoreVertical :size="16" />
              </ElButton>
            </span>
            <template #dropdown>
              <ElDropdownMenu>
                <!-- 仅 stable 版本显示编辑、复制、迁移、创建新服务按钮 -->
                <template v-if="version.versionRole === 'stable'">
                  <ElDropdownItem command="edit">
                    <Edit :size="14" />
                    编辑
                  </ElDropdownItem>
                  <ElDropdownItem command="copy">
                    <Copy :size="14" />
                    复制
                  </ElDropdownItem>
                  <ElDropdownItem command="createNewService" divided>
                    <Plus :size="14" />
                    创建新服务
                  </ElDropdownItem>
                  <ElDropdownItem command="migrate">
                    <ArrowRightLeft :size="14" />
                    迁移版本
                  </ElDropdownItem>
                  <ElDropdownItem command="delete" divided>
                    <Trash2 :size="14" />
                    删除版本
                  </ElDropdownItem>
                </template>
                <!-- 非 stable 版本只显示删除按钮 -->
                <template v-else>
                  <ElDropdownItem command="delete">
                    <Trash2 :size="14" />
                    删除版本
                  </ElDropdownItem>
                </template>
              </ElDropdownMenu>
            </template>
          </ElDropdown>
        </div>

        <!-- 第二行：资源名称 -->
        <div v-if="version.resourceName" class="resource-name">
          <ElTooltip
            :content="version.resourceName"
            placement="top"
            :disabled="version.resourceName.length <= 32"
            :show-after="300"
          >
            <span class="resource-name-text">{{ version.resourceName }}</span>
          </ElTooltip>
        </div>

        <!-- 右下角：版本角色标识 -->
        <div v-if="version.versionRole" class="version-role-corner">
          <ElTag :type="getVersionRoleTagType(version.versionRole)" size="small" effect="dark">
            {{ getVersionRoleLabel(version.versionRole) }}
          </ElTag>
        </div>

        <!-- 右下角：异常状态标识（异常时覆盖版本角色） -->
        <ElTooltip
          v-if="!version.status || version.status === 0"
          content="服务可能异常，请查看事件中心事件，或者后端服务被删除请联系管理员"
          placement="top"
        >
          <ElTag type="danger" size="small" effect="dark" class="status-tag-corner"> 异常 </ElTag>
        </ElTooltip>
      </div>
    </div>

    <!-- 版本详情 - 根据资源类型动态加载对应的管理组件 -->
    <div v-if="selectedVersion" class="version-detail">
      <component
        :is="currentResourceComponent"
        :key="selectedVersion.id"
        :version="selectedVersion"
        :application="application"
        :cluster="cluster"
        :workspace="workspace"
        :refresh-trigger="versionRefreshTrigger"
        @refresh="handleRefresh"
      />
    </div>

    <!-- 空状态 -->
    <div v-else-if="!loading && versions.length === 0" class="empty-state">
      <Layers :size="48" style="color: #c0c4cc" />
      <p>暂无版本，请新增一个版本</p>
    </div>

    <ElDialog
      v-model="migrateDialogVisible"
      title="迁移版本"
      width="500px"
      :close-on-click-modal="false"
    >
      <ElForm :model="migrateForm" label-width="100px">
        <ElFormItem label="源版本">
          <ElInput :value="migratingVersion?.version" disabled />
        </ElFormItem>
        <ElFormItem label="新版本名称" required>
          <ElInput v-model="migrateForm.newVersionName" placeholder="请输入新版本名称" clearable />
        </ElFormItem>
        <ElFormItem label="目标服务" required>
          <ElSelect
            v-model="migrateForm.newApplicationId"
            placeholder="请选择目标服务"
            filterable
            style="width: 100%"
          >
            <ElOption
              v-for="app in targetApplications"
              :key="app.id"
              :label="app.nameCn || app.nameEn"
              :value="app.id"
            >
              <div class="app-option">
                <span>{{ app.nameCn || app.nameEn }}</span>
                <ElTag :type="getResourceTypeTag(app.resourceType)" size="small">
                  {{ app.resourceType }}
                </ElTag>
              </div>
            </ElOption>
          </ElSelect>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="migrateDialogVisible = false">取消</ElButton>
        <ElButton
          type="primary"
          :loading="migrating"
          :disabled="!migrateForm.newVersionName || !migrateForm.newApplicationId"
          @click="handleMigrateConfirm"
        >
          确认迁移
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, defineAsyncComponent, defineComponent, h, onMounted, ref, watch } from 'vue'
  import { useRouter } from 'vue-router'
  import { ElMessage, ElMessageBox, ElSkeleton } from 'element-plus'
  import {
    Plus,
    Tag,
    MoreVertical,
    Edit,
    Copy,
    Trash2,
    Layers,
    ArrowRightLeft
  } from 'lucide-vue-next'
  import { useProjectStore } from '@/store/modules/project'
  import { useApplicationManagementStore } from '@/store/modules/applicationManagement'
  import {
    searchVersionApi,
    deleteVersionApi,
    deleteApplicationApi,
    searchApplicationApi,
    migrateVersionOnProjectApi,
    type OnecProjectApplication,
    type OnecProjectVersion,
    type ProjectCluster,
    type ProjectWorkspace
  } from '@/api'

  defineOptions({ name: 'VersionManagement' })

  // 设置为 true 表示开启多版本管理，允许用户创建多个版本
  const ENABLE_MULTI_VERSION = true

  interface Props {
    application: OnecProjectApplication
    cluster: ProjectCluster | null
    workspace: ProjectWorkspace | null
    refreshTrigger: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{
    refresh: []
    applicationDeleted: []
  }>()

  const ResourceLoading = defineComponent({
    name: 'VersionResourceLoading',
    setup() {
      return () => h('div', { style: 'padding: 16px;' }, [h(ElSkeleton, { rows: 8, animated: true })])
    }
  })

  const createAsyncResourceComponent = (loader: () => Promise<any>) =>
    defineAsyncComponent({
      loader,
      delay: 120,
      loadingComponent: ResourceLoading
    })

  const AsyncDeploymentManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/DeploymentManagement.vue')
  )
  const AsyncStatefulSetManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/StatefulSetManagement.vue')
  )
  const AsyncDaemonSetManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/DaemonSetManagement.vue')
  )
  const AsyncJobManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/JobManagement.vue')
  )
  const AsyncCronJobManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/CronJobManagement.vue')
  )
  const AsyncPodResourceManagement = createAsyncResourceComponent(
    () => import('../version/resource-management/PodResourceManagement.vue')
  )

  const router = useRouter()
  const projectStore = useProjectStore()
  const managementStore = useApplicationManagementStore()
  const versions = ref<OnecProjectVersion[]>([])
  const selectedVersion = ref<OnecProjectVersion | null>(null)
  const loading = ref(false)
  const deleting = ref(false)
  const versionRefreshTrigger = ref(0)
  const isInitialized = ref(false)
  const lastVersionRequestKey = ref('')

  const migrateDialogVisible = ref(false)
  const migrating = ref(false)
  const migratingVersion = ref<OnecProjectVersion | null>(null)
  const targetApplications = ref<OnecProjectApplication[]>([])
  const migrateForm = ref({
    newVersionName: '',
    newApplicationId: null as number | null
  })

  const normalizeResourceType = (type: string): string => {
    return type?.toLowerCase() || ''
  }

  const currentResourceComponent = computed(() => {
    const resourceType = normalizeResourceType(props.application.resourceType)
    const componentMap: Record<string, any> = {
      pod: AsyncPodResourceManagement,
      deployment: AsyncDeploymentManagement,
      statefulset: AsyncStatefulSetManagement,
      daemonset: AsyncDaemonSetManagement,
      cronjob: AsyncCronJobManagement,
      job: AsyncJobManagement
    }
    return componentMap[resourceType]
  })

  const getRouteName = (resourceType: string): string => {
    const type = normalizeResourceType(resourceType)
    const routeNameMap: Record<string, string> = {
      pod: 'CreatePod',
      deployment: 'CreateDeployment',
      statefulset: 'CreateStatefulSet',
      daemonset: 'CreateDaemonSet',
      job: 'CreateJob',
      cronjob: 'CreateCronJob'
    }
    return routeNameMap[type] || ''
  }

  const getResourceTypeTag = (type: string) => {
    const resourceType = normalizeResourceType(type)
    const tagMap: Record<string, string> = {
      pod: 'info',
      deployment: 'success',
      statefulset: 'warning',
      daemonset: 'primary',
      cronjob: 'danger',
      job: ''
    }
    return tagMap[resourceType] || 'info'
  }

  const getVersionRoleTagType = (role: string): string => {
    const typeMap: Record<string, string> = {
      stable: 'success', // 用户发布的稳定版本
      primary: 'primary', // Flagger金丝雀主版本
      canary: 'warning', // 金丝雀测试版本
      blue: 'info', // 蓝版本
      green: '' // 绿版本
    }
    return typeMap[role] || 'info'
  }

  const getVersionRoleLabel = (role: string): string => {
    const labelMap: Record<string, string> = {
      stable: '稳定版本', // 用户发布的主版本
      primary: '金丝雀主版本', // Flagger管理的主版本
      canary: '金丝雀测试', // 金丝雀测试版本
      blue: '蓝版本',
      green: '绿版本'
    }
    return labelMap[role] || role
  }

  const getVersionRequestKey = (refreshToken = props.refreshTrigger) => {
    const clusterUuid = props.cluster?.clusterUuid || props.cluster?.uuid || 'no-cluster'
    return `${props.application.id}:${props.workspace?.id || 0}:${clusterUuid}:${refreshToken}`
  }

  const loadVersions = async (
    options: {
      force?: boolean
      requestKey?: string
    } = {}
  ) => {
    const { force = false, requestKey = getVersionRequestKey() } = options
    if (!props.application?.id || loading.value || (!force && requestKey === lastVersionRequestKey.value)) {
      return
    }

    lastVersionRequestKey.value = requestKey
    loading.value = true
    try {
      const response = await searchVersionApi({ applicationId: props.application.id })
      versions.value = response || []

      if (versions.value.length === 0) {
        selectedVersion.value = null
        return
      }

      if (selectedVersion.value?.id) {
        const matchedVersion = versions.value.find((item) => item.id === selectedVersion.value?.id)
        if (matchedVersion) {
          selectedVersion.value = matchedVersion
          return
        }
      }

      handleSelectVersion(versions.value[0])
    } catch (error) {
      console.error('[VersionManagement] 加载版本失败:', error)
    } finally {
      loading.value = false
    }
  }

  const handleSelectVersion = (version: OnecProjectVersion) => {
    selectedVersion.value = version
    versionRefreshTrigger.value++
  }

  const handleAddVersion = async () => {
    // 检查是否启用多版本管理
    if (!ENABLE_MULTI_VERSION && versions.value.length > 0) {
      try {
        await ElMessageBox.confirm(
          '当前暂未开启多版本管理功能，多版本功能需要通过灰度发布或AB测试等方式操作。',
          '多版本管理提示',
          {
            type: 'warning',
            confirmButtonText: '我知道了',
            showCancelButton: false,
            closeOnClickModal: false
          }
        )
        return
      } catch {
        return
      }
    }

    if (!props.cluster || !props.workspace) {
      return
    }

    const projectId = projectStore.selectedProject?.id
    if (!projectId) {
      return
    }

    const routeName = getRouteName(props.application.resourceType)
    if (!routeName) {
      return
    }

    const clusterUuid =
      props.cluster.clusterUuid || props.cluster.uuid || `cluster-${props.cluster.id}`

    managementStore.markRefreshOnActivate(true)
    router.push({
      name: routeName,
      query: {
        resourceClusterId: String(props.cluster.id),
        clusterUuid: clusterUuid,
        workspaceId: String(props.workspace.id),
        appProjectId: String(projectId),
        namespace: props.workspace.namespace || 'default',
        applicationId: String(props.application.id),
        mode: 'createAppVersion'
      }
    })
  }

  const handleDeleteApplication = async () => {
    if (deleting.value) return

    const appName = props.application.nameCn || props.application.nameEn || '当前服务'

    try {
      await ElMessageBox.confirm(
        `确定要删除服务 "${appName}" 吗？删除后将无法恢复。`,
        '删除服务确认',
        {
          type: 'warning',
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          confirmButtonClass: 'el-button--danger'
        }
      )

      deleting.value = true
      await deleteApplicationApi(props.application.id)

      ElMessage.success('服务删除成功')

      emit('applicationDeleted')
      emit('refresh')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[VersionManagement] 删除应用失败:', error)
      }
    } finally {
      deleting.value = false
    }
  }

  const loadTargetApplications = async () => {
    if (!props.workspace) return

    try {
      const response = await searchApplicationApi({
        workspaceId: props.workspace.id
      })

      const currentResourceType = normalizeResourceType(props.application.resourceType)
      targetApplications.value = (response || []).filter((app) => {
        return (
          app.id !== props.application.id &&
          normalizeResourceType(app.resourceType) === currentResourceType
        )
      })
    } catch (error) {
      console.error('[VersionManagement] 加载目标应用失败:', error)
    }
  }

  const showMigrateDialog = async (version: OnecProjectVersion) => {
    if (!props.workspace) {
      ElMessage.warning('请先选择工作空间')
      return
    }

    migratingVersion.value = version
    migrateForm.value = {
      newVersionName: `${version.version}-copy`,
      newApplicationId: null
    }

    await loadTargetApplications()

    if (targetApplications.value.length === 0) {
      ElMessage.warning('当前工作空间下没有可迁移的目标服务')
      return
    }

    migrateDialogVisible.value = true
  }

  const handleMigrateConfirm = async () => {
    if (!migratingVersion.value || !migrateForm.value.newApplicationId) {
      return
    }

    migrating.value = true
    try {
      await migrateVersionOnProjectApi({
        versionId: migratingVersion.value.id,
        newVersionName: migrateForm.value.newVersionName,
        newApplicationId: migrateForm.value.newApplicationId
      })

      ElMessage.success('版本迁移成功')
      migrateDialogVisible.value = false
      emit('refresh')
    } catch (error) {
      console.error('[VersionManagement] 迁移版本失败:', error)
    } finally {
      migrating.value = false
    }
  }

  const handleCommand = async (command: string, version: OnecProjectVersion) => {
    const projectId = projectStore.selectedProject?.id

    switch (command) {
      case 'edit':
      case 'copy':
      case 'createNewService': {
        if (!props.cluster || !props.workspace) {
          ElMessage.warning('请先选择集群和工作空间')
          return
        }
        if (!projectId) {
          ElMessage.warning('请先选择项目')
          return
        }

        const clusterUuid =
          props.cluster.clusterUuid || props.cluster.uuid || `cluster-${props.cluster.id}`
        const routeName = getRouteName(props.application.resourceType)

        if (!routeName) return

        if (command === 'copy' && !ENABLE_MULTI_VERSION) {
          try {
            await ElMessageBox.confirm(
              '当前暂未开启多版本管理功能，多版本功能需要通过灰度发布或AB测试等方式操作。',
              '多版本管理提示',
              {
                type: 'warning',
                confirmButtonText: '我知道了',
                showCancelButton: false,
                closeOnClickModal: false
              }
            )
          } catch {
            // 用户关闭了对话框
          }
          return
        }

        const baseQuery = {
          resourceClusterId: String(props.cluster.id),
          clusterUuid: clusterUuid,
          workspaceId: String(props.workspace.id),
          appProjectId: String(projectId),
          namespace: props.workspace.namespace || 'default',
          applicationId: String(props.application.id),
          applicationVersionId: String(version.id)
        }

        managementStore.markRefreshOnActivate(true)

        if (command === 'edit') {
          router.push({ name: routeName, query: { ...baseQuery, mode: 'editAppVersion' } })
        } else if (command === 'copy') {
          router.push({ name: routeName, query: { ...baseQuery, mode: 'copyAppVersion' } })
        } else if (command === 'createNewService') {
          router.push({ name: routeName, query: { ...baseQuery, mode: 'createNewService' } })
        }
        break
      }

      case 'migrate':
        await showMigrateDialog(version)
        break

      case 'delete':
        if (deleting.value) return
        try {
          await ElMessageBox.confirm(`确定要删除版本 "${version.version}" 吗？`, '删除版本确认', {
            type: 'warning',
            confirmButtonText: '确定删除',
            cancelButtonText: '取消'
          })
          deleting.value = true
          await deleteVersionApi(version.id)
          ElMessage.success('版本删除成功')
          if (selectedVersion.value?.id === version.id) {
            selectedVersion.value = null
          }
          await loadVersions()
          emit('refresh')
        } catch (error: any) {
          if (error !== 'cancel') {
            console.error('[VersionManagement] 删除版本失败:', error)
          }
        } finally {
          deleting.value = false
        }
        break
    }
  }

  const handleRefresh = () => {
    loadVersions({ force: true })
  }

  watch(
    [() => props.application.id, () => props.workspace?.id, () => props.cluster?.clusterUuid],
    () => {
      selectedVersion.value = null
      isInitialized.value = false
      loadVersions({ force: true, requestKey: getVersionRequestKey() }).then(() => {
        isInitialized.value = true
      })
    }
  )

  watch(
    () => props.refreshTrigger,
    (newVal, oldVal) => {
      if (newVal > 0 && newVal !== oldVal && isInitialized.value) {
        loadVersions({ requestKey: getVersionRequestKey(newVal) })
      }
    }
  )

  onMounted(async () => {
    await loadVersions()
    if (!isInitialized.value) {
      isInitialized.value = true
    }
  })

  defineExpose({ refresh: loadVersions })
</script>

<style lang="scss" scoped>
  .version-management {
    .versions-grid {
      display: grid;
      // 卡片最小宽度加大到 260px，确保内容展示完整
      grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
      gap: 12px;
      margin-bottom: 20px;
      padding: 16px 16px 0 16px;

      .version-card {
        position: relative;
        padding: 14px 16px;
        padding-bottom: 40px; // 底部留空间给版本角色标识
        background: white;
        border: 2px solid #e4e7ed;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;
        min-height: 88px;

        &:hover {
          border-color: #409eff;
          box-shadow: 0 2px 12px rgba(64, 158, 255, 0.1);
          transform: translateY(-1px);
        }

        &.active {
          border-color: #409eff;
          background: #ecf5ff;
          .version-title .version-text {
            color: #409eff;
            font-weight: 600;
          }
        }

        &.role-stable {
          border-color: #67c23a;
          &:hover {
            box-shadow: 0 2px 12px rgba(103, 194, 58, 0.15);
          }
          &.active {
            background: #f0f9ff;
            border-color: #67c23a;
          }
        }

        &.role-primary {
          border-color: #409eff;
          &:hover {
            box-shadow: 0 2px 12px rgba(64, 158, 255, 0.15);
          }
        }

        &.role-canary {
          border-color: #e6a23c;
          &:hover {
            box-shadow: 0 2px 12px rgba(230, 162, 60, 0.15);
          }
          &.active {
            background: #fdf6ec;
            border-color: #e6a23c;
          }
        }

        &.role-blue {
          border-color: #909399;
          &:hover {
            box-shadow: 0 2px 12px rgba(144, 147, 153, 0.15);
          }
        }

        &.role-green {
          border-color: #95de64;
          &:hover {
            box-shadow: 0 2px 12px rgba(149, 222, 100, 0.15);
          }
        }

        &.abnormal {
          border-color: #f56c6c;

          &:hover {
            box-shadow: 0 2px 12px rgba(245, 108, 108, 0.15);
          }

          &.active {
            background: #fef0f0;
          }

          // 异常时隐藏版本角色标识，优先显示异常标识
          .version-role-corner {
            display: none;
          }
        }

        &.add-card {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 8px;
          border-style: dashed;
          color: #909399;
          font-size: 14px;
          padding-bottom: 14px; // 添加卡片不需要底部额外空间
          &:hover {
            color: #409eff;
            background: #f5f7fa;
          }
        }

        &.delete-card {
          display: flex;
          flex-direction: column;
          align-items: center;
          justify-content: center;
          gap: 8px;
          border-style: dashed;
          border-color: #f56c6c;
          color: #f56c6c;
          font-size: 14px;
          background: #fef0f0;
          padding-bottom: 14px; // 删除卡片不需要底部额外空间

          &:hover {
            border-color: #f56c6c;
            background: #fde2e2;
            box-shadow: 0 2px 12px rgba(245, 108, 108, 0.15);
          }
        }

        // 第一行：版本号 + 操作按钮
        .card-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 8px;
          gap: 8px;

          .version-title {
            display: flex;
            align-items: center;
            gap: 6px;
            font-size: 14px;
            font-weight: 600;
            color: #303133;
            flex: 1;
            min-width: 0;
            overflow: hidden;

            .version-text {
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
              display: block;
              cursor: pointer;
              transition: color 0.2s;

              &:hover {
                color: #409eff;
              }
            }
          }

          .more-btn-wrapper {
            display: inline-flex;
            flex-shrink: 0;
          }

          .more-btn {
            opacity: 0;
            transition: opacity 0.2s;
          }
        }

        &:hover .more-btn {
          opacity: 1;
        }

        // 第二行：资源名称
        .resource-name {
          padding-left: 22px; // 与版本号文字对齐（Tag 图标宽度 16px + gap 6px）

          .resource-name-text {
            font-size: 13px;
            color: #606266;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            display: block;
            cursor: pointer;
            line-height: 1.4;

            &:hover {
              color: #409eff;
            }
          }
        }

        // 右下角：版本角色标识
        .version-role-corner {
          position: absolute;
          right: 10px;
          bottom: 10px;
          z-index: 1;

          .el-tag {
            font-size: 11px;
            height: 22px;
            line-height: 22px;
            padding: 0 10px;
            border-radius: 4px;
            font-weight: 500;
          }
        }

        // 右下角：异常状态标识
        .status-tag-corner {
          position: absolute;
          right: 10px;
          bottom: 10px;
          font-size: 11px;
          height: 22px;
          line-height: 22px;
          padding: 0 10px;
          border-radius: 4px;
          z-index: 2;
          box-shadow: 0 2px 6px rgba(245, 108, 108, 0.35);
          font-weight: 500;
          cursor: pointer;
        }
      }
    }

    .version-detail {
      // 由各资源管理组件自己控制样式
    }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      padding: 60px 0;
      color: #909399;
      p {
        margin-top: 12px;
        font-size: 14px;
      }
    }

    .app-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;
    }
  }

  :deep(.el-tooltip__popper) {
    max-width: 400px;
    word-break: break-all;
  }
</style>
