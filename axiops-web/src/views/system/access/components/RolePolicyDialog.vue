<template>
  <ElDialog
    v-model="visible"
    :title="roleId ? '编辑角色' : '新建范围角色'"
    width="920px"
    destroy-on-close
  >
    <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
      <div class="identity-grid">
        <ElFormItem label="角色名称" prop="name"
          ><ElInput v-model="form.name" maxlength="64"
        /></ElFormItem>
        <ElFormItem label="角色编码" prop="code">
          <ElInput
            v-model="form.code"
            :disabled="Boolean(roleId)"
            placeholder="例如：project_readonly"
          />
        </ElFormItem>
      </div>
      <ElFormItem label="角色说明"
        ><ElInput v-model="form.description" type="textarea" :rows="2" maxlength="256"
      /></ElFormItem>
    </ElForm>

    <ElTabs v-model="activeTab" class="policy-tabs">
      <ElTabPane label="权限矩阵" name="permissions">
        <div class="policy-toolbar">
          <span>勾选角色允许执行的操作</span>
          <div
            ><ElButton text @click="selectReadOnly">只读模板</ElButton
            ><ElButton text @click="clearAllows">清空</ElButton></div
          >
        </div>
        <div v-for="group in permissionGroups" :key="group.name" class="permission-group">
          <h4>{{ group.name }}</h4>
          <div v-for="item in group.resources" :key="item.resource" class="permission-row">
            <div
              ><strong>{{ item.label }}</strong
              ><small>{{ item.description }}</small></div
            >
            <ElCheckboxGroup v-model="allowKeys">
              <ElCheckbox
                v-for="action in item.actions"
                :key="action"
                :value="permissionKey(item.resource, action)"
              >
                {{ actionLabels[action] || action }}
              </ElCheckbox>
            </ElCheckboxGroup>
          </div>
        </div>

        <div class="deny-section">
          <div class="section-heading"
            ><div><h4>显式拒绝</h4><p>拒绝策略优先于允许策略，适合排除高风险操作。</p></div
            ><ElButton :icon="Plus" @click="addDeny">添加拒绝</ElButton></div
          >
          <div v-for="(item, index) in denyPermissions" :key="index" class="deny-row">
            <ElSelect v-model="item.resource" filterable placeholder="资源"
              ><ElOption
                v-for="option in resourceOptions"
                :key="option.value"
                :label="option.label"
                :value="option.value"
            /></ElSelect>
            <ElSelect v-model="item.action" placeholder="操作"
              ><ElOption
                v-for="action in actionsFor(item.resource)"
                :key="action"
                :label="actionLabels[action] || action"
                :value="action"
            /></ElSelect>
            <ElButton
              circle
              text
              type="danger"
              :icon="Trash2"
              aria-label="删除拒绝策略"
              @click="denyPermissions.splice(index, 1)"
            />
          </div>
          <ElEmpty v-if="!denyPermissions.length" description="未配置显式拒绝" :image-size="56" />
        </div>
      </ElTabPane>

      <ElTabPane label="可见菜单" name="menus">
        <div class="menu-toolbar"
          ><span>用户只有在绑定该角色后才能看到已选菜单。</span
          ><ElCheckbox v-model="menuExpandAll">展开全部</ElCheckbox></div
        >
        <ElTree
          ref="menuTreeRef"
          :data="menuTree"
          node-key="id"
          show-checkbox
          default-expand-all
          :props="{ label: 'name', children: 'children' }"
        />
      </ElTabPane>
    </ElTabs>

    <template #footer>
      <ElButton @click="visible = false">取消</ElButton>
      <ElButton type="primary" :loading="submitting" @click="submit">保存角色</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, nextTick, reactive, ref, watch } from 'vue'
  import { ElMessage, type FormInstance, type FormRules, type ElTree } from 'element-plus'
  import { Plus, Trash2 } from 'lucide-vue-next'
  import {
    createRoleApi,
    getRoleApi,
    listAccessMenusApi,
    updateRoleApi,
    type AxiOpsMenu,
    type RolePermission
  } from '@/api/axiops'

  const props = defineProps<{ modelValue: boolean; roleId?: number }>()
  const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [] }>()
  const visible = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })
  const activeTab = ref('permissions')
  const submitting = ref(false)
  const formRef = ref<FormInstance>()
  const menuTreeRef = ref<InstanceType<typeof ElTree>>()
  const menuExpandAll = ref(true)
  const menus = ref<AxiOpsMenu[]>([])
  const allowKeys = ref<string[]>([])
  const denyPermissions = ref<RolePermission[]>([])
  const form = reactive({
    name: '',
    code: '',
    description: '',
    binding_scope: 'scoped' as 'platform' | 'scoped'
  })
  const rules: FormRules = {
    name: [{ required: true, message: '请输入角色名称' }],
    code: [
      { required: true, pattern: /^[a-z][a-z0-9_]{1,63}$/, message: '使用小写字母、数字和下划线' }
    ]
  }

  const permissionGroups = [
    {
      name: '项目与交付',
      resources: [
        {
          resource: 'project',
          label: '项目',
          description: '项目资料与成员边界',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'service',
          label: '服务',
          description: 'YAML、Helm 与主机服务',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'environment',
          label: '环境',
          description: '命名空间或主机组环境',
          actions: ['view', 'create', 'update', 'delete', 'manage_production']
        },
        {
          resource: 'workflow',
          label: '工作流',
          description: '编排、执行与生产审批',
          actions: ['view', 'create', 'update', 'run', 'cancel', 'approve']
        },
        {
          resource: 'build',
          label: '构建任务',
          description: '构建执行与状态',
          actions: ['view', 'run', 'cancel']
        },
        {
          resource: 'build_log',
          label: '构建日志',
          description: '构建过程与归档日志',
          actions: ['view']
        },
        {
          resource: 'deployment',
          label: '部署发布',
          description: '发布、回滚与工作负载重启',
          actions: ['view', 'deploy', 'rollback', 'restart']
        }
      ]
    },
    {
      name: '基础设施与可观测',
      resources: [
        {
          resource: 'cluster',
          label: 'Kubernetes 集群',
          description: '集群接入与资源查看',
          actions: ['view', 'create', 'update', 'delete', 'test']
        },
        {
          resource: 'monitoring',
          label: '监控',
          description: '按范围查看指标；管理用于集群监控配置和规则',
          actions: ['view', 'export', 'manage']
        },
        {
          resource: 'logging',
          label: '日志',
          description: '查看与导出独立授权；原始表达式需平台范围权限',
          actions: ['view', 'export', 'manage']
        },
        {
          resource: 'credential',
          label: '凭证',
          description: '仓库、集群与主机凭证',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'code_source',
          label: '代码源',
          description: 'Git 仓库连接',
          actions: ['view', 'create', 'update', 'delete']
        }
      ]
    },
    {
      name: '权限中台',
      resources: [
        {
          resource: 'user',
          label: '用户',
          description: '用户账号与平台角色',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'department',
          label: '部门',
          description: '组织结构与部门成员',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'role',
          label: '角色',
          description: '角色策略与菜单',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'role_binding',
          label: '资源授权',
          description: '用户和部门的范围授权',
          actions: ['view', 'create', 'update', 'delete']
        },
        {
          resource: 'menu',
          label: '菜单',
          description: '菜单可见范围',
          actions: ['view', 'update']
        }
      ]
    }
  ]
  const actionLabels: Record<string, string> = {
    view: '查看',
    create: '新建',
    update: '编辑',
    delete: '删除',
    run: '执行',
    cancel: '取消',
    approve: '审批',
    deploy: '部署',
    rollback: '回滚',
    restart: '滚动重启',
    export: '导出',
    manage: '管理配置',
    test: '测试连接',
    manage_production: '管理生产环境'
  }
  const resourceOptions = computed(() =>
    permissionGroups.flatMap((group) =>
      group.resources.map((item) => ({ label: item.label, value: item.resource }))
    )
  )
  const menuTree = computed(() => {
    const nodes = new Map<number, AxiOpsMenu & { children: AxiOpsMenu[] }>()
    menus.value.forEach((menu) => nodes.set(menu.id, { ...menu, children: [] }))
    const roots: (AxiOpsMenu & { children: AxiOpsMenu[] })[] = []
    nodes.forEach((node) => {
      const parent = nodes.get(node.parent_id)
      if (parent) parent.children.push(node)
      else roots.push(node)
    })
    return roots
  })
  const permissionKey = (resource: string, action: string) => `${resource}:${action}`
  const actionsFor = (resource: string) =>
    permissionGroups.flatMap((group) => group.resources).find((item) => item.resource === resource)
      ?.actions || ['view']
  const addDeny = () =>
    denyPermissions.value.push({ resource: 'project', action: 'delete', effect: 'deny' })
  const clearAllows = () => {
    allowKeys.value = []
  }
  const selectReadOnly = () => {
    allowKeys.value = permissionGroups
      .flatMap((group) => group.resources.map((item) => permissionKey(item.resource, 'view')))
      .filter((key) => actionsFor(key.split(':')[0]).includes('view'))
  }

  const reset = () => {
    Object.assign(form, { name: '', code: '', description: '', binding_scope: 'scoped' })
    allowKeys.value = []
    denyPermissions.value = []
    activeTab.value = 'permissions'
  }
  const load = async () => {
    reset()
    menus.value = await listAccessMenusApi()
    if (!props.roleId) return
    const detail = await getRoleApi(props.roleId)
    Object.assign(form, {
      name: detail.name,
      code: detail.code,
      description: detail.description,
      binding_scope: detail.binding_scope
    })
    allowKeys.value = detail.permissions
      .filter((item) => item.effect === 'allow')
      .map((item) => permissionKey(item.resource, item.action))
    denyPermissions.value = detail.permissions
      .filter((item) => item.effect === 'deny')
      .map((item) => ({ resource: item.resource, action: item.action, effect: 'deny' }))
    await nextTick()
    menuTreeRef.value?.setCheckedKeys(detail.menu_ids || [])
  }
  const submit = async () => {
    if (!(await formRef.value?.validate())) return
    const permissions: RolePermission[] = [
      ...allowKeys.value.map((key) => {
        const [resource, action] = key.split(':')
        return { resource, action, effect: 'allow' as const }
      }),
      ...denyPermissions.value
        .filter((item) => item.resource && item.action)
        .map((item) => ({ ...item, effect: 'deny' as const }))
    ]
    const checked = (menuTreeRef.value?.getCheckedKeys(false) || []) as number[]
    const halfChecked = (menuTreeRef.value?.getHalfCheckedKeys() || []) as number[]
    const payload = {
      name: form.name.trim(),
      description: form.description.trim(),
      binding_scope: form.binding_scope,
      permissions,
      menu_ids: [...new Set([...checked, ...halfChecked])]
    }
    submitting.value = true
    try {
      if (props.roleId) await updateRoleApi(props.roleId, payload)
      else await createRoleApi({ ...payload, code: form.code.trim(), binding_scope: 'scoped' })
      ElMessage.success('角色策略已保存')
      visible.value = false
      emit('saved')
    } finally {
      submitting.value = false
    }
  }
  watch(
    () => props.modelValue,
    (open) => {
      if (open) void load()
    }
  )
  watch(menuExpandAll, (expanded) => {
    const tree = menuTreeRef.value as any
    Object.values(tree?.store?.nodesMap || {}).forEach((node: any) => {
      node.expanded = expanded
    })
  })
</script>

<style scoped lang="scss">
  .identity-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .policy-tabs {
    margin-top: 4px;
  }
  .policy-toolbar,
  .menu-toolbar,
  .section-heading {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 14px;
    color: var(--art-text-gray-500);
    font-size: 13px;
  }
  .permission-group {
    margin-top: 18px;
    border-top: 1px solid var(--art-border-color);
  }
  .permission-group h4 {
    margin: 0;
    padding: 12px 0 8px;
    font-size: 13px;
  }
  .permission-row {
    min-height: 62px;
    display: grid;
    grid-template-columns: 210px 1fr;
    align-items: center;
    gap: 20px;
    border-top: 1px solid var(--art-border-color);
  }
  .permission-row > div {
    display: flex;
    flex-direction: column;
  }
  .permission-row small {
    margin-top: 3px;
    color: var(--art-text-gray-500);
  }
  .permission-row :deep(.el-checkbox) {
    min-width: 76px;
    margin-right: 12px;
  }
  .deny-section {
    margin-top: 24px;
  }
  .section-heading h4 {
    margin: 0;
    color: var(--art-text-gray-900);
  }
  .section-heading p {
    margin: 4px 0 0;
  }
  .deny-row {
    display: grid;
    grid-template-columns: 1fr 1fr 36px;
    gap: 10px;
    margin-top: 10px;
  }
  .menu-toolbar {
    margin-bottom: 12px;
  }
  @media (max-width: 700px) {
    .identity-grid {
      grid-template-columns: 1fr;
      gap: 0;
    }
    .permission-row {
      grid-template-columns: 1fr;
      gap: 8px;
      padding: 10px 0;
    }
  }
</style>
