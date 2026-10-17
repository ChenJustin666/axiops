<template>
  <section v-loading="loading">
    <div class="registry-heading">
      <div
        ><h2>CI/CD 镜像仓库</h2
        ><p>服务选择仓库后，构建会使用对应凭据拉取构建器并推送产物镜像。</p></div
      >
      <ElButton type="primary" @click="edit()">添加仓库</ElButton>
    </div>
    <ElTable :data="registries" empty-text="尚未配置 CI/CD 镜像仓库">
      <ElTableColumn prop="name" label="名称" min-width="150" />
      <ElTableColumn prop="url" label="仓库地址 / 空间" min-width="300" show-overflow-tooltip />
      <ElTableColumn label="可选命名空间" min-width="180">
        <template #default="{ row }">{{ namespaceChoices(row).join('、') || '按需填写' }}</template>
      </ElTableColumn>
      <ElTableColumn label="凭据" min-width="130"
        ><template #default="{ row }">{{
          credentials.find((c) => c.id === row.credential_id)?.name || `#${row.credential_id}`
        }}</template></ElTableColumn
      >
      <ElTableColumn label="状态" width="90"
        ><template #default="{ row }"
          ><ElTag :type="row.status === 'active' ? 'success' : 'info'">{{
            row.status === 'active' ? '启用' : '停用'
          }}</ElTag></template
        ></ElTableColumn
      >
      <ElTableColumn label="操作" width="90"
        ><template #default="{ row }"
          ><ElButton text type="primary" @click="edit(row)">编辑</ElButton></template
        ></ElTableColumn
      >
    </ElTable>
    <ElDialog v-model="visible" title="镜像仓库配置" width="620px" append-to-body destroy-on-close>
      <ElForm label-position="top">
        <ElFormItem label="名称" required
          ><ElInput v-model="form.name" maxlength="64"
        /></ElFormItem>
        <ElFormItem label="类型"
          ><ElSelect v-model="form.type"
            ><ElOption label="阿里云 ACR" value="acr" /><ElOption
              label="Harbor"
              value="harbor" /><ElOption label="Docker Registry" value="registry" /><ElOption
              label="Docker Hub"
              value="dockerhub" /><ElOption label="Amazon ECR" value="ecr" /></ElSelect
        ></ElFormItem>
        <ElFormItem label="仓库地址 / 空间" required
          ><ElInput v-model="form.url" placeholder="https://registry.example.com/team"
        /></ElFormItem>
        <ElFormItem label="可选命名空间 / Harbor 项目">
          <ElSelect
            v-model="form.namespaces"
            multiple
            filterable
            allow-create
            default-first-option
            placeholder="输入已有命名空间后按回车，可添加多个"
            style="width: 100%"
          >
            <ElOption
              v-for="name in namespaceChoices(form)"
              :key="name"
              :label="name"
              :value="name"
            />
          </ElSelect>
          <div class="namespace-hint">
            填写仓库中已存在的命名空间，如
            sinozo。地址中的空间保留为默认选项；仅配置域名时，首个空间为默认值。不会自动在远端创建空间。
          </div>
        </ElFormItem>
        <ElFormItem label="认证凭据" required>
          <ElSelect v-model="form.credential_id" placeholder="选择镜像仓库凭据" style="width: 100%"
            ><ElOption
              v-for="credential in credentials"
              :key="credential.id"
              :label="credential.name"
              :value="credential.id"
          /></ElSelect>
        </ElFormItem>
        <ElCheckbox v-model="newCredential">新建账号密码凭据</ElCheckbox>
        <template v-if="newCredential">
          <ElFormItem label="仓库用户名" required
            ><ElInput v-model="username" autocomplete="off"
          /></ElFormItem>
          <ElFormItem label="仓库密码 / 访问令牌" required
            ><ElInput v-model="password" type="password" show-password autocomplete="new-password"
          /></ElFormItem>
        </template>
        <ElFormItem label="启用"
          ><ElSwitch v-model="form.status" active-value="active" inactive-value="inactive"
        /></ElFormItem>
      </ElForm>
      <template #footer
        ><ElButton @click="visible = false">取消</ElButton
        ><ElButton :loading="saving" @click="testConnection">测试连接</ElButton
        ><ElButton type="primary" :loading="saving" @click="save">保存</ElButton></template
      >
    </ElDialog>
  </section>
</template>

<script setup lang="ts">
  import { onMounted, reactive, ref, watch } from 'vue'
  import { ElMessage } from 'element-plus'
  import {
    createCredentialApi,
    deleteCredentialApi,
    listCredentialsApi,
    type AxiOpsCredential
  } from '@/api/axiops'
  import {
    listBuildRegistriesApi,
    saveBuildRegistryApi,
    testBuildRegistryApi,
    type BuildRegistry
  } from '@/api/axiops/build-registry'

  const registries = ref<BuildRegistry[]>([])
  const credentials = ref<AxiOpsCredential[]>([])
  const loading = ref(false),
    saving = ref(false),
    visible = ref(false)
  const newCredential = ref(false),
    username = ref(''),
    password = ref('')
  const empty = (): BuildRegistry => ({
    id: 0,
    name: '',
    type: 'acr',
    url: '',
    namespaces: [],
    credential_id: 0,
    project_id: 0,
    status: 'active'
  })
  const form = reactive<BuildRegistry>(empty())
  const namespaceChoices = (registry: BuildRegistry) => {
    const path = registry.url
      .replace(/^https?:\/\//, '')
      .split('/')
      .slice(1)
      .join('/')
      .replace(/\/+$/, '')
    return [...new Set([path, ...(registry.namespaces || [])].filter(Boolean))]
  }
  async function load() {
    loading.value = true
    try {
      const [items, creds] = await Promise.all([
        listBuildRegistriesApi(),
        listCredentialsApi({ project_id: 0 })
      ])
      registries.value = items || []
      credentials.value = (creds || []).filter(
        (c) => c.project_id === 0 && ['harbor_auth', 'registry_auth'].includes(c.type)
      )
    } finally {
      loading.value = false
    }
  }
  function edit(row?: BuildRegistry) {
    Object.assign(form, row ? { ...row, namespaces: namespaceChoices(row) } : empty())
    newCredential.value = false
    visible.value = true
  }
  async function testConnection() {
    if (!form.url.trim()) return void ElMessage.warning('请填写仓库地址')
    if (newCredential.value)
      return void ElMessage.info('新建凭据请直接点击保存，系统会先验证再保存')
    if (!form.credential_id) return void ElMessage.warning('请选择镜像仓库凭据')
    saving.value = true
    try {
      await testBuildRegistryApi({
        url: form.url,
        type: form.type,
        namespaces: form.namespaces,
        credential_id: form.credential_id
      })
      ElMessage.success('镜像仓库连接验证通过')
    } catch (error) {
      console.error('镜像仓库验证失败', error)
    } finally {
      saving.value = false
    }
  }
  async function save() {
    if (!form.name.trim() || !form.url.trim()) return void ElMessage.warning('请填写名称和仓库地址')
    if (newCredential.value ? !username.value.trim() || !password.value : !form.credential_id)
      return void ElMessage.warning('请选择凭据或填写完整的仓库账号密码')
    saving.value = true
    let createdCredentialID = 0
    try {
      if (newCredential.value) {
        const result = await createCredentialApi({
          name: `${form.name.trim()} 推送凭据`,
          type: 'registry_auth',
          description: 'CI/CD 镜像拉取与推送',
          project_id: 0,
          data: JSON.stringify({ username: username.value.trim(), password: password.value })
        })
        form.credential_id = result.id
        createdCredentialID = result.id
        newCredential.value = false
        password.value = ''
        credentials.value = (await listCredentialsApi({ project_id: 0 })).filter(
          (c) => c.project_id === 0 && ['harbor_auth', 'registry_auth'].includes(c.type)
        )
      }
      // Saving is intentionally gated by the same remote check as the
      // standalone button. An invalid credential never reaches image_registry.
      await testBuildRegistryApi({
        url: form.url,
        type: form.type,
        namespaces: form.namespaces,
        credential_id: form.credential_id
      })
      await saveBuildRegistryApi({ ...form })
      visible.value = false
      ElMessage.success('连接验证通过，镜像仓库已保存')
      await load()
    } catch (error) {
      if (createdCredentialID > 0) {
        await deleteCredentialApi(createdCredentialID).catch(() => undefined)
        form.credential_id = 0
        newCredential.value = true
      }
      // Keep the dialog and form values so the user can correct the address or
      // rotate the credential without re-entering the entire configuration.
      console.error('镜像仓库验证失败', error)
    } finally {
      saving.value = false
    }
  }
  watch(visible, (value) => {
    if (!value) {
      username.value = ''
      password.value = ''
    }
  })
  onMounted(load)
</script>

<style scoped>
  .namespace-hint {
    margin-top: 6px;
    color: var(--el-text-color-secondary);
    font-size: 12px;
    line-height: 1.5;
  }
  .registry-heading {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: 16px;
    margin-bottom: 20px;
  }
  .registry-heading h2 {
    margin: 0 0 8px;
    font-size: 20px;
  }
  .registry-heading p {
    margin: 0;
    color: var(--el-text-color-secondary);
  }
</style>
