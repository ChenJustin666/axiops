<template>
  <ElDialog
    :model-value="modelValue"
    :title="title"
    width="620px"
    align-center
    :close-on-click-modal="false"
    @update:model-value="emit('update:modelValue', $event)"
    @open="handleOpen"
    @close="resetForm"
  >
    <ElForm ref="formRef" :model="formData" :rules="rules" :label-width="labelWidth">
      <ElRow :gutter="16">
        <ElCol :span="12">
          <ElFormItem label="凭据名称" prop="name">
            <ElInput v-model="formData.name" placeholder="请输入凭据名称" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem label="凭据编码" prop="code">
            <ElInput v-model="formData.code" placeholder="请输入凭据编码" />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElRow :gutter="16">
        <ElCol :span="12">
          <ElFormItem label="凭据级别" prop="scope">
            <ElRadioGroup v-model="formData.scope" @change="handleScopeChange">
              <ElRadioButton v-for="item in scopeOptions" :key="item.value" :value="item.value">
                {{ item.label }}
              </ElRadioButton>
            </ElRadioGroup>
          </ElFormItem>
        </ElCol>
        <ElCol v-if="formData.scope === 'project'" :span="12">
          <ElFormItem label="所属项目" prop="projectId">
            <ElInput :model-value="currentProjectLabel" disabled />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElFormItem label="凭据类型" prop="credentialType">
        <ElSelect v-model="formData.credentialType" style="width: 100%">
          <ElOption
            v-for="item in resolvedTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </ElSelect>
      </ElFormItem>
      <ElRow v-if="formData.credentialType === 'username_password'" :gutter="16">
        <ElCol :span="12">
          <ElFormItem label="用户名" prop="username">
            <ElInput v-model="formData.username" placeholder="请输入用户名" />
          </ElFormItem>
        </ElCol>
        <ElCol :span="12">
          <ElFormItem label="密码" prop="password">
            <ElInput
              v-model="formData.password"
              type="password"
              show-password
              placeholder="请输入密码"
            />
          </ElFormItem>
        </ElCol>
      </ElRow>
      <ElFormItem v-if="formData.credentialType === 'token'" label="Token" prop="token">
        <ElInput
          v-model="formData.token"
          type="password"
          show-password
          placeholder="请输入 Token"
        />
      </ElFormItem>
      <template v-if="formData.credentialType === 'ssh_key'">
        <ElFormItem label="用户名" prop="username">
          <ElInput v-model="formData.username" placeholder="请输入用户名" />
        </ElFormItem>
        <ElFormItem label="私钥" prop="privateKey">
          <ElInput
            v-model="formData.privateKey"
            type="textarea"
            :rows="5"
            placeholder="请输入 SSH 私钥"
          />
        </ElFormItem>
        <ElFormItem label="Passphrase" prop="passphrase">
          <ElInput
            v-model="formData.passphrase"
            type="password"
            show-password
            placeholder="请输入私钥口令"
          />
        </ElFormItem>
      </template>
      <ElFormItem
        v-if="formData.credentialType === 'kubeconfig'"
        label="Kubeconfig"
        prop="kubeconfig"
      >
        <ElInput
          v-model="formData.kubeconfig"
          type="textarea"
          :rows="6"
          placeholder="请输入 kubeconfig"
        />
      </ElFormItem>
      <ElFormItem v-if="formData.credentialType === 'secret_text'" label="Secret" prop="secretText">
        <ElInput
          v-model="formData.secretText"
          type="password"
          show-password
          placeholder="请输入密文"
        />
      </ElFormItem>
      <ElFormItem v-if="formData.credentialType === 'certificate'" label="证书" prop="certificate">
        <ElInput
          v-model="formData.certificate"
          type="textarea"
          :rows="5"
          placeholder="请输入证书内容"
        />
      </ElFormItem>
      <ElFormItem v-if="formData.credentialType === 'json'" label="JSON" prop="jsonData">
        <ElInput
          v-model="formData.jsonData"
          type="textarea"
          :rows="6"
          placeholder='{"username":"xxx","password":"xxx"}'
        />
      </ElFormItem>
      <ElFormItem label="描述" prop="description">
        <ElInput
          v-model="formData.description"
          type="textarea"
          :rows="3"
          placeholder="请输入凭据描述"
        />
      </ElFormItem>
    </ElForm>
    <template #footer>
      <ElButton :disabled="submitLoading" @click="closeDialog">取消</ElButton>
      <ElButton type="primary" :loading="submitLoading" @click="handleSubmit">保存</ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { computed, reactive, ref, watch } from 'vue'
  import { storeToRefs } from 'pinia'
  import { ElMessage } from 'element-plus'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    createDevopsCredentialApi,
    type DevopsCredential,
    type DevopsCredentialForm
  } from '@/api/devops/channel'
  import { useDevopsProjectStore } from '@/store/modules/devopsProject'
  import { useUserStore } from '@/store/modules/user'

  interface CredentialTypeOption {
    label: string
    value: string
  }

  const props = withDefaults(
    defineProps<{
      modelValue: boolean
      title?: string
      typeOptions?: CredentialTypeOption[]
      defaultType?: string
      channelGroupCode?: string
      channelType?: string
      labelWidth?: string
    }>(),
    {
      title: '新增凭据',
      typeOptions: () => [
        { label: '用户名密码', value: 'username_password' },
        { label: 'Token', value: 'token' },
        { label: 'SSH Key', value: 'ssh_key' },
        { label: 'Kubeconfig', value: 'kubeconfig' },
        { label: 'Secret Text', value: 'secret_text' },
        { label: '证书', value: 'certificate' },
        { label: 'JSON', value: 'json' }
      ],
      defaultType: '',
      labelWidth: '108px'
    }
  )

  const emit = defineEmits<{
    'update:modelValue': [value: boolean]
    created: [value: DevopsCredential]
  }>()

  const userStore = useUserStore()
  const devopsProjectStore = useDevopsProjectStore()
  const { selectedProject, selectedProjectId } = storeToRefs(devopsProjectStore)
  const formRef = ref<FormInstance>()
  const submitLoading = ref(false)

  const isSuperAdmin = computed(() =>
    (userStore.getUserInfo.roles || []).some((role) => String(role).toUpperCase() === 'SUPER_ADMIN')
  )

  const currentProjectLabel = computed(() => {
    if (!selectedProject.value) return '未选择项目'
    return `${selectedProject.value.name}（${selectedProject.value.code}）`
  })

  const scopeOptions = computed(() => {
    const options = [{ label: '项目级', value: 'project' }]
    if (isSuperAdmin.value) options.push({ label: '系统级', value: 'system' })
    return options
  })

  const resolvedTypeOptions = computed(() =>
    props.typeOptions.length
      ? props.typeOptions
      : [{ label: '用户名密码', value: 'username_password' }]
  )

  const defaultForm = (): DevopsCredentialForm => ({
    name: '',
    code: '',
    credentialType: props.defaultType || resolvedTypeOptions.value[0]?.value || 'username_password',
    channelGroupCode: props.channelGroupCode || '',
    channelType: props.channelType || '',
    username: '',
    password: '',
    token: '',
    privateKey: '',
    passphrase: '',
    kubeconfig: '',
    secretText: '',
    certificate: '',
    jsonData: '',
    description: '',
    status: 1,
    isSystem: false,
    scope: 'project',
    projectId: selectedProjectId.value
  })

  const formData = reactive<DevopsCredentialForm>(defaultForm())

  const rules: FormRules = {
    name: [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
    code: [{ required: true, message: '请输入凭据编码', trigger: 'blur' }],
    scope: [{ required: true, message: '请选择凭据级别', trigger: 'change' }],
    credentialType: [{ required: true, message: '请选择凭据类型', trigger: 'change' }],
    jsonData: [
      {
        validator: (_, value, callback) => {
          if (formData.credentialType !== 'json') {
            callback()
            return
          }
          const text = String(value || '').trim()
          if (!text) {
            callback(new Error('请输入 JSON'))
            return
          }
          try {
            JSON.parse(text)
            callback()
          } catch {
            callback(new Error('请输入合法 JSON'))
          }
        },
        trigger: 'blur'
      }
    ]
  }

  const resetForm = () => {
    Object.assign(formData, defaultForm())
    formRef.value?.clearValidate()
  }

  const closeDialog = () => {
    emit('update:modelValue', false)
  }

  const handleOpen = async () => {
    if (!devopsProjectStore.initialized) {
      await devopsProjectStore.fetchProjects()
    }
    resetForm()
    if (!isSuperAdmin.value && !selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      closeDialog()
    }
  }

  const handleScopeChange = () => {
    if (formData.scope === 'system') {
      if (!isSuperAdmin.value) {
        formData.scope = 'project'
        ElMessage.warning('只有 SUPER_ADMIN 可以创建系统级凭据')
        return
      }
      formData.projectId = ''
      formData.isSystem = true
      formData.channelGroupCode = props.channelGroupCode || ''
      formData.channelType = props.channelType || ''
      return
    }
    formData.scope = 'project'
    formData.projectId = selectedProjectId.value
    formData.isSystem = false
    formData.channelGroupCode = props.channelGroupCode || ''
    formData.channelType = props.channelType || ''
  }

  const prepareScope = () => {
    if (formData.scope === 'system') {
      if (!isSuperAdmin.value) {
        ElMessage.warning('只有 SUPER_ADMIN 可以创建系统级凭据')
        return false
      }
      formData.projectId = ''
      formData.isSystem = true
      formData.channelGroupCode = props.channelGroupCode || ''
      formData.channelType = props.channelType || ''
      return true
    }
    if (!selectedProjectId.value) {
      ElMessage.warning('请选择项目')
      return false
    }
    formData.scope = 'project'
    formData.projectId = selectedProjectId.value
    formData.isSystem = false
    formData.channelGroupCode = props.channelGroupCode || ''
    formData.channelType = props.channelType || ''
    return true
  }

  const handleSubmit = async () => {
    if (!prepareScope()) return
    await formRef.value?.validate()
    if (formData.credentialType === 'json') {
      formData.jsonData = normalizeJsonText(formData.jsonData)
    }
    submitLoading.value = true
    try {
      const result = await createDevopsCredentialApi(formData)
      emit('created', { id: result.id } as DevopsCredential)
      closeDialog()
      ElMessage.success('凭据已创建')
    } finally {
      submitLoading.value = false
    }
  }

  const normalizeJsonText = (content?: string) => {
    const text = String(content || '').trim()
    if (!text) return ''
    JSON.parse(text)
    return text
  }

  watch(
    () => props.defaultType,
    () => {
      if (props.modelValue) {
        formData.credentialType = props.defaultType || resolvedTypeOptions.value[0]?.value || ''
      }
    }
  )
</script>
