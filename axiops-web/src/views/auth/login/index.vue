<template>
  <main class="login-page">
    <AuthTopBar />

    <section class="brand-panel" aria-label="AxiOps">
      <div class="brand-mark">
        <ArtLogo size="52" />
        <ArtWordmark variant="dark" :height="27" />
      </div>

      <div class="brand-copy">
        <p class="brand-kicker">ENTERPRISE OPERATIONS PLATFORM</p>
        <h1>云原生与混合云<br />统一运维控制台</h1>
        <p class="brand-description">稳定交付，精确授权，统一观测。</p>
      </div>

      <div class="brand-footer">
        <span class="status-dot"></span>
        <span>安全认证通道</span>
      </div>
    </section>

    <section class="form-panel">
      <div class="login-form-wrap">
        <div class="mobile-brand">
          <ArtLogo size="44" />
          <ArtWordmark :height="23" />
        </div>

        <header>
          <p class="section-label">ACCOUNT ACCESS</p>
          <h2>登录 AxiOps</h2>
          <p>使用企业账号进入运维工作台</p>
        </header>

        <ElForm
          ref="formRef"
          :model="formData"
          :rules="rules"
          label-position="top"
          @keyup.enter="handleSubmit"
        >
          <ElFormItem label="用户名" prop="username">
            <ElInput
              v-model.trim="formData.username"
              size="large"
              autocomplete="username"
              placeholder="请输入用户名"
            >
              <template #prefix><UserRound :size="18" /></template>
            </ElInput>
          </ElFormItem>

          <ElFormItem label="密码" prop="password">
            <ElInput
              v-model="formData.password"
              size="large"
              type="password"
              autocomplete="current-password"
              show-password
              placeholder="请输入密码"
            >
              <template #prefix><LockKeyhole :size="18" /></template>
            </ElInput>
          </ElFormItem>

          <ElAlert
            v-if="loginError"
            class="login-error"
            :title="loginError"
            type="error"
            :closable="false"
            show-icon
          />

          <ElButton
            class="login-button"
            type="primary"
            size="large"
            :loading="loading"
            @click="handleSubmit"
          >
            <LogIn v-if="!loading" :size="18" />
            <span>{{ loading ? '正在验证' : '登录' }}</span>
          </ElButton>
        </ElForm>

        <p class="security-note"><ShieldCheck :size="16" />账号权限由 AxiOps 权限中台统一管理</p>
      </div>
    </section>
  </main>
</template>

<script setup lang="ts">
  import type { FormInstance, FormRules } from 'element-plus'
  import { ElNotification } from 'element-plus'
  import { LockKeyhole, LogIn, ShieldCheck, UserRound } from 'lucide-vue-next'
  import { loginAxiOpsApi } from '@/api/axiops/auth'
  import { useUserStore } from '@/store/modules/user'
  import { useMenuStore } from '@/store/modules/menu'
  import { usePlatformStore } from '@/store/modules/platform'
  import { useProjectStore } from '@/store/modules/project'

  defineOptions({ name: 'Login' })

  const router = useRouter()
  const route = useRoute()
  const userStore = useUserStore()
  const formRef = ref<FormInstance>()
  const loading = ref(false)
  const loginError = ref('')
  const formData = reactive({ username: '', password: '' })

  const rules: FormRules = {
    username: [{ required: true, message: '请输入用户名', trigger: 'blur' }],
    password: [
      { required: true, message: '请输入密码', trigger: 'blur' },
      { min: 6, message: '密码至少 6 位', trigger: 'blur' }
    ]
  }

  async function handleSubmit() {
    if (!formRef.value || loading.value) return
    const valid = await formRef.value.validate().catch(() => false)
    if (!valid) return

    loading.value = true
    loginError.value = ''
    try {
      const result = await loginAxiOpsApi(formData)
      const user = result.user_info

      // Clear legacy navigation context before establishing the AxiOps session.
      usePlatformStore().clearPlatformState()
      useProjectStore().clearAllStoredData()
      useMenuStore().setMenuList([])

      userStore.setToken(result.access_token, result.refresh_token)
      userStore.setUserInfo({
        userId: user.id,
        userName: user.username,
        nickName: user.nickname,
        avatar: user.avatar,
        email: user.email,
        roles: user.roles || []
      })
      userStore.setLoginStatus(true)
      userStore.checkAndClearWorktabs()

      const redirect = typeof route.query.redirect === 'string' ? route.query.redirect : '/'
      await router.replace(redirect.startsWith('/') ? redirect : '/')
      ElNotification({
        title: '登录成功',
        message: `欢迎回来，${user.nickname || user.username}`,
        type: 'success'
      })
    } catch (error) {
      loginError.value = error instanceof Error ? error.message : '登录失败，请检查账号和密码'
    } finally {
      loading.value = false
    }
  }
</script>

<style scoped>
  .login-page {
    display: grid;
    grid-template-columns: minmax(420px, 0.95fr) minmax(480px, 1.05fr);
    min-height: 100vh;
    background: var(--default-bg-color);
  }

  .brand-panel {
    position: relative;
    display: flex;
    flex-direction: column;
    min-height: 100vh;
    padding: clamp(32px, 5vw, 72px);
    overflow: hidden;
    color: #f8fafc;
    background: #17202a;
  }

  .brand-panel::before {
    position: absolute;
    inset: 0;
    pointer-events: none;
    content: '';
    background-image:
      linear-gradient(rgb(255 255 255 / 5%) 1px, transparent 1px),
      linear-gradient(90deg, rgb(255 255 255 / 5%) 1px, transparent 1px);
    background-size: 40px 40px;
    mask-image: linear-gradient(to bottom, black, transparent 85%);
  }

  .brand-mark,
  .brand-copy,
  .brand-footer {
    position: relative;
    z-index: 1;
  }

  .brand-mark {
    display: flex;
    gap: 14px;
    align-items: center;
    font-size: 24px;
    font-weight: 700;
  }

  .brand-copy {
    margin: auto 0;
  }

  .brand-kicker,
  .section-label {
    margin: 0 0 18px;
    font-size: 12px;
    font-weight: 700;
    letter-spacing: 0;
  }

  .brand-kicker {
    color: #5eead4;
  }

  .brand-copy h1 {
    margin: 0;
    font-size: clamp(36px, 4vw, 58px);
    font-weight: 650;
    line-height: 1.22;
    letter-spacing: 0;
  }

  .brand-description {
    margin: 26px 0 0;
    font-size: 17px;
    color: #cbd5e1;
  }

  .brand-footer,
  .security-note {
    display: flex;
    gap: 9px;
    align-items: center;
  }

  .brand-footer {
    font-size: 13px;
    color: #94a3b8;
  }

  .status-dot {
    width: 8px;
    height: 8px;
    background: #34d399;
    border-radius: 50%;
    box-shadow: 0 0 0 4px rgb(52 211 153 / 14%);
  }

  .form-panel {
    display: grid;
    place-items: center;
    min-height: 100vh;
    padding: 72px clamp(32px, 7vw, 112px);
    background: var(--default-bg-color);
  }

  .login-form-wrap {
    width: min(100%, 420px);
  }

  .mobile-brand {
    display: none;
  }

  header {
    margin-bottom: 34px;
  }

  .section-label {
    margin-bottom: 10px;
    color: var(--el-color-primary);
  }

  header h2 {
    margin: 0;
    font-size: 32px;
    font-weight: 650;
    color: var(--art-gray-900);
    letter-spacing: 0;
  }

  header p:last-child {
    margin: 10px 0 0;
    font-size: 15px;
    color: var(--art-gray-600);
  }

  :deep(.el-form-item) {
    margin-bottom: 22px;
  }

  :deep(.el-form-item__label) {
    font-weight: 600;
    color: var(--art-gray-800);
  }

  :deep(.el-input__wrapper) {
    min-height: 48px;
    border-radius: 6px;
    box-shadow: 0 0 0 1px var(--art-gray-300) inset;
  }

  .login-error {
    margin-bottom: 20px;
    border-radius: 6px;
  }

  .login-button {
    display: flex;
    gap: 8px;
    width: 100%;
    height: 48px;
    font-weight: 600;
    border-radius: 6px;
  }

  .security-note {
    justify-content: center;
    margin: 28px 0 0;
    font-size: 13px;
    color: var(--art-gray-500);
  }

  @media (width <= 900px) {
    .login-page {
      display: block;
    }

    .brand-panel {
      display: none;
    }

    .form-panel {
      min-height: 100vh;
      padding: 88px 24px 40px;
    }

    .mobile-brand {
      display: flex;
      gap: 10px;
      align-items: center;
      margin-bottom: 48px;
      font-size: 22px;
      font-weight: 700;
      color: var(--art-gray-900);
    }
  }
</style>
