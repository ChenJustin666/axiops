<template>
  <ElDialog
    v-model="visible"
    title="API 权限配置"
    width="1100px"
    top="6vh"
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @close="handleClose"
    class="api-perm-dialog"
  >
    <div class="perm-header">
      <div class="perm-header__left">
        <ElTag type="info" effect="plain" class="tips-tag">
          <ElIcon class="mr-1"><InfoFilled /></ElIcon>
          左侧目录支持级联勾选，右侧管理接口明细
        </ElTag>
      </div>
      <div class="perm-header__right">
        <ElInput
          v-model="filterText"
          placeholder="输入关键字过滤..."
          :prefix-icon="Search"
          clearable
          style="width: 260px"
        />
      </div>
    </div>

    <div class="perm-container" v-loading="loading">
      <div class="perm-sider">
        <div class="perm-sider__header">
          <span>接口目录</span>
          <div class="tree-actions">
            <el-button link type="primary" size="small" @click="expandAllNodes">
              {{ isExpandAll ? '折叠' : '展开' }}
            </el-button>
            <el-button link type="primary" size="small" @click="checkAllNodes"> 全选 </el-button>
          </div>
        </div>

        <ElScrollbar wrap-class="perm-sider__scroll">
          <ElTree
            ref="treeRef"
            :data="treeData"
            :props="defaultProps"
            node-key="id"
            show-checkbox
            highlight-current
            :filter-node-method="filterNode"
            :expand-on-click-node="false"
            :default-expanded-keys="expandedKeys"
            @node-click="handleNodeClick"
            @check="handleTreeCheck"
            class="custom-tree"
          >
            <template #default="{ node, data }">
              <span class="custom-tree-node">
                <span class="icon">{{ data.isPermission === 0 ? '📁' : '⚡' }}</span>
                <span class="label" :title="node.label">{{ node.label }}</span>

                <span
                  v-if="data.isPermission === 0"
                  class="node-stat"
                  :class="{
                    'is-full': data._selectedCount === data._totalCount && data._totalCount > 0
                  }"
                >
                  [{{ data._selectedCount || 0 }}/{{ data._totalCount || 0 }}]
                </span>

                <span
                  v-if="data.isPermission === 1"
                  class="mini-method"
                  :class="getMethodClass(data.Method)"
                >
                  {{ formatMethod(data.Method) }}
                </span>
              </span>
            </template>
          </ElTree>
        </ElScrollbar>
      </div>

      <div class="perm-content">
        <div class="perm-content__header" v-if="currentNode">
          <div class="title-area">
            <span class="current-node">{{ currentNode.name }}</span>
            <span class="sub-text" v-if="currentNode.isPermission === 0">包含的接口列表</span>
          </div>
          <div class="stat-area">
            <ElTag v-if="rightList.length > 0" size="small" effect="plain">
              当前列表已选 {{ currentViewCheckedCount }} / {{ rightList.length }}
            </ElTag>
          </div>
        </div>

        <div class="perm-content__body">
          <ElScrollbar style="height: 100%">
            <div class="api-list-wrapper" v-if="rightList.length > 0">
              <div
                v-for="api in rightList"
                :key="api.id"
                class="api-row"
                :class="{ 'is-checked': isApiChecked(api.id) }"
                @click="toggleApiCheck(api)"
              >
                <div class="row-checkbox">
                  <ElCheckbox
                    :model-value="isApiChecked(api.id)"
                    @change="toggleApiCheck(api)"
                    @click.stop
                  />
                </div>

                <div class="row-method">
                  <span class="method-badge" :class="getMethodClass(api.Method)">
                    {{ formatMethod(api.Method) }}
                  </span>
                </div>

                <div class="row-info">
                  <div class="row-path" v-if="api._parentName">
                    {{ api._parentName }}
                  </div>
                  <div class="row-name" :title="api.path">{{ api.name }}</div>
                  <div class="row-url">{{ api.path }}</div>
                </div>

                <div class="row-status">
                  <ElIcon v-if="isApiChecked(api.id)" color="#67c23a"><Check /></ElIcon>
                </div>
              </div>
            </div>

            <div v-else-if="currentNode" class="empty-state">
              <p>该目录下暂无 API 数据</p>
            </div>

            <div v-else class="empty-state placeholder">
              <ElIcon :size="48" color="#dcdfe6"><Pointer /></ElIcon>
              <p>请点击左侧目录查看详情</p>
            </div>
          </ElScrollbar>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="perm-footer">
        <div class="stats-bar">
          <div class="main-stat">
            已选总数：<span class="count-total">{{ totalSelectedCount }}</span>
          </div>
          <div class="sub-stats">
            <span class="stat-item"><i class="dot all"></i> *: {{ methodStats['*'] || 0 }}</span>
            <span class="stat-item"><i class="dot get"></i> GET: {{ methodStats.GET || 0 }}</span>
            <span class="stat-item"
              ><i class="dot post"></i> POST: {{ methodStats.POST || 0 }}</span
            >
            <span class="stat-item"><i class="dot put"></i> PUT: {{ methodStats.PUT || 0 }}</span>
            <span class="stat-item"
              ><i class="dot delete"></i> DEL: {{ methodStats.DELETE || 0 }}</span
            >
            <span class="stat-item" v-if="methodStats.OTHER > 0"
              >其他: {{ methodStats.OTHER }}</span
            >
          </div>
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
  import { ref, computed, watch, nextTick } from 'vue'
  import { ElMessage, ElTree } from 'element-plus'
  import { Search, InfoFilled, Pointer, Check } from '@element-plus/icons-vue'
  import {
    getApiTreeApi,
    bindRoleApiApi,
    getRoleApiApi,
    type ApiSysAPITreeNode,
    type RoleSysRole
  } from '@/api/portal'

  // --- 类型扩展 ---
  interface ExtendedApiNode extends ApiSysAPITreeNode {
    _parentName?: string // 父级路径名称
    _totalCount?: number // 该节点下所有API总数
    _selectedCount?: number // 该节点下已选API数
    children: ExtendedApiNode[]
    Method?: string // 兼容后端大小写
    method?: string
  }

  interface Props {
    modelValue: boolean
    roleData?: RoleSysRole
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['update:modelValue', 'success'])

  // --- 状态 ---
  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const loading = ref(false)
  const submitLoading = ref(false)
  const filterText = ref('')
  const treeData = ref<ExtendedApiNode[]>([])
  const treeRef = ref<InstanceType<typeof ElTree>>()

  // 核心选中状态
  const checkedKeys = ref<number[]>([]) // 仅包含选中的ID
  const isExpandAll = ref(false)
  const expandedKeys = ref<number[]>([])

  // 右侧视图
  const currentNode = ref<ExtendedApiNode | null>(null)
  const rightList = ref<ExtendedApiNode[]>([]) // 右侧扁平化列表

  // 配置
  const defaultProps = {
    children: 'children',
    label: 'name'
  }

  // --- 监听 ---
  watch(visible, async (val) => {
    if (val && props.roleData) {
      await initData()
    }
  })

  watch(filterText, (val) => {
    treeRef.value?.filter(val)
  })

  // --- 初始化逻辑 ---
  const initData = async () => {
    loading.value = true
    currentNode.value = null
    rightList.value = []
    checkedKeys.value = []

    try {
      // 1. 并发请求：获取树 + 获取角色权限
      const [treeRes, roleIdsRes] = await Promise.all([
        getApiTreeApi(),
        getRoleApiApi(props.roleData!.id)
      ])

      // 2. 处理树数据 (计算总数，初始化统计)
      treeData.value = (treeRes || []) as ExtendedApiNode[]
      initTreeStats(treeData.value)

      // 3. 处理回显 (关键修复：只设置叶子节点)
      const serverIds = roleIdsRes || []
      await nextTick()

      if (treeRef.value) {
        // 过滤出所有的叶子节点ID (isPermission === 1)
        // 避免传入父节点ID导致下面的子节点全部被自动勾选
        const leafIds = filterLeafIds(treeData.value, serverIds)
        treeRef.value.setCheckedKeys(leafIds)

        // 手动触发一次更新，计算统计数据
        handleTreeCheck()
      }
    } finally {
      loading.value = false
    }
  }

  // 递归初始化节点的统计字段
  const initTreeStats = (nodes: ExtendedApiNode[]) => {
    nodes.forEach((node) => {
      node._totalCount = countTotalApis(node)
      node._selectedCount = 0
      if (node.children) {
        initTreeStats(node.children)
      }
    })
  }

  // 计算某节点下所有 API (isPermission=1) 的数量
  const countTotalApis = (node: ExtendedApiNode): number => {
    if (node.isPermission === 1) return 1
    if (!node.children || node.children.length === 0) return 0
    return node.children.reduce((sum, child) => sum + countTotalApis(child), 0)
  }

  // 过滤出叶子节点ID
  const filterLeafIds = (nodes: ExtendedApiNode[], ids: number[]): number[] => {
    const idSet = new Set(ids)
    const result: number[] = []

    const traverse = (list: ExtendedApiNode[]) => {
      list.forEach((node) => {
        // 如果是 API 节点且在集合中
        if (node.isPermission === 1 && idSet.has(node.id)) {
          result.push(node.id)
        }
        if (node.children) traverse(node.children)
      })
    }
    traverse(nodes)
    return result
  }

  // --- 交互逻辑 ---

  // 树节点点击 -> 生成右侧列表
  const handleNodeClick = (data: ExtendedApiNode) => {
    currentNode.value = data
    const apis: ExtendedApiNode[] = []

    // 递归收集该节点下的所有 API
    const collect = (node: ExtendedApiNode, parentName: string) => {
      if (node.isPermission === 1) {
        // 兼容 method/Method 字段
        const m = node.Method || node.method || ''
        apis.push({ ...node, Method: m, _parentName: parentName })
      }
      if (node.children) {
        // 构建面包屑路径
        const path =
          node.isPermission === 0
            ? parentName
              ? `${parentName} / ${node.name}`
              : node.name
            : parentName

        node.children.forEach((child) => collect(child, path))
      }
    }

    if (data.isPermission === 1) {
      // 点的是具体 API，显示它所在的组
      apis.push({ ...data, Method: data.Method || data.method })
    } else {
      // 点的是目录
      collect(data, '')
    }

    rightList.value = apis
  }

  // 树复选框变化 -> 更新统计
  const handleTreeCheck = () => {
    if (!treeRef.value) return

    // 获取当前所有选中的 Key (包括叶子和半选，用于提交，但统计我们只看叶子)
    // 获取所有选中的 API 节点
    const checkedNodes = treeRef.value.getCheckedNodes(false, false) as ExtendedApiNode[]
    const checkedApiIds = checkedNodes.filter((n) => n.isPermission === 1).map((n) => n.id)

    checkedKeys.value = checkedApiIds // 更新响应式变量

    // 递归更新树上每个节点的 _selectedCount
    updateNodeSelectedCounts(treeData.value, new Set(checkedApiIds))
  }

  // 递归更新已选数量 (核心统计逻辑)
  const updateNodeSelectedCounts = (nodes: ExtendedApiNode[], idSet: Set<number>): number => {
    let sum = 0
    nodes.forEach((node) => {
      if (node.isPermission === 1) {
        if (idSet.has(node.id)) sum += 1
      } else {
        // 是目录，递归计算子节点
        const childSum = updateNodeSelectedCounts(node.children || [], idSet)
        node._selectedCount = childSum // 赋值给节点，Vue 会响应式更新 UI
        sum += childSum
      }
    })
    return sum
  }

  // 右侧列表点击 -> 切换选中
  const toggleApiCheck = (api: ExtendedApiNode) => {
    if (!treeRef.value) return
    const isChecked = checkedKeys.value.includes(api.id)
    // 切换状态，deep=true 确保状态一致性
    treeRef.value.setChecked(api.id, !isChecked, true)
    handleTreeCheck() // 触发统计更新
  }

  // 树搜索
  const filterNode = (value: string, data: ExtendedApiNode) => {
    if (!value) return true
    const name = data.name.toLowerCase()
    const path = (data as any).path?.toLowerCase() || ''
    const k = value.toLowerCase()
    return name.includes(k) || path.includes(k)
  }

  // --- 统计计算属性 ---

  // 当前右侧视图的选中数
  const currentViewCheckedCount = computed(() => {
    return rightList.value.filter((item) => checkedKeys.value.includes(item.id)).length
  })

  // 总选中数
  const totalSelectedCount = computed(() => checkedKeys.value.length)

  // 分类统计
  const methodStats = computed(() => {
    const stats: Record<string, number> = {
      '*': 0,
      GET: 0,
      POST: 0,
      PUT: 0,
      DELETE: 0,
      OTHER: 0
    }

    // 需要遍历整个树或者利用 checkedKeys 查找节点
    // 为了性能，我们直接利用 treeRef 的 getCheckedNodes
    if (!treeRef.value) return stats

    const nodes = treeRef.value.getCheckedNodes(false, false) as ExtendedApiNode[]

    nodes.forEach((node) => {
      if (node.isPermission === 1) {
        const m = (node.Method || node.method || '').toUpperCase()
        if (m === '*') stats['*']++
        else if (['GET', 'POST', 'PUT', 'DELETE'].includes(m)) stats[m]++
        else stats.OTHER++
      }
    })

    return stats
  })

  // --- 样式与辅助 ---

  const isApiChecked = (id: number) => checkedKeys.value.includes(id)

  const formatMethod = (m?: string) => (m ? m.toUpperCase() : 'N/A')

  const getMethodClass = (method?: string) => {
    const m = (method || '').toUpperCase()
    switch (m) {
      case 'GET':
        return 'badge-get'
      case 'POST':
        return 'badge-post'
      case 'PUT':
        return 'badge-put'
      case 'DELETE':
        return 'badge-delete'
      case '*':
        return 'badge-all' // 通配符样式
      case 'PATCH':
        return 'badge-patch'
      default:
        return 'badge-gray'
    }
  }

  // 展开/收起
  const expandAllNodes = () => {
    isExpandAll.value = !isExpandAll.value
    const nodesMap = treeRef.value!.store.nodesMap
    for (let key in nodesMap) {
      nodesMap[key].expanded = isExpandAll.value
    }
  }

  const checkAllNodes = () => {
    if (!treeRef.value) return
    treeRef.value.setCheckedNodes(treeData.value as any[])
    handleTreeCheck()
  }

  const handleClose = () => {
    filterText.value = ''
  }

  const handleCancel = () => {
    visible.value = false
  }

  const handleSubmit = async () => {
    submitLoading.value = true
    try {
      // 提交所有选中的ID（包括API和目录的半选状态，视后端需求而定）
      // 通常权限系统只需要叶子节点ID，或者包含半选的完整路径
      // 这里示例：提交所有选中的 API ID
      const apiIds = checkedKeys.value

      await bindRoleApiApi({
        roleId: props.roleData!.id,
        apiIds: apiIds
      })

      ElMessage.success('权限保存成功')
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
  /* 颜色变量 */
  $border-color: #e4e7ed;
  $bg-hover: #f5f7fa;
  $primary: #409eff;
  $text-main: #303133;
  $text-sub: #606266;

  .api-perm-dialog {
    :deep(.el-dialog__body) {
      padding: 0;
    }
  }

  .perm-header {
    padding: 12px 20px;
    border-bottom: 1px solid $border-color;
    display: flex;
    justify-content: space-between;
    align-items: center;
    .tips-tag {
      font-size: 12px;
    }
  }

  .perm-container {
    display: flex;
    height: 580px;
    border-bottom: 1px solid $border-color;
    align-items: stretch;
  }

  /* 左侧 Sider */
  .perm-sider {
    width: 340px;
    border-right: 1px solid $border-color;
    display: flex;
    flex-direction: column;
    background-color: #fff;
    flex-shrink: 0;

    &__header {
      padding: 10px 16px;
      background: $bg-hover;
      border-bottom: 1px solid $border-color;
      display: flex;
      justify-content: space-between;
      align-items: center;
      font-weight: 600;
      color: $text-main;
    }
    &__scroll {
      flex: 1;
      overflow: hidden;
    }
  }

  /* 自定义树节点 */
  .custom-tree {
    padding: 10px 0;
    :deep(.el-tree-node__content) {
      height: 34px;
    }
  }

  .custom-tree-node {
    flex: 1;
    display: flex;
    align-items: center;
    overflow: hidden;
    font-size: 13px;
    padding-right: 8px;

    .icon {
      margin-right: 6px;
      font-size: 14px;
    }
    .label {
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      margin-right: 8px;
    }

    .node-stat {
      font-size: 12px;
      color: #909399;
      font-family: var(--ax-font-mono);
      &.is-full {
        color: $primary;
        font-weight: bold;
      }
    }

    .mini-method {
      font-size: 10px;
      padding: 1px 4px;
      border-radius: 3px;
      transform: scale(0.9);
      font-family: var(--ax-font-mono);
      // 复用下面的样式
      &.badge-get {
        color: #409eff;
        background: #ecf5ff;
      }
      &.badge-post {
        color: #67c23a;
        background: #f0f9eb;
      }
      &.badge-put {
        color: #e6a23c;
        background: #fdf6ec;
      }
      &.badge-delete {
        color: #f56c6c;
        background: #fef0f0;
      }
      &.badge-all {
        color: #303133;
        background: #f4f4f5;
        font-weight: bold;
      }
    }
  }

  /* 右侧 Content */
  .perm-content {
    flex: 1;
    display: flex;
    flex-direction: column;
    background-color: #fff;
    overflow: hidden;
    justify-content: flex-start !important;

    &__header {
      padding: 12px 20px;
      border-bottom: 1px solid $border-color;
      display: flex;
      justify-content: space-between;
      align-items: center;
      flex-shrink: 0;

      .current-node {
        font-size: 15px;
        font-weight: 600;
        color: $text-main;
        margin-right: 10px;
      }
      .sub-text {
        font-size: 12px;
        color: $text-sub;
      }
    }

    &__body {
      flex: 1;
      overflow: hidden;
      position: relative;
    }
  }

  .api-list-wrapper {
    padding: 0;
  }

  .api-row {
    display: flex;
    align-items: center;
    padding: 10px 20px;
    border-bottom: 1px solid #f5f7fa;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      background-color: $bg-hover;
    }
    &.is-checked {
      background-color: #f0f9eb;
    }

    .row-checkbox {
      margin-right: 12px;
    }

    .row-method {
      width: 70px;
      flex-shrink: 0;
      .method-badge {
        display: inline-block;
        width: 54px;
        text-align: center;
        font-size: 11px;
        font-weight: bold;
        padding: 2px 0;
        border-radius: 4px;
        font-family: var(--ax-font-mono);

        &.badge-get {
          color: #409eff;
          background: rgba(64, 158, 255, 0.1);
          border: 1px solid rgba(64, 158, 255, 0.2);
        }
        &.badge-post {
          color: #67c23a;
          background: rgba(103, 194, 58, 0.1);
          border: 1px solid rgba(103, 194, 58, 0.2);
        }
        &.badge-put {
          color: #e6a23c;
          background: rgba(230, 162, 60, 0.1);
          border: 1px solid rgba(230, 162, 60, 0.2);
        }
        &.badge-delete {
          color: #f56c6c;
          background: rgba(245, 108, 108, 0.1);
          border: 1px solid rgba(245, 108, 108, 0.2);
        }
        &.badge-all {
          color: #303133;
          background: #e9e9eb;
          border: 1px solid #dcdfe6;
        } // * 号样式
        &.badge-patch {
          color: #722ed1;
          background: rgba(114, 46, 209, 0.1);
        }
        &.badge-gray {
          color: #909399;
          background: #f4f4f5;
        }
      }
    }

    .row-info {
      flex: 1;
      min-width: 0;
      margin-left: 10px;

      .row-path {
        font-size: 12px;
        color: #909399;
        margin-bottom: 2px;
      }
      .row-name {
        font-size: 14px;
        color: $text-main;
        font-weight: 500;
      }
      .row-url {
        font-size: 12px;
        color: #c0c4cc;
        font-family: var(--ax-font-mono);
      }
    }
  }

  /* 底部统计条 */
  .perm-footer {
    padding: 10px 20px;
    border-top: 1px solid $border-color;
    display: flex;
    justify-content: space-between;
    align-items: center;

    .stats-bar {
      display: flex;
      flex-direction: column;
      gap: 4px;

      .main-stat {
        font-size: 14px;
        color: $text-main;
        .count-total {
          color: $primary;
          font-weight: bold;
          font-size: 16px;
          margin-left: 4px;
        }
      }

      .sub-stats {
        display: flex;
        gap: 12px;
        font-size: 12px;
        color: $text-sub;

        .stat-item {
          display: flex;
          align-items: center;
          .dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            margin-right: 4px;
            &.all {
              background: #303133;
            }
            &.get {
              background: #409eff;
            }
            &.post {
              background: #67c23a;
            }
            &.put {
              background: #e6a23c;
            }
            &.delete {
              background: #f56c6c;
            }
          }
        }
      }
    }
  }

  /* 空状态 */
  .empty-state {
    padding: 40px;
    text-align: center;
    color: #909399;

    &.placeholder {
      height: 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      p {
        margin-top: 12px;
      }
    }
  }
</style>
