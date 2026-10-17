<template>
  <ElDialog v-model="visible" title="新增资源授权" width="680px" destroy-on-close>
    <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
      <div class="form-grid">
        <ElFormItem label="授权对象" prop="subject_type">
          <ElSegmented
            v-model="form.subject_type"
            :options="subjectTypes"
            @change="form.subject_id = 0"
          />
        </ElFormItem>
        <ElFormItem :label="form.subject_type === 'user' ? '用户' : '部门'" prop="subject_id">
          <ElSelect
            v-model="form.subject_id"
            filterable
            style="width: 100%"
            placeholder="请选择授权对象"
          >
            <ElOption
              v-for="item in subjectOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </ElSelect>
        </ElFormItem>
      </div>

      <ElFormItem label="范围角色" prop="role_id">
        <ElSelect
          v-model="form.role_id"
          filterable
          style="width: 100%"
          placeholder="请选择范围角色"
        >
          <ElOption v-for="role in scopedRoles" :key="role.id" :value="role.id" :label="role.name">
            <div class="role-option"
              ><span>{{ role.name }}</span
              ><small>{{ role.description || role.code }}</small></div
            >
          </ElOption>
        </ElSelect>
      </ElFormItem>

      <ElDivider content-position="left">资源范围</ElDivider>
      <ElFormItem label="范围类型" prop="scope_type">
        <ElRadioGroup v-model="form.scope_type" @change="resetScopeTarget">
          <ElRadioButton v-for="scope in scopeTypes" :key="scope.value" :value="scope.value">{{
            scope.label
          }}</ElRadioButton>
        </ElRadioGroup>
      </ElFormItem>

      <ElFormItem v-if="needsProject" label="所属项目" prop="project_id">
        <ElSelect
          v-model="projectId"
          filterable
          style="width: 100%"
          placeholder="先选择项目"
          @change="loadProjectTargets"
        >
          <ElOption
            v-for="project in projects"
            :key="project.id"
            :value="project.id"
            :label="`${project.name} (${project.code})`"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem :label="targetLabel" prop="scope_id">
        <ElSelect
          v-model="form.scope_id"
          filterable
          style="width: 100%"
          :loading="targetLoading"
          :disabled="needsProject && !projectId"
          placeholder="请选择具体资源"
          @change="syncEnvironment"
        >
          <ElOption
            v-for="target in targetOptions"
            :key="target.value"
            :value="target.value"
            :label="target.label"
          />
        </ElSelect>
      </ElFormItem>

      <ElFormItem label="失效时间">
        <ElDatePicker
          v-model="form.expires_at"
          type="datetime"
          value-format="YYYY-MM-DDTHH:mm:ssZ"
          placeholder="永久有效"
          style="width: 100%"
        />
      </ElFormItem>
    </ElForm>

    <div class="scope-note">
      <ShieldCheck :size="18" />
      <span>授权仅在所选资源生效。后端会在每次查看、构建、部署和审批操作时再次校验。</span>
    </div>
    <template #footer
      ><ElButton @click="visible = false">取消</ElButton
      ><ElButton type="primary" :loading="submitting" @click="submit">确认授权</ElButton></template
    >
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import { ShieldCheck } from 'lucide-vue-next'
  import {
    createRoleBindingApi,
    listBuildRunsApi,
    listEnvironmentsApi,
    listAllServicesApi,
    listWorkflowsApi,
    type AxiOpsCluster,
    type AxiOpsDepartment,
    type AxiOpsProject,
    type AxiOpsRole,
    type AxiOpsUser,
    type CreateRoleBindingInput,
    type RoleBinding
  } from '@/api/axiops'

  const props = defineProps<{
    modelValue: boolean
    users: AxiOpsUser[]
    departments: AxiOpsDepartment[]
    roles: AxiOpsRole[]
    projects: AxiOpsProject[]
    clusters: AxiOpsCluster[]
  }>()
  const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [] }>()
  const visible = computed({
    get: () => props.modelValue,
    set: (value) => emit('update:modelValue', value)
  })
  type ScopeType = RoleBinding['scope_type']
  const formRef = ref<FormInstance>()
  const submitting = ref(false)
  const targetLoading = ref(false)
  const projectId = ref(0)
  const targetOptions = ref<{ value: number; label: string; environment?: string }[]>([])
  const form = reactive<CreateRoleBindingInput & { expires_at?: string }>({
    subject_type: 'user',
    subject_id: 0,
    role_id: 0,
    scope_type: 'project',
    scope_id: 0,
    environment: '*',
    expires_at: ''
  })
  const subjectTypes = [
    { label: '用户', value: 'user' },
    { label: '部门', value: 'department' }
  ]
  const scopeTypes: { label: string; value: ScopeType }[] = [
    { label: '项目', value: 'project' },
    { label: '服务', value: 'service' },
    { label: '环境', value: 'environment' },
    { label: '构建', value: 'build' },
    { label: '工作流', value: 'workflow' },
    { label: '集群', value: 'cluster' }
  ]
  const scopedRoles = computed(() =>
    props.roles.filter((role) => role.binding_scope === 'scoped' && role.code !== 'super_admin')
  )
  const subjectOptions = computed(() =>
    form.subject_type === 'user'
      ? props.users.map((user) => ({
          value: user.id,
          label: `${user.nickname || user.username} (${user.username})`
        }))
      : props.departments.map((department) => ({ value: department.id, label: department.name }))
  )
  const needsProject = computed(() =>
    ['service', 'environment', 'build', 'workflow'].includes(form.scope_type)
  )
  const targetLabel = computed(
    () => scopeTypes.find((item) => item.value === form.scope_type)?.label || '资源'
  )
  const rules: FormRules = {
    subject_id: [{ required: true, type: 'number', min: 1, message: '请选择授权对象' }],
    role_id: [{ required: true, type: 'number', min: 1, message: '请选择角色' }],
    scope_id: [{ required: true, type: 'number', min: 1, message: '请选择资源' }]
  }

  const resetScopeTarget = () => {
    projectId.value = 0
    form.scope_id = 0
    form.environment = '*'
    if (form.scope_type === 'project')
      targetOptions.value = props.projects.map((item) => ({
        value: item.id,
        label: `${item.name} (${item.code})`
      }))
    else if (form.scope_type === 'cluster')
      targetOptions.value = props.clusters.map((item) => ({
        value: item.id,
        label: `${item.name} (${item.region || item.provider || item.code})`
      }))
    else targetOptions.value = []
  }
  const loadProjectTargets = async () => {
    form.scope_id = 0
    form.environment = '*'
    if (!projectId.value) return
    targetLoading.value = true
    try {
      if (form.scope_type === 'service') {
        const result = await listAllServicesApi(projectId.value)
        targetOptions.value = result.map((item) => ({
          value: item.id,
          label: `${item.name} (${item.code})`
        }))
      } else if (form.scope_type === 'environment') {
        targetOptions.value = (await listEnvironmentsApi(projectId.value)).map((item) => ({
          value: item.id,
          label: `${item.name} (${item.code})`,
          environment: item.code
        }))
      } else if (form.scope_type === 'build') {
        const result = await listBuildRunsApi(projectId.value)
        targetOptions.value = (result.list || []).map((item) => ({
          value: item.id,
          label:
            `#${item.build_number} ${item.git_branch || ''} ${item.git_commit_short || ''}`.trim()
        }))
      } else if (form.scope_type === 'workflow') {
        targetOptions.value = (await listWorkflowsApi(projectId.value)).map((item) => ({
          value: item.id,
          label: `${item.name} (${item.code})`
        }))
      }
    } finally {
      targetLoading.value = false
    }
  }
  const syncEnvironment = () => {
    form.environment =
      form.scope_type === 'environment'
        ? targetOptions.value.find((item) => item.value === form.scope_id)?.environment || ''
        : '*'
  }
  const reset = () => {
    Object.assign(form, {
      subject_type: 'user',
      subject_id: 0,
      role_id: 0,
      scope_type: 'project',
      scope_id: 0,
      environment: '*',
      expires_at: ''
    })
    resetScopeTarget()
  }
  const submit = async () => {
    if (!(await formRef.value?.validate())) return
    submitting.value = true
    try {
      await createRoleBindingApi({
        subject_type: form.subject_type,
        subject_id: form.subject_id,
        role_id: form.role_id,
        scope_type: form.scope_type,
        scope_id: form.scope_id,
        environment: form.environment,
        expires_at: form.expires_at || undefined
      })
      ElMessage.success('资源授权已生效')
      visible.value = false
      emit('saved')
    } finally {
      submitting.value = false
    }
  }
  watch(
    () => props.modelValue,
    (open) => {
      if (open) reset()
    }
  )
</script>

<style scoped lang="scss">
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  .role-option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 20px;
  }
  .role-option small {
    color: var(--art-text-gray-500);
  }
  .scope-note {
    min-height: 52px;
    display: flex;
    align-items: center;
    gap: 10px;
    padding: 0 14px;
    border-left: 3px solid #16805b;
    background: rgb(22 128 91 / 7%);
    color: var(--art-text-gray-700);
    font-size: 13px;
  }
  @media (max-width: 640px) {
    .form-grid {
      grid-template-columns: 1fr;
      gap: 0;
    }
  }
</style>
