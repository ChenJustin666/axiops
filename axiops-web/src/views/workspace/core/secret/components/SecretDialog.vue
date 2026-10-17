<template>
  <ElDialog
    v-model="visible"
    :title="dialogTitle"
    width="1000px"
    top="5vh"
    align-center
    :close-on-click-modal="false"
    :close-on-press-escape="false"
    @closed="handleClosed"
  >
    <!-- 模式切换 -->
    <div class="unified-mode-switch">
      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <div class="radio-content">
            <FileEdit :size="13" />
            <span>表单模式</span>
          </div>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <div class="radio-content">
            <Code :size="13" />
            <span>YAML 模式</span>
          </div>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态 -->
    <div v-if="dataLoading" class="unified-loading">
      <ElIcon class="is-loading" :size="32">
        <Loader2 />
      </ElIcon>
      <div class="loading-text">正在加载 Secret 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="secret-form-container">
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="rules"
        label-width="120px"
        label-position="right"
        class="secret-form"
      >
        <ElCollapse v-model="activeNames" class="form-collapse">
          <!-- 基本信息 -->
          <ElCollapseItem name="basic" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><InfoFilled /></ElIcon>
                <span>基本信息</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="名称" prop="name">
                <ElInput
                  v-model="formData.name"
                  placeholder="请输入 Secret 名称"
                  :disabled="dialogType === 'edit'"
                  maxlength="63"
                  show-word-limit
                />
              </ElFormItem>

              <ElFormItem label="命名空间">
                <ElInput :value="workspace?.namespace" disabled />
              </ElFormItem>

              <ElFormItem label="类型" prop="type">
                <ElSelect
                  v-model="formData.type"
                  placeholder="请选择类型"
                  :disabled="dialogType === 'edit'"
                  style="width: 100%"
                  @change="handleTypeChange"
                >
                  <ElOption label="Opaque（通用）" value="Opaque" />
                  <ElOption label="TLS 证书" value="kubernetes.io/tls" />
                  <ElOption label="Docker 配置" value="kubernetes.io/dockerconfigjson" />
                  <ElOption label="Basic Auth" value="kubernetes.io/basic-auth" />
                  <ElOption label="SSH Auth" value="kubernetes.io/ssh-auth" />
                </ElSelect>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 数据配置 - Opaque 类型 -->
          <ElCollapseItem
            v-if="formData.type === 'Opaque'"
            name="data"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Lock /></ElIcon>
                <span>敏感数据</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="warning" :closable="false" show-icon style="margin-bottom: 16px">
                <template #title>
                  支持明文和 Base64
                  两种输入模式。可以为每个数据项单独选择输入模式，提交时会根据目标字段自动处理编码。
                </template>
              </ElAlert>

              <div class="data-list">
                <ElCollapse v-model="activeDataItems" accordion>
                  <ElCollapseItem
                    v-for="(item, index) in formData.data"
                    :key="index"
                    :name="index"
                    class="data-collapse-item"
                  >
                    <template #title>
                      <div class="collapse-title">
                        <div class="title-left">
                          <Key :size="14" />
                          <span class="data-key">{{ item.key || `数据项 ${index + 1}` }}</span>
                          <ElTag
                            v-if="item.inputMode"
                            size="small"
                            :type="item.inputMode === 'plain' ? 'success' : 'warning'"
                          >
                            {{ item.inputMode === 'plain' ? '明文' : 'Base64' }}
                          </ElTag>
                          <ElTag v-if="item.dataSource" size="small" type="info">
                            {{ item.dataSource === 'data' ? 'data' : 'stringData' }}
                          </ElTag>
                          <ElTag v-if="item.value" size="small">
                            {{ item.value.length }} 字符
                          </ElTag>
                        </div>
                        <ElButton
                          type="danger"
                          :icon="Trash2"
                          circle
                          size="small"
                          @click.stop="removeData(index)"
                        />
                      </div>
                    </template>

                    <div class="data-item-content">
                      <ElFormItem label="键（Key）" :prop="`data.${index}.key`">
                        <ElInput
                          v-model="item.key"
                          placeholder="例如: password"
                          maxlength="253"
                          show-word-limit
                        />
                      </ElFormItem>

                      <!-- 统一的值输入：支持明文和 Base64 输入 -->
                      <ElFormItem label="值（Value）">
                        <div class="data-value-wrapper">
                          <ElInput
                            v-model="item.value"
                            type="textarea"
                            :placeholder="
                              item.inputMode === 'plain' ? '明文内容' : 'Base64 编码内容'
                            "
                            :rows="6"
                            :autosize="{ minRows: 6, maxRows: 15 }"
                            class="data-value-textarea"
                          />
                          <div class="input-mode-toggle">
                            <ElButton
                              :type="item.inputMode === 'plain' ? 'primary' : ''"
                              size="small"
                              @click="toggleInputMode(index, 'plain')"
                              :title="'明文输入模式'"
                            >
                              明文
                            </ElButton>
                            <ElButton
                              :type="item.inputMode === 'base64' ? 'primary' : ''"
                              size="small"
                              @click="toggleInputMode(index, 'base64')"
                              :title="'Base64 输入模式'"
                            >
                              Base64
                            </ElButton>
                          </div>
                        </div>
                        <div class="value-info">
                          <span>当前长度: {{ item.value?.length || 0 }} 字符</span>
                          <span
                            v-if="item.inputMode === 'plain'"
                            style="margin-left: 12px; color: #67c23a"
                          >
                            • 明文输入
                          </span>
                          <span v-else style="margin-left: 12px; color: #e6a23c">
                            • Base64 输入
                          </span>
                          <span
                            v-if="item.dataSource === 'data'"
                            style="margin-left: 12px; color: #909399"
                          >
                            (提交到 data 字段)
                          </span>
                          <span
                            v-else-if="item.dataSource === 'stringData'"
                            style="margin-left: 12px; color: #909399"
                          >
                            (提交到 stringData 字段)
                          </span>
                        </div>
                      </ElFormItem>
                    </div>
                  </ElCollapseItem>
                </ElCollapse>
              </div>

              <ElButton type="primary" :icon="Plus" plain @click="addData" style="width: 100%">
                添加数据项
              </ElButton>
            </div>
          </ElCollapseItem>

          <!-- TLS 证书类型 -->
          <ElCollapseItem
            v-else-if="formData.type === 'kubernetes.io/tls'"
            name="tls"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><KeyIcon /></ElIcon>
                <span>TLS 证书</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 20px">
                <template #title>
                  TLS Secret 包含 tls.crt（证书）和 tls.key（私钥）两个必需字段
                </template>
              </ElAlert>

              <ElFormItem label="证书 (tls.crt)" required>
                <div class="input-with-visibility">
                  <ElInput
                    v-model="formData.tlsCrt"
                    type="textarea"
                    :rows="15"
                    placeholder="请粘贴 PEM 格式的证书内容"
                    :class="{ 'password-mode': !tlsCrtVisible }"
                  />
                  <ElButton
                    :icon="tlsCrtVisible ? Eye : EyeOff"
                    circle
                    size="small"
                    @click="tlsCrtVisible = !tlsCrtVisible"
                    class="visibility-toggle-btn"
                  />
                </div>
              </ElFormItem>

              <ElFormItem label="私钥 (tls.key)" required>
                <div class="input-with-visibility">
                  <ElInput
                    v-model="formData.tlsKey"
                    type="textarea"
                    :rows="15"
                    placeholder="请粘贴 PEM 格式的私钥内容"
                    :class="{ 'password-mode': !tlsKeyVisible }"
                  />
                  <ElButton
                    :icon="tlsKeyVisible ? Eye : EyeOff"
                    circle
                    size="small"
                    @click="tlsKeyVisible = !tlsKeyVisible"
                    class="visibility-toggle-btn"
                  />
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- Docker 配置类型 -->
          <ElCollapseItem
            v-else-if="formData.type === 'kubernetes.io/dockerconfigjson'"
            name="docker"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><BoxIcon /></ElIcon>
                <span>Docker 配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 20px">
                <template #title>
                  用于拉取私有镜像仓库的镜像，包含 .dockerconfigjson 字段
                </template>
              </ElAlert>

              <ElFormItem label="镜像仓库地址" required>
                <ElInput v-model="formData.dockerServer" placeholder="例如: docker.io" />
              </ElFormItem>

              <ElFormItem label="用户名" required>
                <ElInput v-model="formData.dockerUsername" placeholder="请输入用户名" />
              </ElFormItem>

              <ElFormItem label="密码" required>
                <div class="input-with-visibility">
                  <ElInput
                    v-model="formData.dockerPassword"
                    :type="dockerPasswordVisible ? 'text' : 'password'"
                    placeholder="请输入密码"
                  />
                  <ElButton
                    :icon="dockerPasswordVisible ? Eye : EyeOff"
                    circle
                    size="small"
                    @click="dockerPasswordVisible = !dockerPasswordVisible"
                    class="visibility-toggle-btn"
                  />
                </div>
              </ElFormItem>

              <ElFormItem label="邮箱">
                <ElInput v-model="formData.dockerEmail" placeholder="请输入邮箱（可选）" />
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- Basic Auth 类型 -->
          <ElCollapseItem
            v-else-if="formData.type === 'kubernetes.io/basic-auth'"
            name="basicauth"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><UserIcon /></ElIcon>
                <span>基础认证</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 20px">
                <template #title> 用于 HTTP 基础认证，包含 username 和 password 字段 </template>
              </ElAlert>

              <ElFormItem label="用户名 (username)" required>
                <ElInput v-model="formData.basicAuthUsername" placeholder="请输入用户名" />
              </ElFormItem>

              <ElFormItem label="密码 (password)" required>
                <div class="input-with-visibility">
                  <ElInput
                    v-model="formData.basicAuthPassword"
                    :type="basicAuthPasswordVisible ? 'text' : 'password'"
                    placeholder="请输入密码"
                  />
                  <ElButton
                    :icon="basicAuthPasswordVisible ? Eye : EyeOff"
                    circle
                    size="small"
                    @click="basicAuthPasswordVisible = !basicAuthPasswordVisible"
                    class="visibility-toggle-btn"
                  />
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- SSH Auth 类型 -->
          <ElCollapseItem
            v-else-if="formData.type === 'kubernetes.io/ssh-auth'"
            name="ssh"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><ConnectionIcon /></ElIcon>
                <span>SSH 认证</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElAlert type="info" :closable="false" show-icon style="margin-bottom: 20px">
                <template #title> 用于 SSH 认证，包含 ssh-privatekey 字段 </template>
              </ElAlert>

              <ElFormItem label="SSH 私钥 (ssh-privatekey)" required>
                <div class="input-with-visibility">
                  <ElInput
                    v-model="formData.sshPrivateKey"
                    type="textarea"
                    :rows="18"
                    placeholder="请粘贴 SSH 私钥内容"
                    :class="{ 'password-mode': !sshPrivateKeyVisible }"
                  />
                  <ElButton
                    :icon="sshPrivateKeyVisible ? Eye : EyeOff"
                    circle
                    size="small"
                    @click="sshPrivateKeyVisible = !sshPrivateKeyVisible"
                    class="visibility-toggle-btn"
                  />
                </div>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 标签和注解（可选） -->
          <ElCollapseItem name="metadata" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><PriceTag /></ElIcon>
                <span>标签和注解（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="标签 (Labels)">
                <ElTable :data="formData.labels" border style="width: 100%">
                  <ElTableColumn label="标签名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="标签名" maxlength="63" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="标签值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="标签值" maxlength="63" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeLabel($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addLabel"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加标签
                </ElButton>
              </ElFormItem>

              <ElFormItem label="注解 (Annotations)">
                <ElTable :data="formData.annotations" border style="width: 100%">
                  <ElTableColumn label="注解名" width="200">
                    <template #default="{ row }">
                      <ElInput v-model="row.key" placeholder="注解名" maxlength="253" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="注解值">
                    <template #default="{ row }">
                      <ElInput v-model="row.value" placeholder="注解值" maxlength="253" />
                    </template>
                  </ElTableColumn>
                  <ElTableColumn label="操作" width="60" align="center">
                    <template #default="{ $index }">
                      <ElButton type="danger" link @click="removeAnnotation($index)">删除</ElButton>
                    </template>
                  </ElTableColumn>
                </ElTable>
                <ElButton
                  type="primary"
                  @click="addAnnotation"
                  plain
                  size="small"
                  style="margin-top: 8px"
                >
                  添加注解
                </ElButton>
              </ElFormItem>
            </div>
          </ElCollapseItem>
        </ElCollapse>
      </ElForm>
    </div>

    <!-- YAML 模式 -->
    <div v-else class="unified-yaml-container">
      <YamlEditor
        v-model="yamlContent"
        :filename="`${formData.name || 'secret'}.yaml`"
        :readonly="false"
        height="600px"
        @change="handleYamlChange"
      />
    </div>

    <template #footer>
      <div class="unified-dialog-footer">
        <ElButton @click="handleCancel" :disabled="submitting"> 取消 </ElButton>
        <ElButton type="primary" @click="handleSubmit" :loading="submitting">
          {{ dialogType === 'add' ? '创建' : '更新' }}
        </ElButton>
      </div>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import { ElMessage, ElIcon } from 'element-plus'
  import {
    InfoFilled,
    Lock,
    PriceTag,
    Key as KeyIcon,
    Box as BoxIcon,
    User as UserIcon,
    Connection as ConnectionIcon
  } from '@element-plus/icons-vue'
  import { Loader2 } from 'lucide-vue-next'
  import type { FormInstance, FormRules } from 'element-plus'
  import {
    Plus,
    Trash2,
    FileEdit,
    Code,
    Info,
    FileText,
    Tag,
    Key,
    ShieldCheck,
    Database,
    UserCheck,
    Terminal,
    Eye,
    EyeOff,
    Box,
    User,
    Connection
  } from 'lucide-vue-next'
  import * as yaml from 'js-yaml'
  import {
    createSecretApi,
    updateSecretApi,
    getSecretDetailApi,
    type SecretListItem,
    type ProjectWorkspace
  } from '@/api'
  import YamlEditor from '@/components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../utils/k8s-validator'

  interface Props {
    modelValue: boolean
    dialogType: 'add' | 'edit'
    secretData?: SecretListItem
    workspace: ProjectWorkspace | null
  }

  interface KeyValue {
    key: string
    value: string // 用户看到和编辑的值（可能是明文或 base64）
    dataSource?: 'data' | 'stringData' // 数据的原始来源，决定提交时放入哪个字段
    inputMode?: 'plain' | 'base64' // 当前输入模式：plain=明文输入，base64=base64输入
  }

  interface FormData {
    name: string
    type: string
    labels: KeyValue[]
    annotations: KeyValue[]
    data: KeyValue[]
    tlsCrt: string
    tlsKey: string
    dockerServer: string
    dockerUsername: string
    dockerPassword: string
    dockerEmail: string
    basicAuthUsername: string
    basicAuthPassword: string
    sshPrivateKey: string
  }

  const props = defineProps<Props>()
  const emit = defineEmits(['update:modelValue', 'success'])

  const visible = computed({
    get: () => props.modelValue,
    set: (val) => emit('update:modelValue', val)
  })

  const dialogTitle = computed(() => {
    return props.dialogType === 'add' ? '新增 Secret' : '编辑 Secret'
  })

  const formRef = ref<FormInstance>()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const yamlContent = ref('')
  const activeDataItems = ref<number[]>([0])
  const activeNames = ref<string[]>([
    'basic',
    'data',
    'tls',
    'docker',
    'basicauth',
    'ssh',
    'metadata'
  ])

  // 可见性控制（保留用于其他类型的 Secret）

  const tlsCrtVisible = ref(false)
  const tlsKeyVisible = ref(false)
  const dockerPasswordVisible = ref(false)
  const basicAuthPasswordVisible = ref(false)
  const sshPrivateKeyVisible = ref(false)

  const formData = ref<FormData>({
    name: '',
    type: 'Opaque',
    labels: [],
    annotations: [],
    data: [],
    tlsCrt: '',
    tlsKey: '',
    dockerServer: '',
    dockerUsername: '',
    dockerPassword: '',
    dockerEmail: '',
    basicAuthUsername: '',
    basicAuthPassword: '',
    sshPrivateKey: ''
  })

  const rules: FormRules = {
    name: createDns1123LabelRules(true, 'Secret 名称'),
    type: [{ required: true, message: '请选择类型', trigger: 'change' }]
  }

  watch(visible, async (val) => {
    if (val) {
      await initFormData()
    }
  })

  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return

    if (newMode === 'yaml' && oldMode === 'form') {
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      yamlToForm()
    }
  })

  /**
   * 将字节数组转换为 Base64 字符串
   */
  const byteArrayToBase64 = (byteArray: number[] | string): string => {
    // 如果已经是字符串，直接返回
    if (typeof byteArray === 'string') {
      return byteArray
    }

    try {
      const uint8Array = new Uint8Array(byteArray)
      let binary = ''
      for (let i = 0; i < uint8Array.length; i++) {
        binary += String.fromCharCode(uint8Array[i])
      }
      return btoa(binary)
    } catch (error) {
      console.error('字节数组转Base64失败:', error)
      return ''
    }
  }

  /**
   * Base64 解码为明文
   */
  const base64ToPlainText = (base64Str: string): string => {
    try {
      return decodeURIComponent(escape(atob(base64Str)))
    } catch (error) {
      console.error('Base64解码失败:', error)
      return '[解码失败]'
    }
  }

  /**
   * 明文编码为 Base64
   */
  const plainTextToBase64 = (plainText: string): string => {
    try {
      return btoa(unescape(encodeURIComponent(plainText)))
    } catch (error) {
      console.error('编码失败:', error)
      return ''
    }
  }

  const initFormData = async () => {
    if (props.dialogType === 'edit' && props.secretData) {
      dataLoading.value = true
      try {
        const detail = await getSecretDetailApi({
          workloadId: props.workspace?.id || 0,
          name: props.secretData.name
        })

        formData.value = {
          name: detail.name,
          type: detail.type,
          labels: objectToArray(detail.labels),
          annotations: objectToArray(detail.annotations),
          data: [],
          tlsCrt: '',
          tlsKey: '',
          dockerServer: '',
          dockerUsername: '',
          dockerPassword: '',
          dockerEmail: '',
          basicAuthUsername: '',
          basicAuthPassword: '',
          sshPrivateKey: ''
        }

        // 根据类型解析数据
        if (detail.data) {
          if (detail.type === 'Opaque') {
            // 处理 Opaque 类型的数据
            formData.value.data = Object.entries(detail.data).map(([key, value], index) => {
              // 后端可能返回字节数组或 base64 字符串
              const base64Str = byteArrayToBase64(value)

              // 解码为明文显示
              const plainText = base64ToPlainText(base64Str)

              return {
                key,
                value: plainText, // 默认显示明文
                dataSource: 'data' as const, // 来源是 data 字段
                inputMode: 'plain' as const // 默认明文输入模式
              }
            })
          } else if (detail.type === 'kubernetes.io/tls') {
            const tlsCrt = detail.data['tls.crt']
            const tlsKey = detail.data['tls.key']
            formData.value.tlsCrt =
              typeof tlsCrt === 'string'
                ? base64ToPlainText(tlsCrt)
                : base64ToPlainText(byteArrayToBase64(tlsCrt))
            formData.value.tlsKey =
              typeof tlsKey === 'string'
                ? base64ToPlainText(tlsKey)
                : base64ToPlainText(byteArrayToBase64(tlsKey))
          } else if (detail.type === 'kubernetes.io/basic-auth') {
            const username = detail.data['username']
            const password = detail.data['password']
            formData.value.basicAuthUsername =
              typeof username === 'string'
                ? base64ToPlainText(username)
                : base64ToPlainText(byteArrayToBase64(username))
            formData.value.basicAuthPassword =
              typeof password === 'string'
                ? base64ToPlainText(password)
                : base64ToPlainText(byteArrayToBase64(password))
          } else if (detail.type === 'kubernetes.io/ssh-auth') {
            const sshKey = detail.data['ssh-privatekey']
            formData.value.sshPrivateKey =
              typeof sshKey === 'string'
                ? base64ToPlainText(sshKey)
                : base64ToPlainText(byteArrayToBase64(sshKey))
          } else if (detail.type === 'kubernetes.io/dockerconfigjson') {
            const dockerConfig = detail.data['.dockerconfigjson']
            const dockerConfigStr =
              typeof dockerConfig === 'string'
                ? base64ToPlainText(dockerConfig)
                : base64ToPlainText(byteArrayToBase64(dockerConfig))
            try {
              const config = JSON.parse(dockerConfigStr)
              const firstServer = Object.keys(config.auths || {})[0]
              if (firstServer && config.auths[firstServer]) {
                formData.value.dockerServer = firstServer
                formData.value.dockerUsername = config.auths[firstServer].username || ''
                formData.value.dockerPassword = config.auths[firstServer].password || ''
                formData.value.dockerEmail = config.auths[firstServer].email || ''
              }
            } catch (error) {
              console.error('解析Docker配置失败:', error)
            }
          }
        }
      } catch (error) {
        console.error('获取 Secret 详情失败:', error)
      } finally {
        dataLoading.value = false
      }
    } else {
      resetFormData()
    }

    editMode.value = 'form'
    formToYaml()
  }

  const resetFormData = () => {
    formData.value = {
      name: '',
      type: 'Opaque',
      labels: [],
      annotations: [],
      data: [],
      tlsCrt: '',
      tlsKey: '',
      dockerServer: '',
      dockerUsername: '',
      dockerPassword: '',
      dockerEmail: '',
      basicAuthUsername: '',
      basicAuthPassword: '',
      sshPrivateKey: ''
    }
  }

  const handleTypeChange = () => {
    // 清空特定类型的数据
    formData.value.data = []
    formData.value.tlsCrt = ''
    formData.value.tlsKey = ''
    formData.value.dockerServer = ''
    formData.value.dockerUsername = ''
    formData.value.dockerPassword = ''
    formData.value.dockerEmail = ''
    formData.value.basicAuthUsername = ''
    formData.value.basicAuthPassword = ''
    formData.value.sshPrivateKey = ''
  }

  const objectToArray = (obj?: Record<string, string>): KeyValue[] => {
    if (!obj) return []
    return Object.entries(obj)
      .filter(([key]) => key && key.trim())
      .map(([key, value]) => ({ key, value }))
  }

  const arrayToObject = (arr: KeyValue[]): Record<string, string> => {
    const obj: Record<string, string> = {}
    arr.forEach((item) => {
      if (item.key && item.key.trim() && item.value !== undefined) {
        obj[item.key.trim()] = item.value
      }
    })
    return obj
  }

  const formToYaml = () => {
    const labelsObj = arrayToObject(formData.value.labels)
    const annotationsObj = arrayToObject(formData.value.annotations)

    const secretObj: any = {
      apiVersion: 'v1',
      kind: 'Secret',
      metadata: {
        name: formData.value.name || 'secret-name',
        namespace: props.workspace?.namespace || 'default'
      },
      type: formData.value.type
    }

    // 根据类型设置数据
    if (formData.value.type === 'Opaque') {
      const dataObj: Record<string, string> = {}
      const stringDataObj: Record<string, string> = {}

      formData.value.data.forEach((item) => {
        if (item.key && item.value) {
          // 根据 dataSource 决定数据放入哪个字段
          if (item.dataSource === 'stringData') {
            // 目标是 stringData（明文存储）
            if (item.inputMode === 'plain') {
              // 当前是明文输入，直接使用
              stringDataObj[item.key] = item.value
            } else {
              // 当前是 base64 输入，需要解码为明文
              stringDataObj[item.key] = base64ToPlainText(item.value)
            }
          } else {
            // 目标是 data（base64 存储）
            if (item.inputMode === 'plain') {
              // 当前是明文输入，需要编码为 base64
              dataObj[item.key] = plainTextToBase64(item.value)
            } else {
              // 当前是 base64 输入，直接使用
              dataObj[item.key] = item.value
            }
          }
        }
      })

      if (Object.keys(dataObj).length > 0) {
        secretObj.data = dataObj
      }
      if (Object.keys(stringDataObj).length > 0) {
        secretObj.stringData = stringDataObj
      }
    } else if (formData.value.type === 'kubernetes.io/tls') {
      secretObj.data = {}
      if (formData.value.tlsCrt)
        secretObj.data['tls.crt'] = plainTextToBase64(formData.value.tlsCrt)
      if (formData.value.tlsKey)
        secretObj.data['tls.key'] = plainTextToBase64(formData.value.tlsKey)
    } else if (formData.value.type === 'kubernetes.io/dockerconfigjson') {
      if (
        formData.value.dockerServer &&
        formData.value.dockerUsername &&
        formData.value.dockerPassword
      ) {
        const auth = btoa(`${formData.value.dockerUsername}:${formData.value.dockerPassword}`)
        const dockerConfig = {
          auths: {
            [formData.value.dockerServer]: {
              username: formData.value.dockerUsername,
              password: formData.value.dockerPassword,
              email: formData.value.dockerEmail || '',
              auth: auth
            }
          }
        }
        secretObj.data = {
          '.dockerconfigjson': plainTextToBase64(JSON.stringify(dockerConfig))
        }
      }
    } else if (formData.value.type === 'kubernetes.io/basic-auth') {
      secretObj.data = {}
      if (formData.value.basicAuthUsername)
        secretObj.data['username'] = plainTextToBase64(formData.value.basicAuthUsername)
      if (formData.value.basicAuthPassword)
        secretObj.data['password'] = plainTextToBase64(formData.value.basicAuthPassword)
    } else if (formData.value.type === 'kubernetes.io/ssh-auth') {
      secretObj.data = {}
      if (formData.value.sshPrivateKey)
        secretObj.data['ssh-privatekey'] = plainTextToBase64(formData.value.sshPrivateKey)
    }

    if (Object.keys(labelsObj).length > 0) {
      secretObj.metadata.labels = labelsObj
    }
    if (Object.keys(annotationsObj).length > 0) {
      secretObj.metadata.annotations = annotationsObj
    }

    yamlContent.value = yaml.dump(secretObj, { indent: 2 })
  }

  const yamlToForm = () => {
    try {
      const obj: any = yaml.load(yamlContent.value)

      if (!obj || typeof obj !== 'object') {
        ElMessage.error('YAML 格式不正确')
        return
      }

      if (obj.kind !== 'Secret') {
        ElMessage.error('YAML 类型必须是 Secret')
        return
      }

      if (props.workspace) {
        if (!obj.metadata) {
          obj.metadata = {}
        }
        obj.metadata.namespace = props.workspace.namespace
      }

      formData.value.name = obj.metadata?.name || ''
      formData.value.type = obj.type || 'Opaque'
      formData.value.labels = objectToArray(obj.metadata?.labels)
      formData.value.annotations = objectToArray(obj.metadata?.annotations)

      // 清空所有数据
      handleTypeChange()

      // 根据类型解析 data 和 stringData 字段
      if (obj.data || obj.stringData) {
        if (obj.type === 'Opaque') {
          // Opaque 类型：解析所有键值对
          const dataEntries: Array<[string, any, 'base64']> = obj.data
            ? Object.entries(obj.data).map(([k, v]) => [k, v, 'base64' as const])
            : []
          const stringDataEntries: Array<[string, any, 'plain']> = obj.stringData
            ? Object.entries(obj.stringData).map(([k, v]) => [k, v, 'plain' as const])
            : []

          const allEntries = [...dataEntries, ...stringDataEntries]

          formData.value.data = allEntries.map(([key, value, mode], index) => {
            if (mode === 'base64') {
              const base64Str = String(value)
              // 解码为明文显示
              const plainText = base64ToPlainText(base64Str)

              return {
                key,
                value: plainText, // 默认显示明文
                dataSource: 'data' as const, // 来自 data 字段
                inputMode: 'plain' as const // 默认明文输入模式
              }
            } else {
              // stringData 中的明文数据
              const plainStr = String(value)

              return {
                key,
                value: plainStr,
                dataSource: 'stringData' as const, // 来自 stringData 字段
                inputMode: 'plain' as const // 默认明文输入模式
              }
            }
          })
        } else if (obj.type === 'kubernetes.io/tls') {
          // TLS 类型：支持 data 和 stringData
          const tlsCrtFromData = obj.data?.['tls.crt']
          const tlsCrtFromStringData = obj.stringData?.['tls.crt']
          const tlsKeyFromData = obj.data?.['tls.key']
          const tlsKeyFromStringData = obj.stringData?.['tls.key']

          if (tlsCrtFromStringData) {
            formData.value.tlsCrt = String(tlsCrtFromStringData)
          } else if (tlsCrtFromData) {
            formData.value.tlsCrt = base64ToPlainText(String(tlsCrtFromData))
          }

          if (tlsKeyFromStringData) {
            formData.value.tlsKey = String(tlsKeyFromStringData)
          } else if (tlsKeyFromData) {
            formData.value.tlsKey = base64ToPlainText(String(tlsKeyFromData))
          }
        } else if (obj.type === 'kubernetes.io/dockerconfigjson') {
          // Docker 配置类型：支持 data 和 stringData
          const dockerConfigFromData = obj.data?.['.dockerconfigjson']
          const dockerConfigFromStringData = obj.stringData?.['.dockerconfigjson']

          let dockerConfigStr = ''
          if (dockerConfigFromStringData) {
            dockerConfigStr = String(dockerConfigFromStringData)
          } else if (dockerConfigFromData) {
            dockerConfigStr = base64ToPlainText(String(dockerConfigFromData))
          }

          if (dockerConfigStr) {
            try {
              const config = JSON.parse(dockerConfigStr)
              const firstServer = Object.keys(config.auths || {})[0]
              if (firstServer && config.auths[firstServer]) {
                formData.value.dockerServer = firstServer
                formData.value.dockerUsername = config.auths[firstServer].username || ''
                formData.value.dockerPassword = config.auths[firstServer].password || ''
                formData.value.dockerEmail = config.auths[firstServer].email || ''
              }
            } catch (error) {
              console.error('解析 Docker 配置失败:', error)
              ElMessage.warning('Docker 配置解析失败，请检查格式')
            }
          }
        } else if (obj.type === 'kubernetes.io/basic-auth') {
          // Basic Auth 类型：支持 data 和 stringData
          const usernameFromData = obj.data?.['username']
          const usernameFromStringData = obj.stringData?.['username']
          const passwordFromData = obj.data?.['password']
          const passwordFromStringData = obj.stringData?.['password']

          if (usernameFromStringData) {
            formData.value.basicAuthUsername = String(usernameFromStringData)
          } else if (usernameFromData) {
            formData.value.basicAuthUsername = base64ToPlainText(String(usernameFromData))
          }

          if (passwordFromStringData) {
            formData.value.basicAuthPassword = String(passwordFromStringData)
          } else if (passwordFromData) {
            formData.value.basicAuthPassword = base64ToPlainText(String(passwordFromData))
          }
        } else if (obj.type === 'kubernetes.io/ssh-auth') {
          // SSH Auth 类型：支持 data 和 stringData
          const sshKeyFromData = obj.data?.['ssh-privatekey']
          const sshKeyFromStringData = obj.stringData?.['ssh-privatekey']

          if (sshKeyFromStringData) {
            formData.value.sshPrivateKey = String(sshKeyFromStringData)
          } else if (sshKeyFromData) {
            formData.value.sshPrivateKey = base64ToPlainText(String(sshKeyFromData))
          }
        }
      }

      // 同步回 YAML 以确保格式正确
      formToYaml()
    } catch (error) {
      console.error('YAML 解析失败:', error)
      ElMessage.error('YAML 解析失败，请检查格式')
    }
  }

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  const addData = () => {
    formData.value.data.push({
      key: '',
      value: '',
      dataSource: 'data', // 新增数据项默认放入 data 字段
      inputMode: 'plain' // 默认明文输入模式
    })
  }

  const removeData = (index: number) => {
    formData.value.data.splice(index, 1)
  }

  /**
   * 切换输入模式（明文/Base64）
   */
  const toggleInputMode = (index: number, mode: 'plain' | 'base64') => {
    const item = formData.value.data[index]
    if (!item || item.inputMode === mode) return

    if (mode === 'base64') {
      // 切换到 Base64 输入模式：将明文编码为 Base64
      if (item.inputMode === 'plain' && item.value) {
        try {
          item.value = plainTextToBase64(item.value)
        } catch (error) {
          console.error('编码失败:', error)
          ElMessage.error('编码失败，请检查输入内容')
          return
        }
      }
      item.inputMode = 'base64'
    } else {
      // 切换到明文输入模式：将 Base64 解码为明文
      if (item.inputMode === 'base64' && item.value) {
        try {
          item.value = base64ToPlainText(item.value)
        } catch (error) {
          console.error('解码失败:', error)
          ElMessage.error('解码失败，输入的内容可能不是有效的 Base64')
          return
        }
      }
      item.inputMode = 'plain'
    }
  }

  const validateFormData = (): boolean => {
    if (formData.value.type === 'Opaque') {
      if (formData.value.data.length === 0) {
        return false
      }
      const hasEmptyKey = formData.value.data.some((item) => !item.key || !item.key.trim())
      if (hasEmptyKey) {
        return false
      }
      // 检查所有数据项的值是否为空
      const hasEmptyValue = formData.value.data.some((item) => !item.value || !item.value.trim())
      if (hasEmptyValue) {
        return false
      }
    } else if (formData.value.type === 'kubernetes.io/tls') {
      if (!formData.value.tlsCrt || !formData.value.tlsKey) {
        return false
      }
    } else if (formData.value.type === 'kubernetes.io/dockerconfigjson') {
      if (
        !formData.value.dockerServer ||
        !formData.value.dockerUsername ||
        !formData.value.dockerPassword
      ) {
        return false
      }
    } else if (formData.value.type === 'kubernetes.io/basic-auth') {
      if (!formData.value.basicAuthUsername || !formData.value.basicAuthPassword) {
        return false
      }
    } else if (formData.value.type === 'kubernetes.io/ssh-auth') {
      if (!formData.value.sshPrivateKey) {
        return false
      }
    }
    return true
  }

  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    if (editMode.value === 'yaml') {
      yamlToForm()
    }

    if (editMode.value === 'form') {
      try {
        await formRef.value?.validate()
      } catch {
        return
      }

      if (!validateFormData()) {
        return
      }
    }

    // 构建 YAML 字符串（formToYaml 会根据 encodingMode 自动处理编码）
    formToYaml()

    submitting.value = true
    try {
      const requestData = {
        workloadId: props.workspace.id,
        name: formData.value.name,
        secretYamlStr: yamlContent.value
      }

      if (props.dialogType === 'add') {
        await createSecretApi(requestData)
        ElMessage.success('创建成功')
      } else {
        await updateSecretApi(requestData)
        ElMessage.success('更新成功')
      }

      emit('success')
      visible.value = false
    } finally {
      submitting.value = false
    }
  }

  const handleCancel = () => {
    visible.value = false
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    resetFormData()
    yamlContent.value = ''
    tlsCrtVisible.value = false
    tlsKeyVisible.value = false
    dockerPasswordVisible.value = false
    basicAuthPasswordVisible.value = false
    sshPrivateKeyVisible.value = false
    activeNames.value = ['basic', 'data', 'labels', 'annotations']
  }
</script>

<style lang="scss" scoped>
  .secret-form-container {
    max-height: 65vh;
    overflow-y: auto;
    padding-right: 8px;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-thumb {
      background: #dcdfe6;
      border-radius: 3px;

      &:hover {
        background: #c0c4cc;
      }
    }
  }

  .secret-form {
    :deep(.form-collapse) {
      border: none;
      background: transparent;

      .el-collapse-item {
        background: #fff;
        border: 1px solid #dcdfe6;
        border-radius: 4px;
        margin-bottom: 12px;
        overflow: hidden;

        &.is-active {
          border-color: #409eff;
        }

        &:last-child {
          margin-bottom: 0;
        }
      }

      .el-collapse-item__header {
        height: 48px;
        line-height: 48px;
        background: #f5f7fa;
        border-bottom: 1px solid #e4e7ed;
        padding: 0 16px;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        transition: all 0.3s;

        &:hover {
          background: #ecf5ff;
        }

        .el-collapse-item__arrow {
          margin-right: 8px;
          color: #909399;
          transition: transform 0.3s;
        }

        &.is-active {
          .el-collapse-item__arrow {
            transform: rotate(90deg);
            color: #409eff;
          }
        }
      }

      .el-collapse-item__wrap {
        background: #fff;
        border-bottom: none;
      }

      .el-collapse-item__content {
        padding: 0;
      }
    }

    .collapse-section-item {
      .collapse-title {
        display: flex;
        align-items: center;
        width: 100%;

        .title-icon {
          margin-right: 8px;
          font-size: 16px;
          color: #409eff;
        }

        span {
          flex: 1;
        }
      }
    }

    .collapse-content {
      padding: 20px 24px;
    }
  }

  // 数据列表样式
  .data-list {
    margin-bottom: 12px;

    .data-collapse-item {
      margin-bottom: 8px;
      background: white;
      border-radius: 4px;
      border: 1px solid #dcdfe6;

      :deep(.el-collapse-item__header) {
        padding: 8px 12px;
        background: white;
        border: none;
        border-radius: 4px;

        &:hover {
          background: #f5f7fa;
        }
      }

      :deep(.el-collapse-item__wrap) {
        border: none;
      }

      :deep(.el-collapse-item__content) {
        padding: 12px;
        padding-top: 0;
      }

      .collapse-title {
        display: flex;
        align-items: center;
        justify-content: space-between;
        width: 100%;
        padding-right: 8px;

        .title-left {
          display: flex;
          align-items: center;
          gap: 8px;
          flex: 1;
          min-width: 0;

          .data-key {
            font-weight: 500;
            color: #303133;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }

      .data-item-content {
        .data-value-textarea {
          :deep(.el-textarea__inner) {
            font-family: var(--ax-font-mono);
            font-size: 13px;
            line-height: 1.6;
          }
        }

        .value-info {
          margin-top: 4px;
          font-size: 12px;
          color: #909399;
          text-align: right;
        }

        .data-value-wrapper {
          display: flex;
          flex-direction: column;
          gap: 8px;
          width: 100%;

          .data-value-textarea {
            flex: 1;
          }

          .input-mode-toggle {
            display: flex;
            gap: 8px;
            justify-content: flex-end;
          }
        }
      }
    }
  }

  .input-with-visibility {
    position: relative;
    display: flex;
    align-items: flex-start;
    gap: 8px;
    width: 100%;

    :deep(.el-input),
    :deep(.el-textarea) {
      flex: 1;
    }

    .visibility-toggle-btn {
      flex-shrink: 0;
      margin-top: 4px;
    }

    :deep(.password-mode .el-textarea__inner) {
      -webkit-text-security: disc;
      font-family: var(--ax-font-mono);
      letter-spacing: 2px;
    }
  }
</style>
