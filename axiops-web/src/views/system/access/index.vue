<template>
  <div class="access-center" v-loading="loading">
    <header class="page-header">
      <div
        ><span class="eyebrow">ACCESS CONTROL</span><h1>权限中心</h1
        ><p>统一管理角色能力、菜单可见性与资源范围授权。</p></div
      >
      <div class="header-actions">
        <ElButton :icon="RefreshCw" circle title="刷新" @click="loadAll" />
        <ElButton
          v-if="activeTab === 'bindings' && canCreateBinding"
          type="primary"
          :icon="KeyRound"
          @click="bindingVisible = true"
          >新增授权</ElButton
        >
        <ElButton
          v-else-if="activeTab === 'roles' && canCreateRole"
          type="primary"
          :icon="Plus"
          @click="openRole()"
          >新建角色</ElButton
        >
        <ElButton
          v-else-if="activeTab === 'users' && canCreateUser"
          type="primary"
          :icon="Plus"
          @click="openUser()"
          >新建用户</ElButton
        >
        <ElButton
          v-else-if="activeTab === 'departments' && canCreateDepartment"
          type="primary"
          :icon="Plus"
          @click="openDepartment()"
          >新建部门</ElButton
        >
      </div>
    </header>

    <section class="metric-strip">
      <div
        ><span>有效授权</span><strong>{{ activeBindingCount }}</strong
        ><small>用户与部门范围绑定</small></div
      >
      <div
        ><span>范围角色</span><strong>{{ scopedRoles.length }}</strong
        ><small>可绑定到具体资源</small></div
      >
      <div
        ><span>用户</span><strong>{{ activeUsers.length }}</strong
        ><small>当前已启用账号</small></div
      >
      <div
        ><span>部门</span><strong>{{ departments.length }}</strong
        ><small>组织授权单元</small></div
      >
    </section>

    <ElTabs v-model="activeTab" class="access-tabs">
      <ElTabPane label="资源授权" name="bindings">
        <div class="table-toolbar">
          <div class="filter-row">
            <ElSelect
              v-model="bindingFilter.subjectType"
              clearable
              placeholder="对象类型"
              style="width: 130px"
              ><ElOption label="用户" value="user" /><ElOption label="部门" value="department"
            /></ElSelect>
            <ElSelect
              v-model="bindingFilter.scopeType"
              clearable
              placeholder="资源范围"
              style="width: 150px"
              ><ElOption
                v-for="item in scopeTypes"
                :key="item.value"
                :label="item.label"
                :value="item.value"
            /></ElSelect>
            <ElInput
              v-model="bindingFilter.keyword"
              :prefix-icon="Search"
              clearable
              placeholder="搜索用户、部门或角色"
              style="width: 260px"
            />
          </div>
          <span>{{ filteredBindings.length }} 条授权</span>
        </div>
        <ElTable :data="filteredBindings" row-key="id" class="data-table">
          <ElTableColumn label="授权对象" min-width="210"
            ><template #default="{ row }"
              ><div class="identity-cell"
                ><span class="subject-icon" :class="row.subject_type"
                  ><UserRound v-if="row.subject_type === 'user'" :size="16" /><Building2
                    v-else
                    :size="16" /></span
                ><span
                  ><strong>{{ subjectName(row) }}</strong
                  ><small
                    >{{ row.subject_type === 'user' ? '用户' : '部门' }} #{{
                      row.subject_id
                    }}</small
                  ></span
                ></div
              ></template
            ></ElTableColumn
          >
          <ElTableColumn label="角色" min-width="170"
            ><template #default="{ row }"
              ><div class="stack-cell"
                ><strong>{{ roleName(row.role_id) }}</strong
                ><small>{{ roleCode(row.role_id) }}</small></div
              ></template
            ></ElTableColumn
          >
          <ElTableColumn label="资源范围" min-width="220"
            ><template #default="{ row }"
              ><div class="stack-cell"
                ><span
                  ><ElTag size="small" effect="plain">{{ scopeLabel(row.scope_type) }}</ElTag>
                  {{ scopeName(row) }}</span
                ><small v-if="row.environment && row.environment !== '*'"
                  >环境：{{ row.environment }}</small
                ></div
              ></template
            ></ElTableColumn
          >
          <ElTableColumn label="有效期" width="180"
            ><template #default="{ row }">{{
              row.expires_at ? formatDateTime(row.expires_at) : '永久'
            }}</template></ElTableColumn
          >
          <ElTableColumn label="状态" width="90"
            ><template #default="{ row }"
              ><ElSwitch
                :model-value="row.status === 1"
                :disabled="!canUpdateBinding"
                @change="toggleBinding(row, $event)" /></template
          ></ElTableColumn>
          <ElTableColumn label="操作" width="76" fixed="right"
            ><template #default="{ row }"
              ><ElButton
                v-if="canDeleteBinding"
                circle
                text
                type="danger"
                :icon="Trash2"
                title="删除授权"
                @click="removeBinding(row)" /></template
          ></ElTableColumn>
        </ElTable>
        <ElEmpty v-if="!filteredBindings.length" description="暂无符合条件的资源授权" />
      </ElTabPane>

      <ElTabPane label="角色策略" name="roles">
        <div class="table-toolbar"
          ><span>角色定义能力，资源绑定定义生效范围。平台角色只能直接分配给用户。</span
          ><span>{{ roles.length }} 个角色</span></div
        >
        <ElTable :data="roles" row-key="id" class="data-table">
          <ElTableColumn label="角色" min-width="220"
            ><template #default="{ row }"
              ><div class="identity-cell"
                ><span class="role-symbol"><ShieldCheck :size="17" /></span
                ><span
                  ><strong>{{ row.name }}</strong
                  ><small>{{ row.code }}</small></span
                ></div
              ></template
            ></ElTableColumn
          >
          <ElTableColumn label="类型" width="120"
            ><template #default="{ row }"
              ><ElTag :type="row.type === 'system' ? 'info' : 'success'" effect="plain">{{
                row.type === 'system' ? '系统内置' : '自定义'
              }}</ElTag></template
            ></ElTableColumn
          >
          <ElTableColumn label="绑定方式" width="130"
            ><template #default="{ row }">{{
              row.binding_scope === 'platform' ? '平台角色' : '资源范围'
            }}</template></ElTableColumn
          >
          <ElTableColumn prop="description" label="说明" min-width="280" show-overflow-tooltip />
          <ElTableColumn label="操作" width="110" fixed="right"
            ><template #default="{ row }"
              ><ElButton
                v-if="canEditRole(row)"
                text
                type="primary"
                :icon="Pencil"
                @click="openRole(row.id)"
                >编辑</ElButton
              ><ElButton
                v-if="row.type !== 'system' && canDeleteRole"
                circle
                text
                type="danger"
                :icon="Trash2"
                title="删除角色"
                @click="removeRole(row)" /></template
          ></ElTableColumn>
        </ElTable>
      </ElTabPane>

      <ElTabPane label="用户管理" name="users">
        <div class="table-toolbar"
          ><span>平台角色用于全局管理；项目、服务和集群权限请使用资源授权。</span
          ><span>{{ users.length }} 个启用用户</span></div
        >
        <ElTable :data="users" row-key="id" class="data-table">
          <ElTableColumn label="用户" min-width="220"
            ><template #default="{ row }"
              ><div class="identity-cell"
                ><span class="subject-icon user"><UserRound :size="16" /></span
                ><span
                  ><strong>{{ row.nickname || row.username }}</strong
                  ><small>{{ row.username }}</small></span
                ></div
              ></template
            ></ElTableColumn
          >
          <ElTableColumn prop="email" label="邮箱" min-width="220" show-overflow-tooltip />
          <ElTableColumn label="直属部门" min-width="170"
            ><template #default="{ row }">{{
              departmentName(row.department_id)
            }}</template></ElTableColumn
          >
          <ElTableColumn label="状态" width="100"
            ><template #default="{ row }"
              ><ElTag :type="row.status === 1 ? 'success' : 'info'" effect="plain">{{
                row.status === 1 ? '启用' : '停用'
              }}</ElTag></template
            ></ElTableColumn
          >
          <ElTableColumn label="操作" width="130" fixed="right"
            ><template #default="{ row }"
              ><ElButton
                v-if="canUpdateUser"
                text
                type="primary"
                :icon="Pencil"
                @click="openUser(row)"
                >编辑</ElButton
              ><ElButton
                v-if="canDeleteUser"
                circle
                text
                type="danger"
                :icon="Trash2"
                title="删除用户"
                @click="removeUser(row)" /></template
          ></ElTableColumn>
        </ElTable>
      </ElTabPane>

      <ElTabPane label="部门管理" name="departments">
        <div class="table-toolbar"
          ><span>部门授权会自动作用于当前部门用户；调整用户部门后权限随组织关系变化。</span
          ><span>{{ departments.length }} 个部门</span></div
        >
        <ElTable :data="departmentTree" row-key="id" default-expand-all class="data-table">
          <ElTableColumn prop="name" label="部门名称" min-width="260" />
          <ElTableColumn label="负责人" min-width="180"
            ><template #default="{ row }">{{
              userName(row.leader_user_id)
            }}</template></ElTableColumn
          >
          <ElTableColumn prop="user_count" label="直属用户" width="110" />
          <ElTableColumn prop="sort_order" label="排序" width="90" />
          <ElTableColumn label="操作" width="130" fixed="right"
            ><template #default="{ row }"
              ><ElButton
                v-if="canUpdateDepartment"
                text
                type="primary"
                :icon="Pencil"
                @click="openDepartment(row)"
                >编辑</ElButton
              ><ElButton
                v-if="canDeleteDepartment"
                circle
                text
                type="danger"
                :icon="Trash2"
                title="删除部门"
                @click="removeDepartment(row)" /></template
          ></ElTableColumn>
        </ElTable>
        <ElEmpty v-if="!departments.length" description="暂无部门" />
      </ElTabPane>
    </ElTabs>

    <BindingDialog
      v-model="bindingVisible"
      :users="activeUsers"
      :departments="departments"
      :roles="roles"
      :projects="projects"
      :clusters="clusters"
      @saved="loadBindings"
    />
    <RolePolicyDialog v-model="roleVisible" :role-id="editingRoleId" @saved="loadRoles" />

    <ElDialog
      v-model="userVisible"
      :title="userForm.id ? '编辑用户' : '新建用户'"
      width="620px"
      destroy-on-close
    >
      <ElForm ref="userFormRef" :model="userForm" :rules="userRules" label-position="top">
        <div class="form-grid">
          <ElFormItem label="用户名" prop="username"
            ><ElInput v-model="userForm.username" :disabled="Boolean(userForm.id)" maxlength="64"
          /></ElFormItem>
          <ElFormItem v-if="!userForm.id" label="初始密码" prop="password"
            ><ElInput v-model="userForm.password" type="password" show-password maxlength="64"
          /></ElFormItem>
          <ElFormItem v-else label="账号状态"
            ><ElSwitch v-model="userForm.enabled" active-text="启用" inactive-text="停用"
          /></ElFormItem>
        </div>
        <div class="form-grid"
          ><ElFormItem label="姓名"
            ><ElInput v-model="userForm.nickname" maxlength="64" /></ElFormItem
          ><ElFormItem label="直属部门"
            ><ElSelect v-model="userForm.department_id" clearable filterable style="width: 100%"
              ><ElOption
                v-for="department in departments"
                :key="department.id"
                :value="department.id"
                :label="department.name" /></ElSelect></ElFormItem
        ></div>
        <div class="form-grid"
          ><ElFormItem label="邮箱"><ElInput v-model="userForm.email" /></ElFormItem
          ><ElFormItem label="手机"><ElInput v-model="userForm.phone" maxlength="32" /></ElFormItem
        ></div>
        <ElFormItem label="平台角色"
          ><ElSelect
            v-model="userForm.role_ids"
            multiple
            filterable
            style="width: 100%"
            placeholder="普通业务权限请使用资源授权"
            ><ElOption
              v-for="role in platformRoles"
              :key="role.id"
              :value="role.id"
              :label="role.name" /></ElSelect
        ></ElFormItem>
      </ElForm>
      <template #footer
        ><ElButton @click="userVisible = false">取消</ElButton
        ><ElButton type="primary" :loading="submitting" @click="saveUser">保存</ElButton></template
      >
    </ElDialog>

    <ElDialog
      v-model="departmentVisible"
      :title="departmentForm.id ? '编辑部门' : '新建部门'"
      width="540px"
      destroy-on-close
    >
      <ElForm
        ref="departmentFormRef"
        :model="departmentForm"
        :rules="departmentRules"
        label-position="top"
      >
        <ElFormItem label="部门名称" prop="name"
          ><ElInput v-model="departmentForm.name" maxlength="64"
        /></ElFormItem>
        <ElFormItem label="上级部门"
          ><ElTreeSelect
            v-model="departmentForm.parent_id"
            :data="departmentSelectTree"
            node-key="id"
            :props="{ label: 'name', children: 'children', value: 'id' }"
            check-strictly
            clearable
            :render-after-expand="false"
            style="width: 100%"
            placeholder="顶级部门"
        /></ElFormItem>
        <div class="form-grid"
          ><ElFormItem label="负责人"
            ><ElSelect
              v-model="departmentForm.leader_user_id"
              filterable
              clearable
              style="width: 100%"
              ><ElOption
                v-for="user in users"
                :key="user.id"
                :value="user.id"
                :label="user.nickname || user.username" /></ElSelect></ElFormItem
          ><ElFormItem label="排序"
            ><ElInputNumber
              v-model="departmentForm.sort_order"
              :min="0"
              :max="9999"
              style="width: 100%" /></ElFormItem
        ></div>
      </ElForm>
      <template #footer
        ><ElButton @click="departmentVisible = false">取消</ElButton
        ><ElButton type="primary" :loading="submitting" @click="saveDepartment"
          >保存</ElButton
        ></template
      >
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, ElMessageBox, type FormInstance, type FormRules } from 'element-plus'
  import {
    Building2,
    KeyRound,
    Pencil,
    Plus,
    RefreshCw,
    Search,
    ShieldCheck,
    Trash2,
    UserRound
  } from 'lucide-vue-next'
  import BindingDialog from './components/BindingDialog.vue'
  import RolePolicyDialog from './components/RolePolicyDialog.vue'
  import {
    createDepartmentApi,
    createUserApi,
    deleteDepartmentApi,
    deleteRoleApi,
    deleteRoleBindingApi,
    deleteUserApi,
    getUserApi,
    listClustersApi,
    listDepartmentsApi,
    listProjectsApi,
    listRoleBindingsApi,
    listRolesApi,
    listUsersApi,
    setUserRolesApi,
    updateDepartmentApi,
    updateRoleBindingApi,
    updateUserApi,
    type AxiOpsCluster,
    type AxiOpsDepartment,
    type AxiOpsProject,
    type AxiOpsRole,
    type AxiOpsUser,
    type RoleBinding
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'

  defineOptions({ name: 'AxiOpsAccessCenter' })
  const access = useAxiOpsAccess()
  const loading = ref(false)
  const submitting = ref(false)
  const activeTab = ref('bindings')
  const bindings = ref<RoleBinding[]>([])
  const roles = ref<AxiOpsRole[]>([])
  const users = ref<AxiOpsUser[]>([])
  const departments = ref<AxiOpsDepartment[]>([])
  const projects = ref<AxiOpsProject[]>([])
  const clusters = ref<AxiOpsCluster[]>([])
  const bindingVisible = ref(false)
  const roleVisible = ref(false)
  const editingRoleId = ref<number>()
  const userVisible = ref(false)
  const departmentVisible = ref(false)
  const userFormRef = ref<FormInstance>()
  const userForm = reactive({
    id: 0,
    username: '',
    password: '',
    nickname: '',
    email: '',
    phone: '',
    department_id: 0,
    role_ids: [] as number[],
    enabled: true
  })
  const departmentFormRef = ref<FormInstance>()
  const departmentForm = reactive({
    id: 0,
    name: '',
    parent_id: 0,
    sort_order: 0,
    leader_user_id: 0
  })
  const bindingFilter = reactive({ subjectType: '', scopeType: '', keyword: '' })
  const scopeTypes = [
    { label: '项目', value: 'project' },
    { label: '服务', value: 'service' },
    { label: '环境', value: 'environment' },
    { label: '构建', value: 'build' },
    { label: '工作流', value: 'workflow' },
    { label: '集群', value: 'cluster' }
  ]
  const departmentRules: FormRules = { name: [{ required: true, message: '请输入部门名称' }] }
  const userRules: FormRules = {
    username: [
      { required: true, pattern: /^[a-zA-Z][a-zA-Z0-9_.-]{1,63}$/, message: '请输入有效用户名' }
    ],
    password: [
      {
        validator: (_rule, value, callback) =>
          userForm.id || (value && value.length >= 8)
            ? callback()
            : callback(new Error('初始密码至少 8 位'))
      }
    ]
  }
  const scopedRoles = computed(() => roles.value.filter((item) => item.binding_scope === 'scoped'))
  const activeUsers = computed(() => users.value.filter((item) => item.status === 1))
  const platformRoles = computed(() =>
    roles.value.filter(
      (item) =>
        item.binding_scope === 'platform' &&
        (item.code !== 'super_admin' || access.can('system_role', 'assign'))
    )
  )
  const activeBindingCount = computed(
    () =>
      bindings.value.filter(
        (item) => item.status === 1 && (!item.expires_at || new Date(item.expires_at) > new Date())
      ).length
  )
  const canCreateBinding = computed(() => access.can('role_binding', 'create'))
  const canUpdateBinding = computed(() => access.can('role_binding', 'update'))
  const canDeleteBinding = computed(() => access.can('role_binding', 'delete'))
  const canCreateRole = computed(() => access.can('role', 'create'))
  const canDeleteRole = computed(() => access.can('role', 'delete'))
  const canCreateDepartment = computed(() => access.can('department', 'create'))
  const canUpdateDepartment = computed(() => access.can('department', 'update'))
  const canDeleteDepartment = computed(() => access.can('department', 'delete'))
  const canCreateUser = computed(() => access.can('user', 'create'))
  const canUpdateUser = computed(() => access.can('user', 'update'))
  const canDeleteUser = computed(() => access.can('user', 'delete'))
  const filteredBindings = computed(() =>
    bindings.value.filter((item) => {
      if (bindingFilter.subjectType && item.subject_type !== bindingFilter.subjectType) return false
      if (bindingFilter.scopeType && item.scope_type !== bindingFilter.scopeType) return false
      const keyword = bindingFilter.keyword.trim().toLowerCase()
      if (!keyword) return true
      return `${subjectName(item)} ${roleName(item.role_id)} ${scopeName(item)}`
        .toLowerCase()
        .includes(keyword)
    })
  )
  type DepartmentNode = AxiOpsDepartment & { children: DepartmentNode[] }
  const departmentTree = computed(() => buildDepartmentTree(departments.value))
  const departmentSelectTree = computed(() =>
    buildDepartmentTree(departments.value.filter((item) => item.id !== departmentForm.id))
  )

  function buildDepartmentTree(items: AxiOpsDepartment[]) {
    const map = new Map<number, DepartmentNode>()
    items.forEach((item) => map.set(item.id, { ...item, children: [] }))
    const roots: DepartmentNode[] = []
    map.forEach((item) => {
      const parent = map.get(item.parent_id)
      if (parent) parent.children.push(item)
      else roots.push(item)
    })
    return roots
  }
  const loadBindings = async () => {
    await access.load(true)
    bindings.value = await listRoleBindingsApi()
  }
  const loadRoles = async () => {
    await access.load(true)
    roles.value = await listRolesApi()
  }
  const loadAll = async () => {
    loading.value = true
    try {
      await access.load(true)
      const [bindingData, roleData, userData, departmentData, projectData, clusterData] =
        await Promise.all([
          listRoleBindingsApi(),
          listRolesApi(),
          listUsersApi(),
          listDepartmentsApi(),
          listProjectsApi({ page: 1, page_size: 200 }),
          listClustersApi()
        ])
      bindings.value = bindingData || []
      roles.value = roleData || []
      users.value = userData.list || []
      departments.value = departmentData || []
      projects.value = projectData.list || []
      clusters.value = clusterData || []
    } finally {
      loading.value = false
    }
  }
  const openRole = (id?: number) => {
    editingRoleId.value = id
    roleVisible.value = true
  }
  const openUser = async (item?: AxiOpsUser) => {
    Object.assign(userForm, {
      id: 0,
      username: '',
      password: '',
      nickname: '',
      email: '',
      phone: '',
      department_id: 0,
      role_ids: [],
      enabled: true
    })
    if (item) {
      const detail = await getUserApi(item.id)
      Object.assign(userForm, {
        id: detail.id,
        username: detail.username,
        nickname: detail.nickname,
        email: detail.email,
        phone: detail.phone || '',
        department_id: detail.department_id || 0,
        role_ids: detail.role_ids || [],
        enabled: detail.status === 1
      })
    }
    userVisible.value = true
  }
  const saveUser = async () => {
    if (!(await userFormRef.value?.validate())) return
    submitting.value = true
    try {
      if (userForm.id) {
        await updateUserApi(userForm.id, {
          nickname: userForm.nickname.trim(),
          email: userForm.email.trim(),
          phone: userForm.phone.trim(),
          status: userForm.enabled ? 1 : 0,
          department_id: userForm.department_id || 0
        })
        await setUserRolesApi(userForm.id, userForm.role_ids)
      } else {
        await createUserApi({
          username: userForm.username.trim(),
          password: userForm.password,
          nickname: userForm.nickname.trim(),
          email: userForm.email.trim(),
          phone: userForm.phone.trim(),
          department_id: userForm.department_id || 0,
          role_ids: userForm.role_ids
        })
      }
      ElMessage.success('用户已保存')
      userVisible.value = false
      users.value = (await listUsersApi()).list || []
    } finally {
      submitting.value = false
    }
  }
  const openDepartment = (item?: AxiOpsDepartment) => {
    Object.assign(
      departmentForm,
      item
        ? {
            id: item.id,
            name: item.name,
            parent_id: item.parent_id,
            sort_order: item.sort_order,
            leader_user_id: item.leader_user_id
          }
        : { id: 0, name: '', parent_id: 0, sort_order: 0, leader_user_id: 0 }
    )
    departmentVisible.value = true
  }
  const saveDepartment = async () => {
    if (!(await departmentFormRef.value?.validate())) return
    submitting.value = true
    try {
      const payload = {
        name: departmentForm.name.trim(),
        parent_id: departmentForm.parent_id || 0,
        sort_order: departmentForm.sort_order,
        leader_user_id: departmentForm.leader_user_id || 0
      }
      if (departmentForm.id) await updateDepartmentApi(departmentForm.id, payload)
      else await createDepartmentApi(payload)
      ElMessage.success('部门已保存')
      departmentVisible.value = false
      departments.value = await listDepartmentsApi()
    } finally {
      submitting.value = false
    }
  }
  const toggleBinding = async (row: RoleBinding, value: string | number | boolean) => {
    await updateRoleBindingApi(row.id, { status: value ? 1 : 0 })
    row.status = value ? 1 : 0
    ElMessage.success(value ? '授权已启用' : '授权已停用')
  }
  const removeBinding = async (row: RoleBinding) => {
    await ElMessageBox.confirm(`确定删除“${subjectName(row)}”的这条资源授权？`, '删除授权', {
      type: 'warning'
    })
    await deleteRoleBindingApi(row.id)
    ElMessage.success('授权已删除')
    await loadBindings()
  }
  const removeRole = async (row: AxiOpsRole) => {
    await ElMessageBox.confirm(`确定删除角色“${row.name}”？已被绑定的角色无法删除。`, '删除角色', {
      type: 'warning'
    })
    await deleteRoleApi(row.id)
    ElMessage.success('角色已删除')
    await loadRoles()
  }
  const removeDepartment = async (row: AxiOpsDepartment) => {
    await ElMessageBox.confirm(
      `确定删除部门“${row.name}”？存在子部门或直属用户时不能删除。`,
      '删除部门',
      { type: 'warning' }
    )
    await deleteDepartmentApi(row.id)
    ElMessage.success('部门已删除')
    departments.value = await listDepartmentsApi()
  }
  const removeUser = async (row: AxiOpsUser) => {
    await ElMessageBox.confirm(
      `确定删除用户“${row.nickname || row.username}”？删除后其所有授权立即失效。`,
      '删除用户',
      { type: 'warning' }
    )
    await deleteUserApi(row.id)
    ElMessage.success('用户已删除')
    users.value = (await listUsersApi()).list || []
  }
  const canEditRole = (role: AxiOpsRole) =>
    access.can('role', 'update') && (role.type !== 'system' || access.can('system_role', 'update'))
  const subjectName = (row: RoleBinding) =>
    row.subject_type === 'user'
      ? userName(row.subject_id)
      : departments.value.find((item) => item.id === row.subject_id)?.name ||
        `部门 #${row.subject_id}`
  const userName = (id: number) => {
    if (!id) return '-'
    const user = users.value.find((item) => item.id === id)
    return user ? user.nickname || user.username : `用户 #${id}`
  }
  const departmentName = (id: number) =>
    id ? departments.value.find((item) => item.id === id)?.name || `部门 #${id}` : '-'
  const roleName = (id: number) => roles.value.find((item) => item.id === id)?.name || `角色 #${id}`
  const roleCode = (id: number) => roles.value.find((item) => item.id === id)?.code || '-'
  const scopeLabel = (type: string) => scopeTypes.find((item) => item.value === type)?.label || type
  const scopeName = (row: RoleBinding) =>
    row.scope_type === 'project'
      ? projects.value.find((item) => item.id === row.scope_id)?.name || `项目 #${row.scope_id}`
      : row.scope_type === 'cluster'
        ? clusters.value.find((item) => item.id === row.scope_id)?.name || `集群 #${row.scope_id}`
        : `${scopeLabel(row.scope_type)} #${row.scope_id}`
  const formatDateTime = (value: string) =>
    value ? new Date(value).toLocaleString('zh-CN', { hour12: false }) : '-'
  onMounted(loadAll)
</script>

<style scoped lang="scss">
  .access-center {
    min-height: 100%;
    padding: 0 20px 28px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .page-header {
    min-height: 112px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 24px;
    border-bottom: 1px solid var(--art-border-color);
  }
  .page-header h1 {
    margin: 3px 0 0;
    font-size: 25px;
    letter-spacing: 0;
  }
  .page-header p {
    margin: 5px 0 0;
    color: var(--art-text-gray-500);
  }
  .eyebrow {
    color: #16805b;
    font-size: 11px;
    font-weight: 700;
  }
  .header-actions,
  .filter-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .metric-strip {
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    border-bottom: 1px solid var(--art-border-color);
  }
  .metric-strip > div {
    min-height: 106px;
    display: flex;
    flex-direction: column;
    justify-content: center;
    padding: 16px 22px;
    border-right: 1px solid var(--art-border-color);
  }
  .metric-strip > div:last-child {
    border: 0;
  }
  .metric-strip span,
  .metric-strip small {
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .metric-strip strong {
    margin: 6px 0 2px;
    font-size: 28px;
    line-height: 30px;
  }
  .access-tabs {
    margin-top: 12px;
  }
  .access-tabs :deep(.el-tabs__header) {
    margin-bottom: 12px;
  }
  .table-toolbar {
    min-height: 50px;
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    color: var(--art-text-gray-500);
    font-size: 13px;
  }
  .data-table {
    border-top: 1px solid var(--art-border-color);
  }
  .identity-cell {
    display: flex;
    align-items: center;
    gap: 10px;
    min-width: 0;
  }
  .identity-cell > span:last-child,
  .stack-cell {
    min-width: 0;
    display: flex;
    flex-direction: column;
  }
  .identity-cell strong,
  .identity-cell small,
  .stack-cell strong,
  .stack-cell small {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .identity-cell small,
  .stack-cell small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
    font-size: 12px;
  }
  .subject-icon,
  .role-symbol {
    width: 32px;
    height: 32px;
    flex: 0 0 32px;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    border-radius: 6px;
    color: #fff;
    background: #2563eb;
  }
  .subject-icon.department {
    background: #d97706;
  }
  .role-symbol {
    background: #16805b;
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  @media (max-width: 900px) {
    .metric-strip {
      grid-template-columns: 1fr 1fr;
    }
    .metric-strip > div:nth-child(2) {
      border-right: 0;
    }
    .metric-strip > div:nth-child(-n + 2) {
      border-bottom: 1px solid var(--art-border-color);
    }
    .table-toolbar {
      align-items: flex-start;
      flex-direction: column;
      padding: 8px 0;
    }
    .filter-row {
      width: 100%;
      flex-wrap: wrap;
    }
  }
  @media (max-width: 600px) {
    .access-center {
      padding: 0 12px 20px;
    }
    .page-header {
      min-height: 132px;
      align-items: flex-start;
      flex-direction: column;
      justify-content: center;
    }
    .header-actions {
      width: 100%;
    }
    .header-actions .el-button:not(:first-child) {
      flex: 1;
    }
    .metric-strip > div {
      min-height: 90px;
      padding: 12px;
    }
    .metric-strip strong {
      font-size: 23px;
    }
    .filter-row > * {
      width: 100% !important;
    }
    .form-grid {
      grid-template-columns: 1fr;
      gap: 0;
    }
  }
</style>
