<template>
  <div class="project-page art-full-height">
    <ProjectSearch
      v-show="showSearchBar"
      v-model="searchForm"
      @search="handleSearch"
      @reset="resetSearchParams"
    />

    <ElCard
      class="art-table-card"
      shadow="never"
      :style="{ 'margin-top': showSearchBar ? '12px' : '0' }"
    >
      <ArtTableHeader
        v-model:columns="columns"
        v-model:showSearchBar="showSearchBar"
        :loading="loading"
        :showZebra="false"
        :showBorder="true"
        :showHeaderBackground="true"
        :fullClass="'art-page-view'"
        :layout="'search,refresh,size,fullscreen,columns,settings'"
        @refresh="refreshData"
      >
        <template #left>
          <ElSpace wrap>
            <ElButton type="primary" @click="handleAddProject" :disabled="submitLoading">
              新建项目
            </ElButton>
          </ElSpace>
        </template>
      </ArtTableHeader>

      <ArtTable
        :loading="loading"
        :data="data"
        :columns="columns"
        :pagination="pagination"
        :stripe="false"
        @pagination:size-change="handleSizeChange"
        @pagination:current-change="handleCurrentChange"
      />
    </ElCard>

    <ProjectDialog
      v-model:visible="dialogVisible"
      :editData="editData"
      :submitLoading="submitLoading"
      @submit="handleSubmit"
    />

    <ElDialog
      v-model="memberDialogVisible"
      title="项目成员"
      width="640px"
      align-center
      :close-on-click-modal="false"
      @open="loadMemberOptions"
    >
      <ElForm label-width="90px">
        <ElFormItem label="项目名称">
          <ElInput :model-value="currentProject?.name || '-'" disabled />
        </ElFormItem>
        <ElFormItem label="成员列表">
          <ElSelect
            v-model="selectedMemberIds"
            multiple
            filterable
            clearable
            style="width: 100%"
            placeholder="请选择项目成员"
            :loading="memberLoading"
          >
            <ElOption
              v-for="user in userOptions"
              :key="user.id"
              :label="`${user.nickname || user.username} (${user.username})`"
              :value="user.id"
            />
          </ElSelect>
        </ElFormItem>
        <ElFormItem label="平台授权">
          <ElTable
            :data="selectedMemberRows"
            border
            size="small"
            class="member-platform-table"
            style="width: 100%"
            max-height="320"
            empty-text="请先选择项目成员"
          >
            <ElTableColumn label="成员" min-width="160">
              <template #default="{ row }">
                {{ row.nickname || row.username }} ({{ row.username }})
              </template>
            </ElTableColumn>
            <ElTableColumn label="可访问平台" min-width="260">
              <template #default="{ row }">
                <div class="member-platform-cell">
                  <div class="member-platform-title">平台范围</div>
                  <ElSelect
                    v-model="memberPlatformMap[row.id]"
                    multiple
                    filterable
                    collapse-tags
                    collapse-tags-tooltip
                    style="width: 100%"
                    placeholder="请选择平台"
                  >
                    <ElOption
                      v-for="platform in memberPlatformOptions"
                      :key="platform.platformId"
                      :label="platform.platformName"
                      :value="platform.platformId"
                    />
                  </ElSelect>
                </div>
              </template>
            </ElTableColumn>
          </ElTable>
        </ElFormItem>
      </ElForm>
      <template #footer>
        <ElButton @click="memberDialogVisible = false" :disabled="memberSaving">取消</ElButton>
        <ElButton type="primary" :loading="memberSaving" @click="handleSaveMembers">确定</ElButton>
      </template>
    </ElDialog>

    <ElDialog
      v-model="platformDialogVisible"
      title="项目平台"
      width="760px"
      align-center
      :close-on-click-modal="false"
      @open="loadPlatformOptions"
    >
      <div class="platform-auth-dialog">
        <div class="platform-auth-summary">
          <div>
            <div class="summary-label">当前项目</div>
            <div class="summary-title">{{ currentProject?.name || '-' }}</div>
          </div>
          <ElTag type="primary" effect="plain">
            已授权 {{ selectedPlatformIds.length }} / {{ platformOptions.length }}
          </ElTag>
        </div>
        <ElSkeleton v-if="platformLoading" :rows="4" animated />
        <ElCheckboxGroup v-else v-model="selectedPlatformIds" class="platform-auth-grid">
          <label
            v-for="platform in platformOptions"
            :key="platform.id"
            class="platform-auth-card"
            :class="{
              'is-checked': selectedPlatformIds.includes(platform.id),
              'is-required': isPortalPlatform(platform)
            }"
          >
            <ElCheckbox :value="platform.id" :disabled="isPortalPlatform(platform)" />
            <div class="platform-auth-main">
              <span class="platform-auth-name">{{ platform.platformName }}</span>
              <span class="platform-auth-desc">
                {{ platform.platformDesc || platform.platformCode }}
              </span>
            </div>
            <ElTag v-if="isPortalPlatform(platform)" size="small" type="success" effect="plain">
              默认
            </ElTag>
            <ElTag v-else size="small" effect="plain">{{ platform.platformCode }}</ElTag>
          </label>
        </ElCheckboxGroup>
      </div>
      <template #footer>
        <ElButton @click="platformDialogVisible = false" :disabled="platformSaving">取消</ElButton>
        <ElButton type="primary" :loading="platformSaving" @click="handleSavePlatforms">
          确定
        </ElButton>
      </template>
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, ref, watch } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import ProjectSearch from './modules/project-search.vue'
  import ProjectDialog from './modules/project-dialog.vue'
  import { useProjectTableConfig } from './modules/table-config'
  import {
    bindProjectPlatformApi,
    createProjectApi,
    deleteProjectApi,
    getProjectPlatformsApi,
    listProjectMembersApi,
    listProjectsApi,
    setProjectMembersApi,
    unbindProjectPlatformApi,
    updateProjectApi,
    type ProjectPlatformBinding,
    type PortalProject
  } from '@/api/portal/project'
  import { getAllEnabledPlatformsApi, type PlatformSysPlatform } from '@/api/portal/platform'
  import { searchUserApi, type UserSysUser } from '@/api'

  defineOptions({ name: 'SystemProjectManagement' })

  const { loading, showSearchBar, deleteLoadingMap, submitLoading, createTableColumns } =
    useProjectTableConfig()

  const dialogVisible = ref(false)
  const editData = ref<any>(null)
  const data = ref<PortalProject[]>([])
  const currentProject = ref<PortalProject | null>(null)
  const memberDialogVisible = ref(false)
  const platformDialogVisible = ref(false)
  const memberLoading = ref(false)
  const memberSaving = ref(false)
  const platformLoading = ref(false)
  const platformSaving = ref(false)
  const userOptions = ref<UserSysUser[]>([])
  const platformOptions = ref<PlatformSysPlatform[]>([])
  const memberPlatformOptions = ref<ProjectPlatformBinding[]>([])
  const selectedMemberIds = ref<number[]>([])
  const selectedPlatformIds = ref<number[]>([])
  const originalPlatformIds = ref<number[]>([])
  const memberRoleMap = ref<Record<number, string>>({})
  const memberPlatformMap = ref<Record<number, number[]>>({})
  const portalPlatformCode = 'portal'

  const searchForm = ref({
    name: undefined as string | undefined,
    uuid: undefined as string | undefined
  })

  const searchParams = ref<any>({})

  const pagination = ref({
    current: 1,
    size: 10,
    total: 0
  })

  const handleEditProject = (row: PortalProject) => {
    if (row.isSystem === 1) {
      ElMessage.warning('系统项目不允许编辑')
      return
    }
    if (submitLoading.value) return
    editData.value = row
    dialogVisible.value = true
  }

  const handleDeleteProject = async (id: number) => {
    if (submitLoading.value) return
    const project = data.value.find((item) => item.id === id)
    if (project && project.isSystem === 1) {
      ElMessage.warning('系统项目不允许删除')
      return
    }

    try {
      await ElMessageBox.confirm(
        '确定要删除该项目吗？删除后相关的资源配置也会被清除。',
        '删除确认',
        {
          confirmButtonText: '确定',
          cancelButtonText: '取消',
          type: 'warning'
        }
      )

      deleteLoadingMap.value[id] = true
      await deleteProjectApi({ id })
      ElMessage.success('删除成功')
      refreshData()
    } catch {
      // 用户取消或请求失败
    } finally {
      deleteLoadingMap.value[id] = false
    }
  }

  const handleProjectMembers = async (row: PortalProject) => {
    if (row.isSystem === 1) {
      ElMessage.warning('平台项目不允许维护成员')
      return
    }
    currentProject.value = row
    selectedMemberIds.value = []
    memberPlatformOptions.value = []
    memberPlatformMap.value = {}
    memberDialogVisible.value = true
  }

  const handleProjectPlatforms = async (row: PortalProject) => {
    if (row.isSystem === 1) {
      ElMessage.warning('平台项目不允许修改平台授权')
      return
    }
    currentProject.value = row
    selectedPlatformIds.value = []
    originalPlatformIds.value = []
    platformDialogVisible.value = true
  }

  const { columns } = createTableColumns({
    handleEditProject,
    handleDeleteProject,
    handleProjectMembers,
    handleProjectPlatforms
  })

  const selectedMemberRows = computed(() =>
    selectedMemberIds.value
      .map((userId) => userOptions.value.find((user) => user.id === userId))
      .filter((user): user is UserSysUser => Boolean(user))
  )

  const defaultMemberPlatformIds = () =>
    memberPlatformOptions.value.map((platform) => platform.platformId)

  const portalPlatformId = computed(() => {
    return platformOptions.value.find((platform) => isPortalPlatform(platform))?.id
  })

  const portalMemberPlatformId = computed(() => {
    return memberPlatformOptions.value.find(
      (platform) => String(platform.platformCode).toLowerCase() === portalPlatformCode
    )?.platformId
  })

  const isPortalPlatform = (platform: Pick<PlatformSysPlatform, 'platformCode'>) => {
    return String(platform.platformCode).toLowerCase() === portalPlatformCode
  }

  const ensurePortalPlatformSelected = () => {
    const id = portalPlatformId.value
    if (id && !selectedPlatformIds.value.includes(id)) {
      selectedPlatformIds.value = [id, ...selectedPlatformIds.value]
    }
  }

  const ensureMemberPortalPlatform = (userId: number) => {
    const id = portalMemberPlatformId.value
    if (!id) return
    const values = memberPlatformMap.value[userId] || []
    if (!values.includes(id)) {
      memberPlatformMap.value[userId] = [id, ...values]
    }
  }

  watch(selectedMemberIds, (ids) => {
    const selected = new Set(ids)
    Object.keys(memberPlatformMap.value).forEach((key) => {
      const userId = Number(key)
      if (!selected.has(userId)) {
        delete memberPlatformMap.value[userId]
      }
    })
    ids.forEach((userId) => {
      if (!memberPlatformMap.value[userId]) {
        memberPlatformMap.value[userId] = defaultMemberPlatformIds()
      }
      ensureMemberPortalPlatform(userId)
      if (!memberRoleMap.value[userId]) {
        memberRoleMap.value[userId] = 'member'
      }
    })
  })

  const getData = async () => {
    try {
      loading.value = true
      const params = {
        page: pagination.value.current,
        pageSize: pagination.value.size,
        ...searchParams.value
      }
      const response = await listProjectsApi(params)
      data.value = response.data || []
      pagination.value.total = response.total || 0
    } catch {
      // 请求失败
    } finally {
      loading.value = false
    }
  }

  const refreshData = () => {
    getData()
  }

  const handleSearch = (params: Record<string, any>) => {
    searchParams.value = params
    pagination.value.current = 1
    getData()
  }

  const resetSearchParams = () => {
    searchParams.value = {}
    pagination.value.current = 1
    getData()
  }

  const handleSizeChange = (size: number) => {
    pagination.value.size = size
    pagination.value.current = 1
    getData()
  }

  const handleCurrentChange = (current: number) => {
    pagination.value.current = current
    getData()
  }

  const handleAddProject = () => {
    if (submitLoading.value) return
    editData.value = null
    dialogVisible.value = true
  }

  const handleSubmit = async (formData: any) => {
    submitLoading.value = true
    try {
      if (formData.id) {
        await updateProjectApi({
          id: formData.id,
          name: formData.name,
          description: formData.description
        })
        ElMessage.success('更新成功')
      } else {
        await createProjectApi({
          name: formData.name,
          description: formData.description
        })
        ElMessage.success('创建成功')
      }
      dialogVisible.value = false
      refreshData()
    } catch {
      // 请求失败
    } finally {
      submitLoading.value = false
    }
  }

  const loadMemberOptions = async () => {
    if (!currentProject.value) return
    memberLoading.value = true
    memberRoleMap.value = {}
    memberPlatformMap.value = {}
    memberPlatformOptions.value = []
    try {
      const [userResp, memberResp, platformResp] = await Promise.all([
        searchUserApi({ page: 1, pageSize: 200 }),
        listProjectMembersApi(currentProject.value.id),
        getProjectPlatformsApi(currentProject.value.id)
      ])
      userOptions.value = userResp.items || []
      memberPlatformOptions.value = platformResp.data || []
      const defaultPlatformIds = defaultMemberPlatformIds()
      const members = memberResp.data || []
      selectedMemberIds.value = members.map((item) => item.userId)
      memberRoleMap.value = (memberResp.data || []).reduce(
        (acc, item) => {
          acc[item.userId] = item.role || 'member'
          return acc
        },
        {} as Record<number, string>
      )
      memberPlatformMap.value = members.reduce(
        (acc, item) => {
          const platformIds = (item.platformRoles || []).map((platform) => platform.platformId)
          acc[item.userId] = platformIds.length > 0 ? platformIds : defaultPlatformIds
          return acc
        },
        {} as Record<number, number[]>
      )
      selectedMemberIds.value.forEach(ensureMemberPortalPlatform)
    } finally {
      memberLoading.value = false
    }
  }

  const handleSaveMembers = async () => {
    if (!currentProject.value) return
    memberSaving.value = true
    try {
      selectedMemberIds.value.forEach(ensureMemberPortalPlatform)
      await setProjectMembersApi(
        currentProject.value.id,
        selectedMemberIds.value.map((userId) => ({
          userId,
          role: memberRoleMap.value[userId] || 'member',
          platformRoles: (memberPlatformMap.value[userId] || []).map((platformId) => ({
            platformId,
            role: memberRoleMap.value[userId] || 'member'
          }))
        }))
      )
      ElMessage.success('成员保存成功')
      memberDialogVisible.value = false
      refreshData()
    } finally {
      memberSaving.value = false
    }
  }

  const loadPlatformOptions = async () => {
    if (!currentProject.value) return
    platformLoading.value = true
    selectedPlatformIds.value = []
    originalPlatformIds.value = []
    try {
      const [platforms, bindingResp] = await Promise.all([
        getAllEnabledPlatformsApi(),
        getProjectPlatformsApi(currentProject.value.id)
      ])
      platformOptions.value = platforms || []
      selectedPlatformIds.value = (bindingResp.data || []).map((item) => item.platformId)
      ensurePortalPlatformSelected()
      originalPlatformIds.value = [...selectedPlatformIds.value]
    } finally {
      platformLoading.value = false
    }
  }

  const handleSavePlatforms = async () => {
    if (!currentProject.value) return
    platformSaving.value = true
    try {
      ensurePortalPlatformSelected()
      const next = new Set(selectedPlatformIds.value)
      const prev = new Set(originalPlatformIds.value)
      const adds = selectedPlatformIds.value.filter((id) => !prev.has(id))
      const removes = originalPlatformIds.value.filter(
        (id) => !next.has(id) && id !== portalPlatformId.value
      )

      await Promise.all([
        ...adds.map((platformId) => bindProjectPlatformApi(currentProject.value!.id, platformId)),
        ...removes.map((platformId) =>
          unbindProjectPlatformApi(currentProject.value!.id, platformId)
        )
      ])
      ElMessage.success('平台保存成功')
      platformDialogVisible.value = false
      refreshData()
    } finally {
      platformSaving.value = false
    }
  }

  onMounted(() => {
    getData()
  })
</script>

<style lang="scss" scoped>
  .project-page {
    padding-bottom: 15px;
  }

  .platform-auth-dialog {
    display: flex;
    flex-direction: column;
    gap: 16px;
  }

  :deep(.member-platform-table) {
    .el-table__cell {
      vertical-align: top;
    }
  }

  .member-platform-cell {
    display: flex;
    flex-direction: column;
    gap: 8px;
    padding: 2px 0;
  }

  .member-platform-title {
    font-size: 12px;
    color: var(--el-text-color-secondary);
    line-height: 1;
  }

  .platform-auth-summary {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    background: var(--el-fill-color-lighter);
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
  }

  .summary-label {
    margin-bottom: 4px;
    font-size: 12px;
    color: var(--el-text-color-secondary);
  }

  .summary-title {
    font-size: 16px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .platform-auth-grid {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
    max-height: 420px;
    overflow-y: auto;
  }

  .platform-auth-card {
    display: flex;
    align-items: center;
    gap: 12px;
    min-height: 76px;
    padding: 14px;
    cursor: pointer;
    border: 1px solid var(--el-border-color-lighter);
    border-radius: 8px;
    transition:
      border-color 0.2s,
      background-color 0.2s,
      box-shadow 0.2s;

    &:hover {
      border-color: var(--el-color-primary-light-5);
      background: var(--el-color-primary-light-9);
    }

    &.is-checked {
      border-color: var(--el-color-primary);
      background: var(--el-color-primary-light-9);
      box-shadow: 0 0 0 1px var(--el-color-primary-light-7);
    }

    :deep(.el-checkbox) {
      height: auto;
      margin-right: 0;
    }
  }

  .platform-auth-main {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-width: 0;
    gap: 6px;
  }

  .platform-auth-name {
    font-size: 14px;
    font-weight: 600;
    color: var(--el-text-color-primary);
  }

  .platform-auth-desc {
    overflow: hidden;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    text-overflow: ellipsis;
    white-space: nowrap;
  }
</style>
