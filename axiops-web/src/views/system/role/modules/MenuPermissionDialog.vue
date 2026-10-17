<template>
  <ElDialog
    v-model="visible"
    title="菜单权限配置"
    width="1000px"
    top="6vh"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @close="handleClose"
    class="menu-perm-dialog"
  >
    <div class="perm-header">
      <div class="perm-header__left">
        <span class="label">所属平台：</span>
        <ElSelect
          v-model="selectedPlatformId"
          placeholder="请选择平台"
          style="width: 240px"
          :loading="platformLoading"
          @change="handlePlatformChange"
        >
          <ElOption
            v-for="item in platformList"
            :key="item.id"
            :label="item.platformName"
            :value="item.id"
            :disabled="item.isEnable !== 1"
          >
            <span style="float: left">{{ item.platformName }}</span>
            <ElTag
              v-if="item.isDefault === 1"
              size="small"
              type="info"
              effect="plain"
              style="float: right; margin-top: 6px"
              >默认</ElTag
            >
          </ElOption>
        </ElSelect>
      </div>

      <div class="perm-header__right">
        <ElInput
          v-model="searchText"
          placeholder="筛选菜单..."
          :prefix-icon="Search"
          clearable
          style="width: 220px"
        />
      </div>
    </div>

    <div class="perm-container" v-loading="loading">
      <div v-if="!selectedPlatformId" class="perm-empty">
        <div class="empty-content">
          <ElIcon :size="40" color="#dcdfe6"><icon-menu /></ElIcon>
          <p>请先选择上方平台进行配置</p>
        </div>
      </div>

      <template v-else>
        <div class="perm-sider">
          <div class="perm-sider__header">
            <span class="title">一级导航</span>
            <ElCheckbox
              v-model="allSelected"
              :indeterminate="isIndeterminateAll"
              @change="handleSelectAll"
              label="全选"
              size="small"
            />
          </div>
          <ElScrollbar wrap-class="perm-sider__scroll">
            <div class="perm-nav-list">
              <div
                v-for="group in filteredGroups"
                :key="group.id"
                class="perm-nav-item"
                :class="{ 'is-active': activeGroupId === group.id }"
                @click="activeGroupId = group.id"
              >
                <div class="perm-nav-item__inner">
                  <ElCheckbox
                    v-model="group.checked"
                    :indeterminate="group.indeterminate"
                    @click.stop
                    @change="handleGroupCheck(group)"
                  />
                  <span class="title" :title="group.title">{{ group.title }}</span>
                  <span class="count" v-if="group.checkedCount > 0">
                    {{ group.checkedCount }}
                  </span>
                </div>
              </div>
            </div>
            <div v-if="filteredGroups.length === 0" class="no-data-text"> 暂无匹配菜单 </div>
          </ElScrollbar>
        </div>

        <div class="perm-content">
          <div class="perm-content__header" v-if="currentActiveGroup">
            <div class="title-area">
              <span class="parent-name">{{ currentActiveGroup.title }}</span>
              <span class="divider">/</span>
              <span class="sub-text">权限明细</span>
            </div>
            <div class="stat-area">
              <ElTag type="info" size="small" effect="plain">
                已选 {{ currentActiveGroup.checkedCount }} /
                {{ currentActiveGroup.totalCount - 1 }} 项
              </ElTag>
            </div>
          </div>

          <div class="perm-content__body">
            <ElScrollbar>
              <div class="grid-wrapper" v-if="currentActiveGroup">
                <div class="perm-grid">
                  <div
                    v-for="item in currentActiveGroup.items"
                    :key="item.id"
                    class="perm-card"
                    :class="{ 'is-checked': item.checked }"
                    @click="toggleItemCheck(item, currentActiveGroup)"
                  >
                    <div class="card-checkbox">
                      <ElCheckbox
                        v-model="item.checked"
                        @click.stop
                        @change="handleItemCheck(item, currentActiveGroup)"
                      />
                    </div>
                    <div class="card-info">
                      <div class="card-title">{{ item.title }}</div>
                      <div class="card-tags" v-if="getMenuTypeText(item.menuType)">
                        <ElTag size="small" :type="getMenuTypeTag(item.menuType)" effect="plain">
                          {{ getMenuTypeText(item.menuType) }}
                        </ElTag>
                      </div>
                    </div>
                  </div>
                </div>

                <div v-if="currentActiveGroup.items.length === 0" class="empty-sub">
                  该目录下没有配置子菜单或按钮权限
                </div>
              </div>
            </ElScrollbar>
          </div>
        </div>
      </template>
    </div>

    <template #footer>
      <div class="perm-footer">
        <div class="status-bar">
          <template v-if="roleData">
            当前配置角色：<span class="highlight">{{ roleData.name }}</span>
          </template>
          <span class="separator" v-if="roleData">|</span>
          <span
            >总计已选：<span class="count">{{ selectedCount }}</span> 项权限</span
          >
        </div>
        <div class="actions">
          <ElButton @click="handleCancel">取消</ElButton>
          <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">
            保存配置
          </ElButton>
        </div>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import { Search, Menu as IconMenu } from '@element-plus/icons-vue'
  import {
    getSysMenuSimpleTreeApi,
    bindRoleMenuApi,
    getRoleMenuApi,
    type MenuSysMenuSimpleTreeNode,
    type RoleSysRole
  } from '@/api/portal'
  import { searchPlatformApi, type PlatformSysPlatform } from '@/api/portal/platform'

  // --- 类型定义 ---
  interface ExtendedMenuItem extends MenuSysMenuSimpleTreeNode {
    checked?: boolean
    menuType?: number | string // 兼容后端可能返回字符串
  }

  interface PermissionGroup {
    id: number
    title: string
    checked: boolean
    indeterminate: boolean
    items: ExtendedMenuItem[]
    checkedCount: number
    totalCount: number
  }

  interface Props {
    modelValue: boolean
    roleData?: RoleSysRole
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['update:modelValue', 'success'])

  // --- 状态变量 ---
  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const platformLoading = ref(false)
  const loading = ref(false)
  const submitLoading = ref(false)
  const platformList = ref<PlatformSysPlatform[]>([])
  const selectedPlatformId = ref<number | null>(null)
  const searchText = ref('')
  const activeGroupId = ref<number | null>(null)
  const groups = ref<PermissionGroup[]>([])
  const allNodes = ref<Map<number, ExtendedMenuItem>>(new Map())
  const checkedIds = ref<Set<number>>(new Set())

  // --- 计算属性 ---
  const filteredGroups = computed(() => {
    if (!searchText.value) return groups.value
    const keyword = searchText.value.toLowerCase()
    return groups.value.filter((group) => {
      const groupMatch = group.title.toLowerCase().includes(keyword)
      const childMatch = group.items.some((item) => item.title?.toLowerCase().includes(keyword))
      return groupMatch || childMatch
    })
  })

  const currentActiveGroup = computed(() => {
    if (!activeGroupId.value) return null
    return groups.value.find((g) => g.id === activeGroupId.value) || null
  })

  const selectedCount = computed(() => checkedIds.value.size)

  const allSelected = computed({
    get: () => allNodes.value.size > 0 && checkedIds.value.size === allNodes.value.size,
    set: () => {}
  })

  const isIndeterminateAll = computed(() => {
    return checkedIds.value.size > 0 && checkedIds.value.size < allNodes.value.size
  })

  // --- 监听与初始化 ---
  watch(visible, async (val) => {
    if (val && props.roleData) {
      await initPlatformList()
    }
  })

  watch(searchText, () => {
    if (filteredGroups.value.length > 0) {
      const exists = filteredGroups.value.find((g) => g.id === activeGroupId.value)
      if (!exists) {
        activeGroupId.value = filteredGroups.value[0].id
      }
    }
  })

  const initPlatformList = async () => {
    platformLoading.value = true
    try {
      const res = await searchPlatformApi({ page: 1, pageSize: 100 })
      platformList.value = res.items || []

      if (!selectedPlatformId.value && platformList.value.length > 0) {
        const def =
          platformList.value.find((p) => p.isDefault === 1 && p.isEnable === 1) ||
          platformList.value[0]
        if (def && def.isEnable === 1) {
          selectedPlatformId.value = def.id
          await loadMenuData()
        }
      }
    } finally {
      platformLoading.value = false
    }
  }

  const handlePlatformChange = () => {
    loadMenuData()
  }

  // --- 核心逻辑 ---
  const loadMenuData = async () => {
    if (!selectedPlatformId.value) return

    loading.value = true
    checkedIds.value.clear()
    allNodes.value.clear()
    activeGroupId.value = null

    try {
      const [treeRes, roleIdsRes] = await Promise.all([
        getSysMenuSimpleTreeApi({ platformId: selectedPlatformId.value, status: 1 }),
        getRoleMenuApi(props.roleData!.id)
      ])

      processData(treeRes || [])

      const serverIds = roleIdsRes || []
      serverIds.forEach((id) => {
        if (allNodes.value.has(id)) checkedIds.value.add(id)
      })

      syncState()

      if (groups.value.length > 0) {
        activeGroupId.value = groups.value[0].id
      }
    } finally {
      loading.value = false
    }
  }

  const processData = (tree: MenuSysMenuSimpleTreeNode[]) => {
    const tempGroups: PermissionGroup[] = []

    tree.forEach((root) => {
      const rootExt = { ...root, checked: false }
      allNodes.value.set(root.id, rootExt)

      const children: ExtendedMenuItem[] = []
      if (root.children) {
        flatten(root.children, children)
      }

      tempGroups.push({
        id: root.id,
        title: root.title,
        checked: false,
        indeterminate: false,
        items: children,
        checkedCount: 0,
        totalCount: children.length + 1
      })
    })

    groups.value = tempGroups
  }

  const flatten = (nodes: MenuSysMenuSimpleTreeNode[], res: ExtendedMenuItem[]) => {
    nodes.forEach((node) => {
      const ext = { ...node, checked: false }
      allNodes.value.set(node.id, ext)
      res.push(ext)
      if (node.children) flatten(node.children, res)
    })
  }

  const syncState = () => {
    allNodes.value.forEach((node) => {
      node.checked = checkedIds.value.has(node.id)
    })

    groups.value.forEach((group) => {
      let count = 0
      if (checkedIds.value.has(group.id)) count++
      group.items.forEach((child) => {
        if (checkedIds.value.has(child.id)) count++
      })

      group.checkedCount = count
      group.checked = count === group.totalCount
      group.indeterminate = count > 0 && count < group.totalCount
    })
  }

  // --- 事件处理 ---
  const handleSelectAll = (val: boolean) => {
    if (val) {
      allNodes.value.forEach((n) => checkedIds.value.add(n.id))
    } else {
      checkedIds.value.clear()
    }
    syncState()
  }

  const handleGroupCheck = (group: PermissionGroup) => {
    const target = group.checked
    if (target) checkedIds.value.add(group.id)
    else checkedIds.value.delete(group.id)

    group.items.forEach((item) => {
      if (target) checkedIds.value.add(item.id)
      else checkedIds.value.delete(item.id)
    })
    syncState()
  }

  const toggleItemCheck = (item: ExtendedMenuItem, group: PermissionGroup) => {
    item.checked = !item.checked
    handleItemCheck(item, group)
  }

  const handleItemCheck = (item: ExtendedMenuItem, group: PermissionGroup) => {
    if (item.checked) {
      checkedIds.value.add(item.id)
      if (!checkedIds.value.has(group.id)) {
        checkedIds.value.add(group.id)
      }
    } else {
      checkedIds.value.delete(item.id)
    }
    syncState()
  }

  // --- 辅助方法 (修复未知标签问题) ---
  const getMenuTypeTag = (type?: number | string) => {
    const t = Number(type) // 强制转数字
    switch (t) {
      case 1:
        return ''
      case 2:
        return 'primary'
      case 3:
        return 'info'
      default:
        return ''
    }
  }

  const getMenuTypeText = (type?: number | string) => {
    const t = Number(type) // 强制转数字
    switch (t) {
      case 1:
        return '目录'
      case 2:
        return '菜单'
      case 3:
        return '按钮'
      default:
        return null // 返回null以在模板中通过v-if隐藏
    }
  }

  const handleCancel = () => {
    visible.value = false
  }

  const handleClose = () => {
    searchText.value = ''
  }

  const handleSubmit = async () => {
    if (!selectedPlatformId.value) return ElMessage.warning('未选择平台')
    submitLoading.value = true
    try {
      await bindRoleMenuApi({
        roleId: props.roleData!.id,
        menuIds: Array.from(checkedIds.value)
      })
      ElMessage.success('保存成功')
      emit('success')
      visible.value = false
    } catch (e) {
      console.error(e)
    } finally {
      submitLoading.value = false
    }
  }
</script>

<style lang="scss" scoped>
  /* 变量定义 */
  $border-color: #e4e7ed;
  $bg-active: #ecf5ff;
  $bg-hover: #f5f7fa;
  $text-primary: #303133;
  $text-regular: #606266;
  $text-secondary: #909399;
  $primary-color: #409eff;

  /* 对话框全局覆写 */
  .menu-perm-dialog {
    :deep(.el-dialog__body) {
      padding: 0;
    }
  }

  /* 顶部操作栏 */
  .perm-header {
    padding: 0 20px 16px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid $border-color;

    &__left {
      display: flex;
      align-items: center;
      .label {
        font-size: 14px;
        color: $text-regular;
        margin-right: 8px;
      }
    }
  }

  /* 主容器 */
  .perm-container {
    display: flex;
    height: 520px;
    border-bottom: 1px solid $border-color;
    overflow: hidden;
    align-items: stretch; /* 强制子元素高度铺满 */
  }

  /* 左侧 Sider */
  .perm-sider {
    width: 240px;
    border-right: 1px solid $border-color;
    display: flex;
    flex-direction: column;
    background-color: #fff;
    flex-shrink: 0;

    &__header {
      padding: 12px 16px;
      border-bottom: 1px solid $border-color;
      background-color: $bg-hover;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-shrink: 0;
      .title {
        font-weight: 600;
        color: $text-primary;
        font-size: 14px;
      }
    }
  }

  .perm-nav-list {
    padding: 8px 0;
  }

  .perm-nav-item {
    cursor: pointer;
    transition: all 0.2s;
    position: relative;

    &:hover {
      background-color: $bg-hover;
    }

    &.is-active {
      background-color: $bg-active;
      &::after {
        content: '';
        position: absolute;
        right: 0;
        top: 0;
        bottom: 0;
        width: 3px;
        background-color: $primary-color;
      }
      .title {
        color: $primary-color;
        font-weight: 500;
      }
    }

    &__inner {
      padding: 10px 16px;
      display: flex;
      align-items: center;
      gap: 8px;

      .title {
        flex: 1;
        font-size: 14px;
        color: $text-regular;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
      }
      .count {
        font-size: 12px;
        color: $primary-color;
        background: rgba($primary-color, 0.1);
        padding: 2px 6px;
        border-radius: 10px;
      }
    }
  }

  /* 右侧 Content - 修复核心 */
  .perm-content {
    flex: 1;
    display: flex;
    flex-direction: column; /* 垂直排列 */
    background-color: #fff;
    overflow: hidden;
    justify-content: flex-start !important; /* 强制内容靠上 */

    &__header {
      padding: 12px 20px;
      border-bottom: 1px solid $border-color;
      display: flex;
      justify-content: space-between;
      align-items: center;
      background-color: #fff;
      flex-shrink: 0; /* 防止被压缩 */

      .title-area {
        font-size: 14px;
        display: flex;
        align-items: center;
        .parent-name {
          font-weight: 600;
          color: $text-primary;
        }
        .divider {
          margin: 0 6px;
          color: $text-secondary;
        }
        .sub-text {
          color: $text-secondary;
        }
      }
    }

    &__body {
      flex: 1;
      overflow: hidden;
      /* 这里不加 padding，由 grid-wrapper 加 */
      position: relative;
    }
  }

  .grid-wrapper {
    padding: 20px;
  }

  .perm-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
    gap: 12px;
  }

  .perm-card {
    border: 1px solid $border-color;
    border-radius: 4px;
    padding: 12px;
    display: flex;
    align-items: flex-start;
    gap: 10px;
    cursor: pointer;
    transition: all 0.2s;
    background-color: #fff;
    /* 确保高度自适应，不要被 stretch */
    height: auto;
    align-self: start;

    &:hover {
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.05);
      border-color: #c0c4cc;
    }
    &.is-checked {
      background-color: $bg-active;
      border-color: #d9ecff;
      .card-title {
        color: $primary-color;
      }
    }

    .card-info {
      flex: 1;
      overflow: hidden;
      .card-title {
        font-size: 14px;
        color: $text-primary;
        margin-bottom: 6px;
        line-height: 1.4;
      }
      .card-tags {
        display: flex;
        gap: 4px;
      }
    }
  }

  /* 空状态 */
  .perm-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    background-color: #fff;
    .empty-content {
      text-align: center;
      color: $text-secondary;
      p {
        margin-top: 12px;
        font-size: 14px;
      }
    }
  }

  .no-data-text,
  .empty-sub {
    text-align: center;
    padding: 20px;
    color: $text-secondary;
    font-size: 13px;
  }
  .empty-sub {
    padding-top: 40px;
  }

  /* 底部 */
  .perm-footer {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 16px 20px;
    border-top: 1px solid $border-color;

    .status-bar {
      font-size: 13px;
      color: $text-regular;
      .highlight {
        color: $text-primary;
        font-weight: 600;
      }
      .separator {
        margin: 0 10px;
        color: $border-color;
      }
      .count {
        color: $primary-color;
        font-weight: bold;
        font-size: 15px;
        margin: 0 2px;
      }
    }
    .actions {
      display: flex;
      gap: 12px;
    }
  }
</style>
