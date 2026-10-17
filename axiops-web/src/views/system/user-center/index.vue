<template>
  <div class="user-center-container">
    <!-- 用户信息头部 -->
    <div class="user-header">
      <div class="header-content">
        <div class="avatar-section">
          <div class="avatar-wrapper">
            <img :src="userInfo.avatar || defaultAvatar" alt="avatar" class="user-avatar" />
            <div class="avatar-upload" @click="handleAvatarClick">
              <el-icon :size="16"><Camera /></el-icon>
            </div>
            <input
              ref="avatarInput"
              type="file"
              accept="image/*"
              style="display: none"
              @change="handleAvatarChange"
            />
          </div>
          <div class="user-info">
            <h2>{{ userInfo.nickname || userInfo.username || '用户' }}</h2>
            <p>{{ userInfo.email || '暂未设置邮箱' }}</p>
            <div class="user-meta">
              <span v-if="userInfo.workNumber">
                <el-icon><Ticket /></el-icon>
                工号: {{ userInfo.workNumber }}
              </span>
              <span v-if="userInfo.deptNames">
                <el-icon><OfficeBuilding /></el-icon>
                {{ userInfo.deptNames }}
              </span>
            </div>
          </div>
        </div>
        <div class="header-stats">
          <div class="stat-item">
            <div class="stat-value">{{ daysSinceJoined }}</div>
            <div class="stat-label">加入天数</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ userRoles.length }}</div>
            <div class="stat-label">角色数量</div>
          </div>
          <div class="stat-item">
            <div class="stat-value">{{ lastLoginTime }}</div>
            <div class="stat-label">最后登录</div>
          </div>
        </div>
      </div>
    </div>

    <!-- 标签页内容 -->
    <div class="user-content">
      <ElTabs v-model="activeTab" class="user-tabs">
        <ElTabPane label="基本信息" name="basic">
          <div class="tab-content">
            <ElForm
              ref="basicFormRef"
              :model="basicForm"
              :rules="basicRules"
              label-width="100px"
              class="user-form"
            >
              <ElRow :gutter="20">
                <ElCol :span="12">
                  <ElFormItem label="用户名">
                    <ElInput v-model="basicForm.username" disabled />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="昵称" prop="nickname">
                    <ElInput
                      v-model="basicForm.nickname"
                      placeholder="请输入昵称"
                      maxlength="20"
                      show-word-limit
                    />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElRow :gutter="20">
                <ElCol :span="12">
                  <ElFormItem label="邮箱" prop="email">
                    <ElInput v-model="basicForm.email" placeholder="请输入邮箱" />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="手机号" prop="phone">
                    <ElInput v-model="basicForm.phone" placeholder="请输入手机号" maxlength="11" />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElRow :gutter="20">
                <ElCol :span="12">
                  <ElFormItem label="工号">
                    <ElInput v-model="basicForm.workNumber" disabled />
                  </ElFormItem>
                </ElCol>
                <ElCol :span="12">
                  <ElFormItem label="部门">
                    <ElInput v-model="basicForm.deptNames" disabled />
                  </ElFormItem>
                </ElCol>
              </ElRow>

              <ElDivider content-position="left">IM 账号绑定</ElDivider>

              <div class="im-accounts-section">
                <div class="im-account-card">
                  <div class="im-account-content">
                    <div class="im-account-label">钉钉</div>
                    <div class="im-account-value">
                      {{ basicForm.dingtalkId || '未绑定' }}
                    </div>
                  </div>
                  <div class="im-account-status" :class="{ bound: basicForm.dingtalkId }">
                    {{ basicForm.dingtalkId ? '已绑定' : '未绑定' }}
                  </div>
                </div>

                <div class="im-account-card">
                  <div class="im-account-content">
                    <div class="im-account-label">企业微信</div>
                    <div class="im-account-value">
                      {{ basicForm.wechatId || '未绑定' }}
                    </div>
                  </div>
                  <div class="im-account-status" :class="{ bound: basicForm.wechatId }">
                    {{ basicForm.wechatId ? '已绑定' : '未绑定' }}
                  </div>
                </div>

                <div class="im-account-card">
                  <div class="im-account-content">
                    <div class="im-account-label">飞书</div>
                    <div class="im-account-value">
                      {{ basicForm.feishuId || '未绑定' }}
                    </div>
                  </div>
                  <div class="im-account-status" :class="{ bound: basicForm.feishuId }">
                    {{ basicForm.feishuId ? '已绑定' : '未绑定' }}
                  </div>
                </div>
              </div>

              <ElFormItem>
                <ElButton type="primary" @click="handleSaveBasicInfo" :loading="basicLoading">
                  保存信息
                </ElButton>
                <ElButton @click="resetBasicForm" :disabled="basicLoading">重置</ElButton>
              </ElFormItem>
            </ElForm>
          </div>
        </ElTabPane>

        <ElTabPane label="修改密码" name="password">
          <div class="tab-content">
            <ElForm
              ref="passwordFormRef"
              :model="passwordForm"
              :rules="passwordRules"
              label-width="120px"
              class="user-form password-form"
            >
              <ElFormItem label="当前密码" prop="oldPassword">
                <ElInput
                  v-model="passwordForm.oldPassword"
                  type="password"
                  placeholder="请输入当前密码"
                  show-password
                  clearable
                />
              </ElFormItem>

              <ElFormItem label="新密码" prop="newPassword">
                <ElInput
                  v-model="passwordForm.newPassword"
                  type="password"
                  placeholder="请输入新密码(至少6位)"
                  show-password
                  clearable
                />
                <div v-if="passwordForm.newPassword" class="password-strength">
                  <span class="strength-label">密码强度：</span>
                  <div class="strength-bar">
                    <div
                      class="strength-level"
                      :class="passwordStrength"
                      :style="{ width: passwordStrengthWidth }"
                    />
                  </div>
                  <span class="strength-text" :class="passwordStrength">{{
                    passwordStrengthText
                  }}</span>
                </div>
              </ElFormItem>

              <ElFormItem label="确认密码" prop="confirmPassword">
                <ElInput
                  v-model="passwordForm.confirmPassword"
                  type="password"
                  placeholder="请再次输入新密码"
                  show-password
                  clearable
                />
              </ElFormItem>

              <ElFormItem>
                <ElButton type="primary" @click="handleChangePassword" :loading="passwordLoading">
                  修改密码
                </ElButton>
                <ElButton @click="resetPasswordForm" :disabled="passwordLoading">重置</ElButton>
              </ElFormItem>
            </ElForm>
          </div>
        </ElTabPane>

        <ElTabPane label="角色权限" name="roles">
          <div class="tab-content roles-content">
            <div v-if="userRoles.length > 0" class="roles-list">
              <div v-for="role in userRoles" :key="role" class="role-item">
                <span>{{ role }}</span>
              </div>
            </div>
            <ElEmpty v-else description="暂无角色分配" />
          </div>
        </ElTabPane>
      </ElTabs>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, reactive, computed, onMounted } from 'vue'
  import { ElMessage, type FormInstance, type FormRules } from 'element-plus'
  import { useRouter } from 'vue-router'
  import {
    Camera,
    Ticket,
    OfficeBuilding
  } from '@element-plus/icons-vue'
  import {
    getUserInfoApi,
    updateUserInfoApi,
    updateUserPasswordApi,
    updateUserAvatarApi,
    uploadImageApi,
    type UserSysUserInfo
  } from '@/api/portal'

  defineOptions({ name: 'UserCenter' })

  const router = useRouter()

  // 默认头像
  const defaultAvatar = '/imgs/kube-nova.png'

  // 状态
  const activeTab = ref('basic')
  const basicLoading = ref(false)
  const passwordLoading = ref(false)
  const avatarInput = ref<HTMLInputElement>()

  // 用户信息
  const userInfo = ref<UserSysUserInfo>({
    id: 0,
    username: '',
    nickname: '',
    avatar: '',
    email: '',
    phone: '',
    workNumber: '',
    status: 1,
    isNeedResetPwd: 0,
    createdBy: '',
    updatedBy: '',
    createdAt: 0,
    updatedAt: 0,
    deptNames: '',
    roleNames: '',
    dingtalkId: '',
    wechatId: '',
    feishuId: ''
  })

  // 用户角色列表
  const userRoles = computed(() => {
    const roles = userInfo.value.roleNames
    if (!roles) return []
    return roles
      .split(',')
      .map((r) => r.trim())
      .filter(Boolean)
  })

  // 表单引用
  const basicFormRef = ref<FormInstance>()
  const passwordFormRef = ref<FormInstance>()

  // 基本信息表单
  const basicForm = reactive({
    username: '',
    nickname: '',
    email: '',
    phone: '',
    workNumber: '',
    deptNames: '',
    dingtalkId: '',
    wechatId: '',
    feishuId: ''
  })

  // 密码表单
  const passwordForm = reactive({
    oldPassword: '',
    newPassword: '',
    confirmPassword: ''
  })

  // 验证规则
  const basicRules: FormRules = {
    nickname: [
      { required: true, message: '请输入昵称', trigger: 'blur' },
      { min: 2, max: 20, message: '昵称长度在 2 到 20 个字符', trigger: 'blur' }
    ],
    email: [
      { required: true, message: '请输入邮箱', trigger: 'blur' },
      { type: 'email', message: '请输入正确的邮箱格式', trigger: ['blur', 'change'] }
    ],
    phone: [
      { required: true, message: '请输入手机号', trigger: 'blur' },
      { pattern: /^1[3-9]\d{9}$/, message: '请输入正确的手机号', trigger: ['blur', 'change'] }
    ]
  }

  const validatePassword = (_rule: any, value: any, callback: any) => {
    if (value === '') {
      callback(new Error('请输入新密码'))
    } else if (value.length < 6) {
      callback(new Error('密码长度不能少于6位'))
    } else if (value === passwordForm.oldPassword) {
      callback(new Error('新密码不能与当前密码相同'))
    } else {
      if (passwordForm.confirmPassword !== '') {
        passwordFormRef.value?.validateField('confirmPassword')
      }
      callback()
    }
  }

  const validateConfirmPassword = (_rule: any, value: any, callback: any) => {
    if (value === '') {
      callback(new Error('请再次输入新密码'))
    } else if (value !== passwordForm.newPassword) {
      callback(new Error('两次输入密码不一致'))
    } else {
      callback()
    }
  }

  const passwordRules: FormRules = {
    oldPassword: [
      { required: true, message: '请输入当前密码', trigger: 'blur' },
      { min: 6, message: '密码长度不能少于6位', trigger: 'blur' }
    ],
    newPassword: [{ validator: validatePassword, trigger: ['blur', 'change'] }],
    confirmPassword: [{ validator: validateConfirmPassword, trigger: ['blur', 'change'] }]
  }

  // 计算属性
  const daysSinceJoined = computed(() => {
    const createdAt = userInfo.value.createdAt
    if (!createdAt) return 0
    const timestamp = createdAt > 9999999999 ? createdAt : createdAt * 1000
    const joinDate = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - joinDate.getTime()
    return Math.floor(diff / (1000 * 60 * 60 * 24))
  })

  const lastLoginTime = computed(() => {
    const updatedAt = userInfo.value.updatedAt
    if (!updatedAt) return '未知'
    const timestamp = updatedAt > 9999999999 ? updatedAt : updatedAt * 1000
    const lastTime = new Date(timestamp)
    const now = new Date()
    const diff = now.getTime() - lastTime.getTime()

    const minutes = Math.floor(diff / (1000 * 60))
    const hours = Math.floor(diff / (1000 * 60 * 60))
    const days = Math.floor(diff / (1000 * 60 * 60 * 24))

    if (minutes < 1) return '刚刚'
    if (minutes < 60) return `${minutes}分钟前`
    if (hours < 24) return `${hours}小时前`
    if (days < 7) return `${days}天前`

    return lastTime.toLocaleDateString()
  })

  // 密码强度计算
  const passwordStrength = computed(() => {
    const pwd = passwordForm.newPassword
    if (!pwd) return ''

    let strength = 0
    if (pwd.length >= 6) strength++
    if (pwd.length >= 10) strength++
    if (/\d/.test(pwd)) strength++
    if (/[a-z]/.test(pwd)) strength++
    if (/[A-Z]/.test(pwd)) strength++
    if (/[!@#$%^&*(),.?":{}|<>]/.test(pwd)) strength++

    if (strength <= 2) return 'weak'
    if (strength <= 4) return 'medium'
    return 'strong'
  })

  const passwordStrengthWidth = computed(() => {
    const strength = passwordStrength.value
    if (strength === 'weak') return '33%'
    if (strength === 'medium') return '66%'
    if (strength === 'strong') return '100%'
    return '0%'
  })

  const passwordStrengthText = computed(() => {
    const strength = passwordStrength.value
    if (strength === 'weak') return '弱'
    if (strength === 'medium') return '中'
    if (strength === 'strong') return '强'
    return ''
  })

  // 获取用户信息
  const fetchUserInfo = async () => {
    try {
      const data = await getUserInfoApi()
      if (!data) throw new Error('未获取到用户数据')

      userInfo.value = data
      Object.assign(basicForm, {
        username: data.username || '',
        nickname: data.nickname || '',
        email: data.email || '',
        phone: data.phone || '',
        workNumber: data.workNumber || '',
        deptNames: data.deptNames || '',
        dingtalkId: data.dingtalkId || '',
        wechatId: data.wechatId || '',
        feishuId: data.feishuId || ''
      })
    } catch (error: any) {
      console.error('获取用户信息失败:', error)
    }
  }

  // 保存基本信息
  const handleSaveBasicInfo = async () => {
    try {
      const valid = await basicFormRef.value?.validate()
      if (!valid) return

      basicLoading.value = true
      await updateUserInfoApi({
        nickname: basicForm.nickname,
        email: basicForm.email,
        phone: basicForm.phone
      })

      await fetchUserInfo()
      ElMessage.success('保存成功')
    } finally {
      basicLoading.value = false
    }
  }

  // 退出登录
  const handleLogout = () => {
    localStorage.removeItem('token')
    localStorage.removeItem('refreshToken')
    localStorage.removeItem('userInfo')
    sessionStorage.clear()
    router.push('/login')
  }

  // 修改密码
  const handleChangePassword = async () => {
    try {
      const valid = await passwordFormRef.value?.validate()
      if (!valid) return

      passwordLoading.value = true
      await updateUserPasswordApi({
        username: basicForm.username,
        oldPassword: passwordForm.oldPassword,
        newPassword: passwordForm.newPassword,
        confirmPassword: passwordForm.confirmPassword
      })

      ElMessage.success('密码修改成功，请重新登录')
      resetPasswordForm()
      setTimeout(() => handleLogout(), 1500)
    } finally {
      passwordLoading.value = false
    }
  }

  // 头像相关
  const handleAvatarClick = () => {
    avatarInput.value?.click()
  }

  const handleAvatarChange = async (e: Event) => {
    const input = e.target as HTMLInputElement
    const file = input.files?.[0]
    if (!file) return

    if (!file.type.startsWith('image/')) {
      ElMessage.error('请选择图片文件')
      return
    }

    if (file.size > 5 * 1024 * 1024) {
      ElMessage.error('图片大小不能超过 5MB')
      return
    }

    try {
      const uploadResponse = await uploadImageApi({ project: 'user' }, file)
      const imageUrl = uploadResponse?.imageUri
      if (!imageUrl) throw new Error('上传成功但未返回图片地址')

      await updateUserAvatarApi({ avatar: imageUrl })
      await fetchUserInfo()
      ElMessage.success('头像更新成功')
    } finally {
      if (input) input.value = ''
    }
  }

  // 重置表单
  const resetBasicForm = () => {
    Object.assign(basicForm, {
      username: userInfo.value.username || '',
      nickname: userInfo.value.nickname || '',
      email: userInfo.value.email || '',
      phone: userInfo.value.phone || '',
      workNumber: userInfo.value.workNumber || '',
      deptNames: userInfo.value.deptNames || '',
      dingtalkId: userInfo.value.dingtalkId || '',
      wechatId: userInfo.value.wechatId || '',
      feishuId: userInfo.value.feishuId || ''
    })
    basicFormRef.value?.clearValidate()
  }

  const resetPasswordForm = () => {
    passwordForm.oldPassword = ''
    passwordForm.newPassword = ''
    passwordForm.confirmPassword = ''
    passwordFormRef.value?.clearValidate()
  }

  // 生命周期
  onMounted(async () => {
    await fetchUserInfo()
  })
</script>

<style lang="scss" scoped>
  .user-center-container {
    height: 100%;
    background: #ffffff;
    border-radius: 8px;
    overflow: hidden;
    display: flex;
    flex-direction: column;

    .user-header {
      background: linear-gradient(135deg, #409eff 0%, #337ecc 100%);
      padding: 32px 40px;
      color: white;

      .header-content {
        max-width: 1000px;
        margin: 0 auto;
        display: flex;
        justify-content: space-between;
        align-items: center;
        flex-wrap: wrap;
        gap: 24px;

        .avatar-section {
          display: flex;
          align-items: center;
          gap: 20px;

          .avatar-wrapper {
            position: relative;
            flex-shrink: 0;

            .user-avatar {
              width: 80px;
              height: 80px;
              border-radius: 50%;
              border: 3px solid rgba(255, 255, 255, 0.25);
              object-fit: cover;
              background: rgba(255, 255, 255, 0.15);
            }

            .avatar-upload {
              position: absolute;
              bottom: 0;
              right: 0;
              width: 28px;
              height: 28px;
              background: rgba(255, 255, 255, 0.2);
              border-radius: 50%;
              display: flex;
              align-items: center;
              justify-content: center;
              cursor: pointer;
              transition: background 0.2s;
              border: 2px solid rgba(255, 255, 255, 0.5);

              &:hover {
                background: rgba(255, 255, 255, 0.3);
              }
            }
          }

          .user-info {
            h2 {
              font-size: 22px;
              margin: 0 0 6px 0;
              font-weight: 600;
            }

            p {
              opacity: 0.95;
              margin: 0 0 10px 0;
              font-size: 14px;
            }

            .user-meta {
              display: flex;
              gap: 16px;
              font-size: 13px;
              opacity: 0.9;
              flex-wrap: wrap;

              span {
                display: flex;
                align-items: center;
                gap: 4px;

                .el-icon {
                  font-size: 14px;
                }
              }
            }
          }
        }

        .header-stats {
          display: flex;
          gap: 32px;

          .stat-item {
            text-align: center;
            padding: 12px 20px;
            background: rgba(255, 255, 255, 0.15);
            border-radius: 8px;
            min-width: 80px;

            .stat-value {
              font-size: 24px;
              font-weight: 600;
              margin-bottom: 4px;
            }

            .stat-label {
              font-size: 12px;
              opacity: 0.85;
            }
          }
        }
      }
    }

    .user-content {
      flex: 1;
      padding: 24px;
      overflow: hidden;

      .user-tabs {
        height: 100%;
        background: #fafbfc;
        border-radius: 8px;
        border: 1px solid #e4e7ed;

        :deep(.el-tabs__header) {
          margin: 0;
          padding: 0 20px;
          border-bottom: 1px solid #e4e7ed;
          background: #ffffff;
          border-radius: 8px 8px 0 0;
        }

        :deep(.el-tabs__nav-wrap) {
          padding: 0;
        }

        :deep(.el-tabs__content) {
          flex: 1;
          overflow: hidden;
        }

        :deep(.el-tabs__item) {
          height: 48px;
          line-height: 48px;
          font-size: 14px;
          color: #606266;
          padding: 0 20px;
          transition: all 0.2s;

          &:hover {
            color: #409eff;
          }

          &.is-active {
            color: #409eff;
            font-weight: 500;
          }
        }

        :deep(.el-tabs__active-bar) {
          background-color: #409eff;
        }

        .tab-content {
          padding: 32px;
          height: 100%;
          overflow-y: auto;

          .user-form {
            max-width: 700px;

            &.password-form {
              max-width: 500px;
            }

            :deep(.el-form-item) {
              margin-bottom: 20px;
            }

            :deep(.el-form-item__label) {
              font-weight: 500;
              color: #303133;
            }

            :deep(.el-input__wrapper) {
              transition: border-color 0.2s;

              &:hover {
                box-shadow: 0 0 0 1px #c0c4cc inset;
              }
            }

            .password-strength {
              margin-top: 10px;
              display: flex;
              align-items: center;
              gap: 12px;
              font-size: 13px;
              padding: 8px 12px;
              background: #ffffff;
              border: 1px solid #e4e7ed;
              border-radius: 4px;

              .strength-label {
                color: #606266;
              }

              .strength-bar {
                flex: 1;
                max-width: 150px;
                height: 6px;
                background: #e5e7eb;
                border-radius: 3px;
                overflow: hidden;

                .strength-level {
                  height: 100%;
                  transition:
                    width 0.3s ease,
                    background 0.3s ease;

                  &.weak {
                    background: #ef4444;
                  }

                  &.medium {
                    background: #f59e0b;
                  }

                  &.strong {
                    background: #10b981;
                  }
                }
              }

              .strength-text {
                min-width: 24px;
                font-weight: 500;

                &.weak {
                  color: #ef4444;
                }

                &.medium {
                  color: #f59e0b;
                }

                &.strong {
                  color: #10b981;
                }
              }
            }
          }

          .roles-content {
            .roles-list {
              display: flex;
              flex-wrap: wrap;
              gap: 12px;

              .role-item {
                display: inline-flex;
                align-items: center;
                padding: 8px 16px;
                background: #ffffff;
                border: 1px solid #dcdfe6;
                border-radius: 4px;
                font-size: 14px;
                color: #606266;
              }
            }
          }
        }
      }
    }

    .im-accounts-section {
      display: grid;
      grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
      gap: 16px;
      margin-top: 16px;

      .im-account-card {
        display: flex;
        align-items: center;
        gap: 12px;
        padding: 16px;
        background: #ffffff;
        border: 1px solid #e4e7ed;
        border-radius: 8px;
        transition: all 0.3s ease;

        &:hover {
          border-color: #c0c4cc;
          box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
        }

        .im-account-content {
          flex: 1;
          min-width: 0;

          .im-account-label {
            font-size: 12px;
            color: #909399;
            margin-bottom: 4px;
            font-weight: 500;
          }

          .im-account-value {
            font-size: 14px;
            color: #303133;
            font-weight: 500;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }

        .im-account-status {
          flex-shrink: 0;
          padding: 4px 12px;
          border-radius: 12px;
          font-size: 12px;
          font-weight: 500;
          background: #f5f7fa;
          color: #909399;
          transition: all 0.3s ease;

          &.bound {
            background: #f0f9ff;
            color: #1890ff;
          }
        }
      }
    }
  }

  :deep(.el-empty) {
    padding: 40px 0;
  }
</style>
