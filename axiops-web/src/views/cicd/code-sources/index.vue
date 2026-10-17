<template>
  <div class="code-sources-page">
    <header class="page-header">
      <div>
        <h1>代码源</h1>
        <p>统一管理公司 GitLab、Gitea、GitHub 和 Gitee 连接，项目可复用仓库。</p>
      </div>
      <div class="header-actions">
        <ElButton :icon="RefreshCw" circle title="刷新" :loading="loading" @click="loadSources" />
        <ElButton v-if="canCreate" type="primary" :icon="Plus" @click="openCreate"
          >新增代码源</ElButton
        >
      </div>
    </header>

    <section class="notice">
      <Info :size="17" />
      <span>服务地址支持内网 HTTP；Token 只在服务端加密保存，列表不会回显。</span>
    </section>

    <ElTable v-loading="loading" :data="sources" row-key="id">
      <ElTableColumn label="名称" min-width="180">
        <template #default="{ row }">
          <div class="identity-cell"
            ><strong>{{ row.name }}</strong
            ><small>{{ row.type }}</small></div
          >
        </template>
      </ElTableColumn>
      <ElTableColumn prop="url" label="服务地址" min-width="280" />
      <ElTableColumn label="范围" width="110">
        <template #default><ElTag size="small" effect="plain">平台</ElTag></template>
      </ElTableColumn>
      <ElTableColumn label="状态" width="100">
        <template #default="{ row }"
          ><ElTag size="small" :type="row.status === 'active' ? 'success' : 'info'">{{
            row.status === 'active' ? '启用' : '停用'
          }}</ElTag></template
        >
      </ElTableColumn>
      <ElTableColumn prop="updated_at" label="更新时间" width="180" />
      <ElTableColumn label="操作" width="300" fixed="right">
        <template #default="{ row }">
          <ElButton text type="primary" @click="openEdit(row)">编辑</ElButton>
          <ElButton text type="primary" :loading="checkingId === row.id" @click="check(row)"
            >测试连接</ElButton
          >
          <ElButton text @click="showRepositories(row)">查看仓库</ElButton>
        </template>
      </ElTableColumn>
    </ElTable>
    <ElEmpty v-if="!loading && !sources.length" description="暂无平台代码源" />

    <ElDialog
      v-model="createVisible"
      :title="editingSource ? '编辑平台代码源' : '新增平台代码源'"
      width="600px"
      destroy-on-close
    >
      <ElForm ref="formRef" :model="form" :rules="rules" label-position="top">
        <div class="form-grid">
          <ElFormItem label="名称" prop="name"
            ><ElInput v-model="form.name" placeholder="公司 Gitea"
          /></ElFormItem>
          <ElFormItem label="类型" prop="type">
            <ElSelect v-model="form.type" style="width: 100%">
              <ElOption label="GitLab" value="gitlab" /><ElOption label="Gitea" value="gitea" />
              <ElOption label="GitHub" value="github" /><ElOption label="Gitee" value="gitee" />
            </ElSelect>
          </ElFormItem>
        </div>
        <ElFormItem label="服务地址" prop="url"
          ><ElInput v-model="form.url" placeholder="http://gitea.intra.example.com"
        /></ElFormItem>
        <ElFormItem v-if="editingSource" label="状态" prop="status">
          <ElSelect v-model="form.status" style="width: 100%">
            <ElOption label="启用" value="active" />
            <ElOption label="停用" value="inactive" />
          </ElSelect>
        </ElFormItem>
        <ElDivider content-position="left">访问凭据</ElDivider>
        <ElFormItem label="Token 凭据">
          <ElRadioGroup v-model="credentialMode">
            <ElRadio value="existing">选择已有凭据</ElRadio><ElRadio value="new">新建凭据</ElRadio>
          </ElRadioGroup>
        </ElFormItem>
        <ElFormItem
          v-if="credentialMode === 'existing'"
          label="Git Token 凭据"
          prop="credential_id"
        >
          <ElSelect
            v-model="form.credential_id"
            filterable
            style="width: 100%"
            placeholder="选择平台凭据"
          >
            <ElOption
              v-for="credential in credentials"
              :key="credential.id"
              :label="credential.name"
              :value="credential.id"
            />
          </ElSelect>
        </ElFormItem>
        <template v-else>
          <ElFormItem label="凭据名称" prop="credential_name"
            ><ElInput v-model="form.credential_name"
          /></ElFormItem>
          <ElFormItem label="Personal Access Token" prop="token"
            ><ElInput
              v-model="form.token"
              type="password"
              show-password
              autocomplete="new-password"
          /></ElFormItem>
        </template>
      </ElForm>
      <template #footer
        ><ElButton @click="createVisible = false">取消</ElButton
        ><ElButton type="primary" :loading="submitting" @click="submit">{{
          editingSource ? '保存' : '创建'
        }}</ElButton></template
      >
    </ElDialog>

    <ElDialog
      v-model="repositoriesVisible"
      :title="`${selectedSource?.name || ''} 仓库`"
      width="720px"
      destroy-on-close
    >
      <ElTable v-loading="repositoriesLoading" :data="repositories">
        <ElTableColumn prop="full_name" label="仓库" min-width="260" />
        <ElTableColumn prop="default_branch" label="默认分支" width="140" />
        <ElTableColumn label="可见性" width="110"
          ><template #default="{ row }">{{
            row.private ? '私有' : '公开'
          }}</template></ElTableColumn
        >
        <ElTableColumn prop="clone_url" label="克隆地址" min-width="280" />
      </ElTable>
      <ElEmpty v-if="!repositoriesLoading && !repositories.length" description="未获取到仓库" />
    </ElDialog>
  </div>
</template>

<script setup lang="ts">
  import { computed, onMounted, reactive, ref } from 'vue'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import { Info, Plus, RefreshCw } from 'lucide-vue-next'
  import {
    checkCodeSourceApi,
    createCredentialApi,
    createGlobalCodeSourceApi,
    listCodeSourcesApi,
    listCredentialsApi,
    listRepositoriesApi,
    updateCodeSourceApi,
    type AxiOpsCodeSource,
    type AxiOpsCredential,
    type AxiOpsRepository
  } from '@/api/axiops'
  import { useAxiOpsAccess } from '@/composables/useAxiOpsAccess'

  defineOptions({ name: 'CICDCodeSources' })
  const access = useAxiOpsAccess()
  const loading = ref(false)
  const submitting = ref(false)
  const checkingId = ref<number | null>(null)
  const createVisible = ref(false)
  const repositoriesVisible = ref(false)
  const repositoriesLoading = ref(false)
  const formRef = ref<FormInstance>()
  const sources = ref<AxiOpsCodeSource[]>([])
  const credentials = ref<AxiOpsCredential[]>([])
  const repositories = ref<AxiOpsRepository[]>([])
  const selectedSource = ref<AxiOpsCodeSource>()
  const editingSource = ref<AxiOpsCodeSource>()
  const credentialMode = ref<'existing' | 'new'>('existing')
  const form = reactive({
    name: '',
    type: 'gitea',
    url: '',
    credential_id: 0,
    credential_name: '',
    token: '',
    status: 'active'
  })
  const rules: FormRules = {
    name: [{ required: true, message: '请输入名称', trigger: 'blur' }],
    type: [{ required: true, message: '请选择类型', trigger: 'change' }],
    url: [{ required: true, message: '请输入服务地址', trigger: 'blur' }],
    credential_id: [{ required: true, message: '请选择凭据', trigger: 'change' }],
    credential_name: [{ required: true, message: '请输入凭据名称', trigger: 'blur' }],
    token: [{ required: true, message: '请输入 Token', trigger: 'blur' }]
  }
  const canCreate = computed(() => access.can('code_source', 'create'))

  const loadSources = async () => {
    loading.value = true
    try {
      sources.value = (await listCodeSourcesApi(0)) || []
      credentials.value = (await listCredentialsApi({ type: 'git_token', project_id: 0 })) || []
    } finally {
      loading.value = false
    }
  }
  const openCreate = () => {
    editingSource.value = undefined
    Object.assign(form, {
      name: '',
      type: 'gitea',
      url: '',
      credential_id: 0,
      credential_name: '',
      token: '',
      status: 'active'
    })
    credentialMode.value = credentials.value.length ? 'existing' : 'new'
    createVisible.value = true
  }
  const openEdit = (source: AxiOpsCodeSource) => {
    editingSource.value = source
    Object.assign(form, {
      name: source.name,
      type: source.type,
      url: source.url,
      credential_id: source.credential_id,
      credential_name: '',
      token: '',
      status: source.status || 'active'
    })
    credentialMode.value = 'existing'
    createVisible.value = true
  }
  const submit = async () => {
    if (!(await formRef.value?.validate())) return
    let credentialID = form.credential_id
    submitting.value = true
    try {
      if (credentialMode.value === 'new') {
        credentialID = (
          await createCredentialApi({
            name: form.credential_name.trim(),
            type: 'git_token',
            description: '平台代码源访问凭据',
            data: JSON.stringify({ token: form.token }),
            project_id: 0
          })
        ).id
      }
      if (!credentialID) return ElMessage.warning('请选择或创建凭据')
      const parsed = new URL(form.url.trim())
      if (!['http:', 'https:'].includes(parsed.protocol))
        return ElMessage.warning('服务地址必须使用 HTTP 或 HTTPS')
      const sourceData = {
        name: form.name.trim(),
        type: form.type,
        url: form.url.trim().replace(/\/$/, ''),
        credential_id: credentialID,
        status: form.status as 'active' | 'inactive'
      }
      if (editingSource.value) {
        await updateCodeSourceApi(editingSource.value.id, sourceData)
        ElMessage.success('代码源已更新')
      } else {
        await createGlobalCodeSourceApi(sourceData)
        ElMessage.success('代码源已创建')
      }
      createVisible.value = false
      editingSource.value = undefined
      await loadSources()
    } catch (error: any) {
      // Preserve the API error so an operator can distinguish permission,
      // encryption, database and provider failures from a generic 500.
      ElMessage.error(error?.message || '代码源创建失败，请检查地址和凭据')
    } finally {
      submitting.value = false
    }
  }
  const check = async (source: AxiOpsCodeSource) => {
    checkingId.value = source.id
    try {
      await checkCodeSourceApi(source.id)
      ElMessage.success('平台服务连接代码源成功；构建集群还需能访问仓库地址')
    } finally {
      checkingId.value = null
    }
  }
  const showRepositories = async (source: AxiOpsCodeSource) => {
    selectedSource.value = source
    repositories.value = []
    repositoriesVisible.value = true
    repositoriesLoading.value = true
    try {
      repositories.value = (await listRepositoriesApi(source.id)) || []
    } finally {
      repositoriesLoading.value = false
    }
  }
  onMounted(async () => {
    await access.load()
    await loadSources()
  })
</script>

<style scoped lang="scss">
  .code-sources-page {
    min-height: 100%;
    padding: 20px;
    background: var(--art-main-bg-color);
    color: var(--art-text-gray-900);
  }
  .page-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 16px;
    margin-bottom: 18px;
  }
  .page-header h1 {
    margin: 0;
    font-size: 24px;
    letter-spacing: 0;
  }
  .page-header p {
    margin: 4px 0 0;
    color: var(--art-text-gray-600);
    font-size: 13px;
  }
  .header-actions {
    display: flex;
    gap: 8px;
  }
  .notice {
    display: flex;
    align-items: center;
    gap: 8px;
    margin-bottom: 14px;
    padding: 10px 12px;
    color: var(--art-text-gray-600);
    background: var(--art-bg-color);
    border: 1px solid var(--art-border-color);
    border-radius: 6px;
    font-size: 13px;
  }
  .identity-cell {
    display: flex;
    flex-direction: column;
    gap: 3px;
  }
  .identity-cell small {
    color: var(--art-text-gray-600);
  }
  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 14px;
  }
  @media (max-width: 720px) {
    .page-header {
      align-items: flex-start;
      flex-direction: column;
    }
    .form-grid {
      grid-template-columns: 1fr;
    }
  }
</style>
