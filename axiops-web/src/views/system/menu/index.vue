<!-- 菜单管理页面 - 多平台版本 -->
<template>
  <div class="menu-page-container flex flex-row gap-4 art-full-height">
    <!-- 左侧平台列表 -->
    <div class="platform-panel flex flex-col w-64 min-w-64 bg-box rounded-lg overflow-hidden">
      <!-- 平台列表头部 -->
      <div class="platform-header flex-b px-4 py-3 border-b border-[var(--art-border-color)]">
        <span class="font-medium">平台列表</span>
        <ElButton type="primary" size="small" @click="handleAddPlatform" v-ripple>
          <ArtSvgIcon icon="ri:add-line" :size="14" class="mr-1" />
          新增
        </ElButton>
      </div>

      <!-- 平台列表内容 -->
      <div class="platform-list flex-1 overflow-y-auto">
        <div v-if="platformLoading" class="flex-center py-8">
          <ElIcon class="is-loading" :size="24">
            <Loading />
          </ElIcon>
        </div>

        <div
          v-else-if="platformList.length === 0"
          class="flex-center flex-col py-8 text-[var(--art-text-secondary)]"
        >
          <ArtSvgIcon icon="ri:apps-2-line" :size="48" class="mb-2 opacity-50" />
          <span class="text-sm">暂无平台数据</span>
        </div>

        <div v-else class="py-2">
          <div
            v-for="platform in platformList"
            :key="platform.id"
            class="platform-item flex-b px-4 py-3 mx-2 mb-1 rounded-md cursor-pointer transition-all"
            :class="[
              selectedPlatformId === platform.id ? 'is-active' : '',
              platform.isEnable !== 1 ? 'is-disabled' : ''
            ]"
            @click="handleSelectPlatform(platform)"
          >
            <div class="flex-c gap-2 min-w-0">
              <ArtSvgIcon
                v-if="platform.platformIcon"
                :icon="platform.platformIcon"
                :size="20"
                class="flex-shrink-0"
              />
              <ArtSvgIcon v-else icon="ri:apps-2-line" :size="20" class="flex-shrink-0" />
              <span class="truncate">{{ platform.platformName }}</span>
            </div>
            <div class="flex-c gap-1 flex-shrink-0">
              <ElTag v-if="platform.isDefault === 1" type="primary" size="small" effect="plain">
                默认
              </ElTag>
              <ElTag v-if="platform.isEnable !== 1" type="info" size="small" effect="plain">
                禁用
              </ElTag>
              <ElDropdown
                trigger="click"
                @command="(cmd: string) => handlePlatformAction(cmd, platform)"
                @click.stop
              >
                <ElButton text size="small" class="platform-more-btn" @click.stop>
                  <ArtSvgIcon icon="ri:more-2-fill" :size="16" />
                </ElButton>
                <template #dropdown>
                  <ElDropdownMenu>
                    <ElDropdownItem command="edit">
                      <ArtSvgIcon icon="ri:edit-line" :size="14" class="mr-2" />
                      编辑
                    </ElDropdownItem>
                    <ElDropdownItem v-if="platform.isEnable === 1" command="disable">
                      <ArtSvgIcon icon="ri:forbid-line" :size="14" class="mr-2" />
                      禁用
                    </ElDropdownItem>
                    <ElDropdownItem v-else command="enable">
                      <ArtSvgIcon icon="ri:checkbox-circle-line" :size="14" class="mr-2" />
                      启用
                    </ElDropdownItem>
                    <ElDropdownItem v-if="platform.isDefault !== 1" command="setDefault">
                      <ArtSvgIcon icon="ri:star-line" :size="14" class="mr-2" />
                      设为默认
                    </ElDropdownItem>
                    <ElDropdownItem command="delete" divided>
                      <span class="text-danger flex-c">
                        <ArtSvgIcon icon="ri:delete-bin-line" :size="14" class="mr-2" />
                        删除
                      </span>
                    </ElDropdownItem>
                  </ElDropdownMenu>
                </template>
              </ElDropdown>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- 右侧菜单管理 -->
    <div class="menu-panel flex-1 flex flex-col min-w-0 overflow-hidden">
      <!-- 未选择平台时的提示 -->
      <div
        v-if="!selectedPlatformId"
        class="empty-state-container flex-center h-full bg-box rounded-lg"
      >
        <div class="empty-state-content">
          <ArtSvgIcon icon="ri:cursor-line" :size="48" class="empty-icon" />
          <h3 class="empty-title">选择平台</h3>
          <p class="empty-desc">从左侧列表选择一个平台来管理菜单</p>
        </div>
      </div>

      <!-- 选择平台后显示菜单管理 -->
      <template v-else>
        <!-- 搜索栏 -->
        <ArtSearchBar
          v-model="formFilters"
          :items="formItems"
          :showExpand="false"
          @reset="handleReset"
          @search="handleSearch"
        />

        <ElCard class="art-table-card flex-1 flex flex-col overflow-hidden" shadow="never">
          <!-- 表格头部 -->
          <ArtTableHeader
            :showZebra="false"
            :loading="menuLoading"
            v-model:columns="columnChecks"
            @refresh="handleRefresh"
          >
            <template #left>
              <div class="flex-c gap-2">
                <ElTag type="info" effect="plain" class="!mr-2">
                  {{ selectedPlatform?.platformName }}
                </ElTag>
                <ElButton @click="handleAddMenu" v-ripple>添加菜单</ElButton>
                <ElButton @click="toggleExpand" v-ripple>
                  {{ isExpanded ? '收起' : '展开' }}
                </ElButton>
              </div>
            </template>
          </ArtTableHeader>

          <div class="flex-1 overflow-hidden">
            <ArtTable
              ref="tableRef"
              rowKey="id"
              :loading="menuLoading"
              :columns="columns"
              :data="filteredTableData"
              :stripe="false"
              :tree-props="{ children: 'children', hasChildren: 'hasChildren' }"
              :default-expand-all="false"
              height="100%"
            />
          </div>

          <!-- 菜单弹窗 -->
          <MenuDialog
            v-model:visible="menuDialogVisible"
            :type="dialogType"
            :editData="editData"
            :parentMenuList="parentMenuList"
            :platformId="selectedPlatformId"
            @submit="handleMenuSubmit"
          />
        </ElCard>
      </template>
    </div>

    <!-- 平台弹窗 -->
    <PlatformDialog
      v-model:visible="platformDialogVisible"
      :editData="editPlatformData"
      @submit="handlePlatformSubmit"
    />
  </div>
</template>

<script setup lang="ts">
  import { formatMenuTitle } from '@/utils/router'
  import ArtButtonTable from '@/components/core/forms/art-button-table/index.vue'
  import { useTableColumns } from '@/hooks/core/useTableColumns'
  import MenuDialog from './modules/menu-dialog.vue'
  import PlatformDialog from './modules/platform-dialog.vue'
  import { Loading } from '@element-plus/icons-vue'
  import {
    searchMenuTreeApi,
    addMenuApi,
    updateMenuApi,
    deleteMenuApi,
    getSysMenuSimpleTreeApi,
    type MenuSysMenuTree,
    type MenuAddRequest,
    type MenuUpdateRequest,
    type MenuSearchRequest
  } from '@/api/portal/menu'
  import {
    searchPlatformApi,
    addPlatformApi,
    updatePlatformApi,
    deletePlatformApi,
    enablePlatformApi,
    disablePlatformApi,
    setDefaultPlatformApi,
    type PlatformSysPlatform,
    type PlatformAddRequest,
    type PlatformUpdateRequest
  } from '@/api/portal/platform'
  import { ElTag, ElMessageBox, ElMessage } from 'element-plus'

  defineOptions({ name: 'MenuManagement' })

  // ========== 平台相关状态 ==========
  const platformLoading = ref(false)
  const platformList = ref<PlatformSysPlatform[]>([])
  const selectedPlatformId = ref<number | null>(null)
  const platformDialogVisible = ref(false)
  const editPlatformData = ref<PlatformSysPlatform | null>(null)

  // 当前选中的平台
  const selectedPlatform = computed(() => {
    if (!selectedPlatformId.value) return null
    return platformList.value.find((p) => p.id === selectedPlatformId.value) || null
  })

  // ========== 菜单相关状态 ==========
  const menuLoading = ref(false)
  const isExpanded = ref(false)
  const tableRef = ref()
  const menuDialogVisible = ref(false)
  const dialogType = ref<1 | 2 | 3>(2)
  const editData = ref<MenuSysMenuTree | null>(null)
  const parentMenuList = ref<any[]>([])
  const tableData = ref<MenuSysMenuTree[]>([])

  // 搜索相关
  const initialSearchState = {
    name: '',
    title: ''
  }

  const formFilters = reactive({ ...initialSearchState })
  const appliedFilters = reactive({ ...initialSearchState })

  const formItems = computed(() => [
    {
      label: '菜单名称',
      key: 'name',
      type: 'input',
      props: { clearable: true, placeholder: '请输入菜单名称' }
    },
    {
      label: '菜单标题',
      key: 'title',
      type: 'input',
      props: { clearable: true, placeholder: '请输入菜单标题' }
    }
  ])

  // ========== 生命周期 ==========
  onMounted(() => {
    getPlatformList()
  })

  // ========== 平台相关方法 ==========

  /**
   * 获取平台列表
   */
  const getPlatformList = async (): Promise<void> => {
    platformLoading.value = true
    try {
      const response = await searchPlatformApi({ page: 1, pageSize: 100 })
      platformList.value = response.items || []

      // 如果有平台且未选择，自动选择第一个启用的平台
      if (platformList.value.length > 0 && !selectedPlatformId.value) {
        const defaultPlatform = platformList.value.find(
          (p) => p.isDefault === 1 && p.isEnable === 1
        )
        const firstEnabled = platformList.value.find((p) => p.isEnable === 1)
        const targetPlatform = defaultPlatform || firstEnabled || platformList.value[0]
        if (targetPlatform) {
          handleSelectPlatform(targetPlatform)
        }
      }
    } catch (error) {
      console.error('获取平台列表失败:', error)
    } finally {
      platformLoading.value = false
    }
  }

  /**
   * 选择平台
   */
  const handleSelectPlatform = (platform: PlatformSysPlatform): void => {
    if (selectedPlatformId.value === platform.id) return

    selectedPlatformId.value = platform.id
    // 重置菜单数据
    tableData.value = []
    parentMenuList.value = []
    // 加载该平台的菜单
    getMenuList()
    getParentMenuList()
  }

  /**
   * 新增平台
   */
  const handleAddPlatform = (): void => {
    editPlatformData.value = null
    platformDialogVisible.value = true
  }

  /**
   * 平台操作
   */
  const handlePlatformAction = async (
    action: string,
    platform: PlatformSysPlatform
  ): Promise<void> => {
    switch (action) {
      case 'edit':
        editPlatformData.value = { ...platform }
        platformDialogVisible.value = true
        break

      case 'enable':
        try {
          await enablePlatformApi(platform.id)
          ElMessage.success('启用成功')
          getPlatformList()
        } finally {
          // 操作完成
        }
        break

      case 'disable':
        try {
          await ElMessageBox.confirm(`确定要禁用平台「${platform.platformName}」吗？`, '禁用确认', {
            type: 'warning'
          })
          await disablePlatformApi(platform.id)
          ElMessage.success('禁用成功')
          getPlatformList()
        } catch (error: any) {
          // 用户取消操作
        }
        break

      case 'setDefault':
        try {
          await setDefaultPlatformApi(platform.id)
          ElMessage.success('设置成功')
          getPlatformList()
        } finally {
          // 操作完成
        }
        break

      case 'delete':
        try {
          await ElMessageBox.confirm(
            `确定要删除平台「${platform.platformName}」吗？删除后该平台下的所有菜单也将被删除！`,
            '删除确认',
            {
              confirmButtonText: '确定删除',
              cancelButtonText: '取消',
              type: 'warning'
            }
          )
          await deletePlatformApi(platform.id)
          ElMessage.success('删除成功')

          // 如果删除的是当前选中的平台，清空选择
          if (selectedPlatformId.value === platform.id) {
            selectedPlatformId.value = null
            tableData.value = []
          }

          getPlatformList()
        } catch (error: any) {
          // 用户取消操作
        }
        break
    }
  }

  /**
   * 提交平台表单
   */
  const handlePlatformSubmit = async (formData: any): Promise<void> => {
    if (formData.id && formData.id > 0) {
      await updatePlatformApi(formData as PlatformUpdateRequest)
      ElMessage.success('更新平台成功')
    } else {
      await addPlatformApi(formData as PlatformAddRequest)
      ElMessage.success('添加平台成功')
    }
    platformDialogVisible.value = false
    getPlatformList()
  }

  // ========== 菜单相关方法 ==========

  /**
   * 获取父级菜单列表
   */
  const getParentMenuList = async (): Promise<void> => {
    if (!selectedPlatformId.value) return

    try {
      const list = await getSysMenuSimpleTreeApi({
        platformId: selectedPlatformId.value,
        status: 1
      })
      parentMenuList.value = list || []
    } catch (error) {
      console.error('获取父级菜单失败:', error)
    }
  }

  /**
   * 获取菜单列表
   */
  const getMenuList = async (): Promise<void> => {
    if (!selectedPlatformId.value) return

    menuLoading.value = true
    try {
      const params: MenuSearchRequest = {
        platformId: selectedPlatformId.value
      }

      if (appliedFilters.name) {
        params.name = appliedFilters.name
      }
      if (appliedFilters.title) {
        params.title = appliedFilters.title
      }

      const response = await searchMenuTreeApi(params)
      tableData.value = response.items || []
    } catch (error) {
      console.error('获取菜单失败:', error)
    } finally {
      menuLoading.value = false
    }
  }

  /**
   * 获取菜单类型标签颜色
   */
  const getMenuTypeTag = (
    menuType: number
  ): 'primary' | 'success' | 'warning' | 'info' | 'danger' => {
    const typeMap = {
      1: 'info' as const,
      2: 'primary' as const,
      3: 'danger' as const
    }
    return typeMap[menuType as keyof typeof typeMap] || 'info'
  }

  /**
   * 获取菜单类型文本
   */
  const getMenuTypeText = (menuType: number): string => {
    const typeMap = {
      1: '目录',
      2: '菜单',
      3: '按钮'
    }
    return typeMap[menuType as keyof typeof typeMap] || '未知'
  }

  // 表格列配置
  const { columnChecks, columns } = useTableColumns(() => [
    {
      prop: 'title',
      label: '菜单名称',
      minWidth: 180,
      formatter: (row: MenuSysMenuTree) => formatMenuTitle(row.title)
    },
    {
      prop: 'menuType',
      label: '菜单类型',
      width: 100,
      formatter: (row: MenuSysMenuTree) => {
        return h(ElTag, { type: getMenuTypeTag(row.menuType) }, () => getMenuTypeText(row.menuType))
      }
    },
    {
      prop: 'name',
      label: '路由名称',
      width: 150,
      formatter: (row: MenuSysMenuTree) => row.name || '-'
    },
    {
      prop: 'path',
      label: '路由地址',
      minWidth: 150,
      formatter: (row: MenuSysMenuTree) => {
        if (row.menuType === 3) return '-'
        return row.link || row.path || '-'
      }
    },
    {
      prop: 'component',
      label: '组件路径',
      minWidth: 180,
      formatter: (row: MenuSysMenuTree) => row.component || '-'
    },
    {
      prop: 'label',
      label: '权限标识',
      width: 120,
      formatter: (row: MenuSysMenuTree) => row.label || '-'
    },
    {
      prop: 'sort',
      label: '排序',
      width: 80,
      align: 'center',
      formatter: (row: MenuSysMenuTree) => row.sort || 0
    },
    {
      prop: 'status',
      label: '状态',
      width: 80,
      align: 'center',
      formatter: (row: MenuSysMenuTree) => {
        return h(ElTag, { type: row.status === 1 ? 'success' : 'info', size: 'small' }, () =>
          row.status === 1 ? '启用' : '停用'
        )
      }
    },
    {
      prop: 'operation',
      label: '操作',
      width: 200,
      align: 'right',
      fixed: 'right',
      formatter: (row: MenuSysMenuTree) => {
        const buttonStyle = { style: 'text-align: right' }

        if (row.menuType === 3) {
          return h('div', buttonStyle, [
            h(ArtButtonTable, {
              type: 'edit',
              onClick: () => handleEditMenu(row)
            }),
            h(ArtButtonTable, {
              type: 'delete',
              onClick: () => handleDeleteMenu(row)
            })
          ])
        }

        return h('div', buttonStyle, [
          h(ArtButtonTable, {
            type: 'add',
            onClick: () => handleAddSubMenu(row),
            title: '新增子菜单'
          }),
          h(ArtButtonTable, {
            type: 'edit',
            onClick: () => handleEditMenu(row)
          }),
          h(ArtButtonTable, {
            type: 'delete',
            onClick: () => handleDeleteMenu(row)
          })
        ])
      }
    }
  ])

  const handleReset = (): void => {
    Object.assign(formFilters, { ...initialSearchState })
    Object.assign(appliedFilters, { ...initialSearchState })
    getMenuList()
  }

  const handleSearch = (): void => {
    Object.assign(appliedFilters, { ...formFilters })
    getMenuList()
  }

  const handleRefresh = (): void => {
    getMenuList()
    getParentMenuList()
  }

  const filteredTableData = computed(() => tableData.value)

  const handleAddMenu = (): void => {
    dialogType.value = 2
    editData.value = null
    menuDialogVisible.value = true
  }

  const handleAddSubMenu = (row: MenuSysMenuTree): void => {
    dialogType.value = 2
    editData.value = {
      ...row,
      parentId: row.id,
      id: 0
    } as any
    menuDialogVisible.value = true
  }

  const handleEditMenu = (row: MenuSysMenuTree): void => {
    dialogType.value = row.menuType as 1 | 2 | 3
    editData.value = { ...row }
    menuDialogVisible.value = true
  }

  const handleMenuSubmit = async (formData: any): Promise<void> => {
    if (!selectedPlatformId.value) {
      ElMessage.error('请先选择平台')
      return
    }

    try {
      menuLoading.value = true

      const submitData: any = {
        platformId: selectedPlatformId.value,
        parentId: formData.parentId || 0,
        menuType: formData.menuType,
        name: formData.name,
        path: formData.path || '',
        component: formData.component || '',
        redirect: formData.redirect || '',
        label: formData.label || '',
        title: formData.title,
        icon: formData.icon || '',
        sort: formData.sort || 0,
        link: formData.link || '',
        isEnable: formData.isEnable ? 1 : 0,
        isMenu: formData.isMenu ? 1 : 0,
        keepAlive: formData.keepAlive ? 1 : 0,
        isHide: formData.isHide ? 1 : 0,
        isIframe: formData.isIframe ? 1 : 0,
        isHideTab: formData.isHideTab ? 1 : 0,
        showBadge: formData.showBadge ? 1 : 0,
        showTextBadge: formData.showTextBadge || '',
        isFirstLevel: formData.isFirstLevel ? 1 : 0,
        fixedTab: formData.fixedTab ? 1 : 0,
        isFullPage: formData.isFullPage ? 1 : 0,
        activePath: formData.activePath || '',
        roles: formData.roles && formData.roles.length > 0 ? JSON.stringify(formData.roles) : '',
        status: formData.status ? 1 : 0
      }

      if (formData.menuType === 3) {
        submitData.authName = formData.authName || ''
        submitData.authLabel = formData.authLabel || ''
        submitData.authIcon = formData.authIcon || ''
        submitData.authSort = formData.authSort || 0
      }

      if (formData.id && formData.id > 0) {
        await updateMenuApi({ id: formData.id, ...submitData } as MenuUpdateRequest)
        ElMessage.success('更新菜单成功')
      } else {
        await addMenuApi(submitData as MenuAddRequest)
        ElMessage.success('添加菜单成功')
      }

      menuDialogVisible.value = false
      getMenuList()
      getParentMenuList()
    } catch (error: any) {
      console.error('提交失败:', error)
    } finally {
      menuLoading.value = false
    }
  }

  const handleDeleteMenu = async (row: MenuSysMenuTree): Promise<void> => {
    try {
      await ElMessageBox.confirm(
        `确定要删除菜单"${row.title}"吗？删除后无法恢复，且会删除所有子菜单！`,
        '删除确认',
        {
          confirmButtonText: '确定删除',
          cancelButtonText: '取消',
          type: 'warning',
          distinguishCancelAndClose: true
        }
      )

      menuLoading.value = true
      await deleteMenuApi(row.id)
      ElMessage.success('删除成功')
      getMenuList()
      getParentMenuList()
    } catch (error: any) {
      if (error !== 'cancel' && error !== 'close') {
        console.error('删除失败:', error)
      }
    } finally {
      menuLoading.value = false
    }
  }

  const toggleExpand = (): void => {
    isExpanded.value = !isExpanded.value
    nextTick(() => {
      if (tableRef.value?.elTableRef && filteredTableData.value) {
        const processRows = (rows: MenuSysMenuTree[]) => {
          rows.forEach((row) => {
            if (row.children?.length) {
              tableRef.value.elTableRef.toggleRowExpansion(row, isExpanded.value)
              processRows(row.children)
            }
          })
        }
        processRows(filteredTableData.value)
      }
    })
  }
</script>

<style lang="scss" scoped>
  .menu-page-container {
    height: 100%;
    display: flex;
    flex-direction: row;
  }

  .platform-panel {
    border: 1px solid var(--art-border-color);
    flex-shrink: 0;
  }

  .menu-panel {
    flex: 1;
    min-width: 0;
  }

  .platform-item {
    &:hover {
      background-color: var(--art-hover-bg);

      .platform-more-btn {
        opacity: 1;
      }
    }

    &.is-active {
      background-color: var(--el-color-primary-light-9);
      color: var(--el-color-primary);

      .platform-more-btn {
        color: var(--el-color-primary);
      }
    }

    &.is-disabled {
      opacity: 0.6;

      &:not(.is-active) {
        color: var(--art-text-secondary);
      }
    }

    .platform-more-btn {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 28px;
      height: 28px;
      padding: 0;
      border-radius: 6px;
      opacity: 0;
      transition: all 0.2s;
      color: var(--art-text-secondary);

      &:hover {
        background-color: var(--art-hover-bg);
        color: var(--el-color-primary);
      }
    }
  }

  .art-table-card {
    :deep(.el-card__body) {
      display: flex;
      flex-direction: column;
      height: 100%;
      overflow: hidden;
    }
  }

  // 空状态样式
  .empty-state-container {
    border: 1px solid var(--art-border-color);
  }

  .empty-state-content {
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;

    .empty-icon {
      color: var(--art-text-secondary);
      opacity: 0.4;
      margin-bottom: 16px;
    }

    .empty-title {
      font-size: 16px;
      font-weight: 500;
      color: var(--art-text-primary);
      margin: 0 0 8px 0;
    }

    .empty-desc {
      font-size: 13px;
      color: var(--art-text-secondary);
      margin: 0;
    }
  }
</style>
