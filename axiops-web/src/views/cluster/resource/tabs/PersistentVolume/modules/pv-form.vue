<!-- modules/pv-form.vue -->
<template>
  <ElForm
    ref="formRef"
    :model="formData"
    :rules="rules"
    label-width="120px"
    :disabled="disabled"
    class="pv-form"
  >
    <ElCollapse v-model="activeCollapse" class="unified-collapse">
      <!-- 顶部说明 -->
      <div class="form-alert-wrapper">
        <ElAlert type="info" :closable="false" show-icon>
          <template #default>
            <div class="help-content">
              <p><strong>静态制备 PersistentVolume</strong></p>
              <p>• 手动创建 PV，配置具体的存储源（NFS、HostPath、CSI、Local）</p>
              <p>• StorageClass 为可选项，仅用于分类管理和 PVC 匹配</p>
              <p style="margin-top: 8px; color: #909399; font-size: 12px">
                💡 如需动态制备，请前往
                <ElLink type="primary" :underline="false" style="font-size: 12px">PVC 管理</ElLink>
                创建 PVC，由 StorageClass 自动创建 PV
              </p>
            </div>
          </template>
        </ElAlert>
      </div>

      <!-- 基本信息 -->
      <ElCollapseItem name="basic" class="collapse-section-item">
        <template #title>
          <div class="unified-collapse-header">
            <div class="header-left">
              <ElIcon class="header-icon"><ServerIcon /></ElIcon>
              <span class="header-title">基本信息</span>
            </div>
          </div>
        </template>
        <div class="collapse-content">
          <ElRow :gutter="20">
            <ElCol :span="12">
              <ElFormItem label="PV 名称" prop="name">
                <ElInput
                  v-model="formData.name"
                  placeholder="my-pv"
                  :disabled="isEdit"
                  maxlength="253"
                  show-word-limit
                />
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="容量" prop="capacity">
                <ElInputNumber
                  v-model="formData.capacity"
                  :min="1"
                  :max="10240"
                  :precision="0"
                  style="width: 150px"
                />
                <ElSelect v-model="formData.capacityUnit" style="width: 90px; margin-left: 8px">
                  <ElOption label="Mi" value="Mi" />
                  <ElOption label="Gi" value="Gi" />
                  <ElOption label="Ti" value="Ti" />
                </ElSelect>
                <span class="capacity-hint">≈ {{ capacityHint }}</span>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElFormItem label="访问模式" prop="accessModes">
            <ElCheckboxGroup v-model="formData.accessModes">
              <ElCheckbox value="ReadWriteOnce">
                <DatabaseIcon :size="14" />
                RWO
                <ElTooltip placement="top" :show-after="300">
                  <template #content>
                    <strong>ReadWriteOnce</strong><br />
                    单节点读写
                  </template>
                  <InfoIcon :size="12" class="inline-help" />
                </ElTooltip>
              </ElCheckbox>
              <ElCheckbox value="ReadOnlyMany">
                <EyeIcon :size="14" />
                ROX
                <ElTooltip placement="top" :show-after="300">
                  <template #content>
                    <strong>ReadOnlyMany</strong><br />
                    多节点只读
                  </template>
                  <InfoIcon :size="12" class="inline-help" />
                </ElTooltip>
              </ElCheckbox>
              <ElCheckbox value="ReadWriteMany" :disabled="!supportsRWX">
                <DatabaseZapIcon :size="14" />
                RWX
                <ElTooltip placement="top" :show-after="300">
                  <template #content>
                    <strong>ReadWriteMany</strong><br />
                    {{
                      supportsRWX ? '多节点读写（当前存储支持）' : '多节点读写（当前存储不支持）'
                    }}
                  </template>
                  <InfoIcon :size="12" class="inline-help" />
                </ElTooltip>
              </ElCheckbox>
              <ElCheckbox value="ReadWriteOncePod">
                <BoxIcon :size="14" />
                RWOP
                <ElTooltip placement="top" :show-after="300">
                  <template #content>
                    <strong>ReadWriteOncePod</strong><br />
                    单 Pod 读写 (K8s 1.22+)
                  </template>
                  <InfoIcon :size="12" class="inline-help" />
                </ElTooltip>
              </ElCheckbox>
            </ElCheckboxGroup>
          </ElFormItem>

          <ElRow :gutter="20">
            <ElCol :span="12">
              <ElFormItem label="卷模式" prop="volumeMode">
                <ElRadioGroup v-model="formData.volumeMode">
                  <ElRadio value="Filesystem">
                    <FolderIcon :size="14" />
                    文件系统
                  </ElRadio>
                  <ElRadio value="Block">
                    <HardDriveIcon :size="14" />
                    块设备
                  </ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </ElCol>
            <ElCol :span="12">
              <ElFormItem label="回收策略" prop="reclaimPolicy">
                <ElSelect v-model="formData.reclaimPolicy" style="width: 100%">
                  <ElOption value="Retain">
                    <div class="select-option">
                      <SaveIcon :size="14" />
                      <span>Retain - 保留（推荐）</span>
                    </div>
                  </ElOption>
                  <ElOption value="Delete">
                    <div class="select-option danger">
                      <TrashIcon :size="14" />
                      <span>Delete - 自动删除</span>
                    </div>
                  </ElOption>
                </ElSelect>
              </ElFormItem>
            </ElCol>
          </ElRow>

          <ElFormItem label="StorageClass">
            <ElSelect
              v-model="formData.storageClassName"
              filterable
              allow-create
              clearable
              placeholder="可选，用于分类管理和 PVC 匹配"
              style="width: 100%"
            >
              <ElOption
                v-for="sc in storageClasses"
                :key="sc.name"
                :label="sc.name"
                :value="sc.name"
              >
                <div class="sc-option">
                  <span>{{ sc.name }}</span>
                  <ElTag v-if="sc.isDefault" type="primary" size="small">默认</ElTag>
                  <ElTag type="info" size="small">{{ sc.provisioner }}</ElTag>
                </div>
              </ElOption>
            </ElSelect>
            <div class="field-tip"> 可选项，指定后 PVC 可通过 storageClassName 匹配到此 PV </div>
          </ElFormItem>
        </div>
      </ElCollapseItem>

      <!-- 存储源配置 -->
      <ElCollapseItem name="storage" class="collapse-section-item">
        <template #title>
          <div class="unified-collapse-header">
            <div class="header-left">
              <ElIcon class="header-icon"><HardDriveIcon /></ElIcon>
              <span class="header-title">存储源配置</span>
              <span class="optional-mark" style="background: #fde2e2; color: #f56c6c">必填</span>
            </div>
          </div>
        </template>
        <div class="collapse-content">
          <ElFormItem label="存储类型" prop="sourceType">
            <ElRadioGroup v-model="formData.sourceType" :disabled="isEdit">
              <ElTooltip content="网络文件系统，支持多节点共享" placement="top" :show-after="300">
                <ElRadioButton value="nfs">
                  <ServerIcon :size="14" />
                  NFS
                </ElRadioButton>
              </ElTooltip>
              <ElTooltip content="主机路径，仅用于单节点测试" placement="top" :show-after="300">
                <ElRadioButton value="hostPath">
                  <FolderIcon :size="14" />
                  HostPath
                </ElRadioButton>
              </ElTooltip>
              <ElTooltip content="容器存储接口，灵活的存储驱动" placement="top" :show-after="300">
                <ElRadioButton value="csi">
                  <PlugZapIcon :size="14" />
                  CSI
                </ElRadioButton>
              </ElTooltip>
              <ElTooltip content="本地存储，绑定到特定节点" placement="top" :show-after="300">
                <ElRadioButton value="local">
                  <HardDriveIcon :size="14" />
                  Local
                </ElRadioButton>
              </ElTooltip>
            </ElRadioGroup>
          </ElFormItem>

          <!-- NFS 配置 -->
          <template v-if="formData.sourceType === 'nfs'">
            <ElRow :gutter="20">
              <ElCol :span="12">
                <ElFormItem label="NFS 服务器" prop="nfs.server">
                  <ElInput
                    v-model="formData.nfs.server"
                    placeholder="192.168.1.100 或 nfs.example.com"
                  >
                    <template #prefix>
                      <ServerIcon :size="16" />
                    </template>
                  </ElInput>
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="导出路径" prop="nfs.path">
                  <ElInput v-model="formData.nfs.path" placeholder="/exports/data">
                    <template #prefix>
                      <FolderOpenIcon :size="16" />
                    </template>
                  </ElInput>
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElFormItem label="只读挂载">
              <ElSwitch v-model="formData.nfs.readOnly" active-text="是" inactive-text="否" />
            </ElFormItem>
          </template>

          <!-- HostPath 配置 -->
          <template v-if="formData.sourceType === 'hostPath'">
            <ElAlert type="warning" :closable="false" style="margin-bottom: 16px" show-icon>
              <template #default>
                ⚠️ HostPath 仅用于单节点集群测试，生产环境请使用其他存储类型
              </template>
            </ElAlert>
            <ElRow :gutter="20">
              <ElCol :span="12">
                <ElFormItem label="主机路径" prop="hostPath.path">
                  <ElInput v-model="formData.hostPath.path" placeholder="/mnt/data">
                    <template #prefix>
                      <FolderIcon :size="16" />
                    </template>
                  </ElInput>
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="路径类型" prop="hostPath.type">
                  <ElSelect v-model="formData.hostPath.type" style="width: 100%">
                    <ElOption label="DirectoryOrCreate" value="DirectoryOrCreate" />
                    <ElOption label="Directory" value="Directory" />
                    <ElOption label="FileOrCreate" value="FileOrCreate" />
                    <ElOption label="File" value="File" />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
            </ElRow>
          </template>

          <!-- CSI 配置 -->
          <template v-if="formData.sourceType === 'csi'">
            <ElRow :gutter="20">
              <ElCol :span="12">
                <ElFormItem label="CSI 驱动" prop="csi.driver">
                  <ElSelect
                    v-model="formData.csi.driver"
                    filterable
                    allow-create
                    placeholder="选择或输入驱动名称"
                    style="width: 100%"
                  >
                    <ElOption label="nfs.csi.k8s.io" value="nfs.csi.k8s.io" />
                    <ElOption label="driver.longhorn.io" value="driver.longhorn.io" />
                    <ElOption label="rbd.csi.ceph.com" value="rbd.csi.ceph.com" />
                    <ElOption label="cephfs.csi.ceph.com" value="cephfs.csi.ceph.com" />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="卷句柄" prop="csi.volumeHandle">
                  <ElInput v-model="formData.csi.volumeHandle" placeholder="卷的唯一标识符" />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElRow :gutter="20">
              <ElCol :span="12">
                <ElFormItem label="文件系统类型">
                  <ElSelect v-model="formData.csi.fsType" style="width: 100%">
                    <ElOption label="ext4 (推荐)" value="ext4" />
                    <ElOption label="xfs" value="xfs" />
                  </ElSelect>
                </ElFormItem>
              </ElCol>
              <ElCol :span="12">
                <ElFormItem label="只读">
                  <ElSwitch v-model="formData.csi.readOnly" active-text="是" inactive-text="否" />
                </ElFormItem>
              </ElCol>
            </ElRow>
            <ElFormItem label="卷属性">
              <div class="key-value-list">
                <div
                  v-for="(item, index) in formData.csi.volumeAttributes"
                  :key="index"
                  class="key-value-item"
                >
                  <ElInput v-model="item.key" placeholder="属性名" style="width: 200px" />
                  <span class="separator">=</span>
                  <ElInput v-model="item.value" placeholder="属性值" style="flex: 1" />
                  <ElButton
                    type="danger"
                    :icon="XIcon"
                    circle
                    size="small"
                    @click="formData.csi.volumeAttributes.splice(index, 1)"
                  />
                </div>
                <ElButton
                  type="primary"
                  :icon="PlusIcon"
                  @click="formData.csi.volumeAttributes.push({ key: '', value: '' })"
                  plain
                >
                  添加属性
                </ElButton>
              </div>
            </ElFormItem>
          </template>

          <!-- Local 配置 -->
          <template v-if="formData.sourceType === 'local'">
            <ElAlert type="info" :closable="false" style="margin-bottom: 16px" show-icon>
              <template #default> Local 卷会被绑定到特定节点，Pod 只能在该节点上运行 </template>
            </ElAlert>
            <ElFormItem label="本地路径" prop="local.path">
              <ElInput v-model="formData.local.path" placeholder="/mnt/local-storage">
                <template #prefix>
                  <FolderIcon :size="16" />
                </template>
              </ElInput>
            </ElFormItem>
            <ElFormItem label="节点亲和性" required>
              <div class="node-affinity-list">
                <div
                  v-for="(req, index) in formData.nodeAffinity"
                  :key="index"
                  class="affinity-item"
                >
                  <ElRow :gutter="12">
                    <ElCol :span="8">
                      <ElInput
                        v-model="req.key"
                        placeholder="kubernetes.io/hostname"
                        size="small"
                      />
                    </ElCol>
                    <ElCol :span="4">
                      <ElSelect v-model="req.operator" size="small">
                        <ElOption label="In" value="In" />
                        <ElOption label="NotIn" value="NotIn" />
                        <ElOption label="Exists" value="Exists" />
                        <ElOption label="DoesNotExist" value="DoesNotExist" />
                      </ElSelect>
                    </ElCol>
                    <ElCol :span="10">
                      <ElSelect
                        v-model="req.values"
                        multiple
                        filterable
                        allow-create
                        default-first-option
                        placeholder="节点值"
                        size="small"
                        :disabled="['Exists', 'DoesNotExist'].includes(req.operator)"
                      />
                    </ElCol>
                    <ElCol :span="2">
                      <ElButton
                        type="danger"
                        :icon="XIcon"
                        circle
                        size="small"
                        @click="formData.nodeAffinity.splice(index, 1)"
                      />
                    </ElCol>
                  </ElRow>
                </div>
                <ElButton
                  type="primary"
                  :icon="PlusIcon"
                  @click="
                    formData.nodeAffinity.push({
                      key: 'kubernetes.io/hostname',
                      operator: 'In',
                      values: []
                    })
                  "
                  size="small"
                  plain
                >
                  添加节点选择器
                </ElButton>
              </div>
            </ElFormItem>
          </template>
        </div>
      </ElCollapseItem>

      <!-- 高级选项 -->
      <ElCollapseItem name="advanced" class="collapse-section-item">
        <template #title>
          <div class="unified-collapse-header">
            <div class="header-left">
              <ElIcon class="header-icon"><Setting /></ElIcon>
              <span class="header-title">高级选项</span>
              <span class="optional-mark">可选</span>
            </div>
          </div>
        </template>
        <div class="collapse-content">
          <ElFormItem label="挂载选项">
            <div class="mount-options-wrapper">
              <div class="quick-options">
                <span class="quick-label">快捷选项：</span>
                <ElTag
                  v-for="opt in commonMountOptions"
                  :key="opt.value"
                  :type="formData.mountOptions.includes(opt.value) ? 'primary' : 'info'"
                  :effect="formData.mountOptions.includes(opt.value) ? 'dark' : 'plain'"
                  class="quick-tag"
                  @click="toggleMountOption(opt.value)"
                >
                  {{ opt.label }}
                </ElTag>
              </div>
              <div class="mount-options">
                <ElTag
                  v-for="(option, index) in formData.mountOptions"
                  :key="index"
                  closable
                  @close="formData.mountOptions.splice(index, 1)"
                >
                  {{ option }}
                </ElTag>
                <ElInput
                  v-model="newMountOption"
                  placeholder="自定义选项"
                  style="width: 160px"
                  size="small"
                  @keyup.enter="addMountOption"
                >
                  <template #append>
                    <ElButton :icon="PlusIcon" @click="addMountOption" />
                  </template>
                </ElInput>
              </div>
            </div>
          </ElFormItem>

          <ElFormItem label="标签">
            <div class="key-value-list">
              <div v-for="(item, index) in formData.labels" :key="index" class="key-value-item">
                <ElInput v-model="item.key" placeholder="键" style="width: 200px" size="small" />
                <span class="separator">:</span>
                <ElInput v-model="item.value" placeholder="值" style="flex: 1" size="small" />
                <ElButton
                  type="danger"
                  :icon="XIcon"
                  circle
                  size="small"
                  @click="formData.labels.splice(index, 1)"
                />
              </div>
              <ElButton
                type="primary"
                :icon="PlusIcon"
                @click="formData.labels.push({ key: '', value: '' })"
                plain
                size="small"
              >
                添加标签
              </ElButton>
            </div>
          </ElFormItem>

          <ElFormItem label="注解">
            <div class="key-value-list">
              <div
                v-for="(item, index) in formData.annotations"
                :key="index"
                class="key-value-item"
              >
                <ElInput v-model="item.key" placeholder="键" style="width: 200px" size="small" />
                <span class="separator">:</span>
                <ElInput v-model="item.value" placeholder="值" style="flex: 1" size="small" />
                <ElButton
                  type="danger"
                  :icon="XIcon"
                  circle
                  size="small"
                  @click="formData.annotations.splice(index, 1)"
                />
              </div>
              <ElButton
                type="primary"
                :icon="PlusIcon"
                @click="formData.annotations.push({ key: '', value: '' })"
                plain
                size="small"
              >
                添加注解
              </ElButton>
            </div>
          </ElFormItem>
        </div>
      </ElCollapseItem>
    </ElCollapse>
  </ElForm>
</template>

<script setup lang="ts">
  import { ref, computed, watch } from 'vue'
  import type { FormInstance, FormRules } from 'element-plus'
  import { Setting } from '@element-plus/icons-vue'
  import {
    Plus as PlusIcon,
    X as XIcon,
    Info as InfoIcon,
    Server as ServerIcon,
    Folder as FolderIcon,
    FolderOpen as FolderOpenIcon,
    PlugZap as PlugZapIcon,
    HardDrive as HardDriveIcon,
    Database as DatabaseIcon,
    DatabaseZap as DatabaseZapIcon,
    Eye as EyeIcon,
    Box as BoxIcon,
    Save as SaveIcon,
    Trash as TrashIcon,
    Settings as SettingsIcon
  } from 'lucide-vue-next'
  import type { PVFormData } from './type'
  import { supportsRWX as checkRWXSupport } from './type'

  interface Props {
    modelValue: PVFormData
    disabled?: boolean
    isEdit?: boolean
    storageClasses?: Array<{
      name: string
      provisioner: string
      isDefault: boolean
    }>
  }

  const props = withDefaults(defineProps<Props>(), {
    disabled: false,
    isEdit: false,
    storageClasses: () => []
  })

  const emit = defineEmits<{
    'update:modelValue': [value: PVFormData]
  }>()

  const formRef = ref<FormInstance>()
  const formData = ref<PVFormData>(props.modelValue)
  const newMountOption = ref('')

  // 折叠面板激活状态
  const activeCollapse = ref(['basic', 'storage', 'advanced'])

  // 容量提示
  const capacityHint = computed(() => {
    const { capacity, capacityUnit } = formData.value
    if (capacityUnit === 'Mi') return `${(capacity / 1024).toFixed(2)} Gi`
    if (capacityUnit === 'Gi') return `${(capacity / 1024).toFixed(2)} Ti`
    if (capacityUnit === 'Ti') return `${capacity * 1024} Gi`
    return ''
  })

  // 是否支持 RWX
  const supportsRWX = computed(() => {
    return checkRWXSupport(formData.value.sourceType, formData.value.csi.driver)
  })

  // 表单验证规则
  const rules: FormRules = {
    name: [
      { required: true, message: '请输入 PV 名称', trigger: 'blur' },
      {
        pattern: /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/,
        message: '名称格式不正确',
        trigger: 'blur'
      }
    ],
    capacity: [
      { required: true, message: '请输入容量', trigger: 'blur' },
      { type: 'number', min: 1, message: '容量必须大于 0', trigger: 'blur' }
    ],
    accessModes: [
      {
        required: true,
        message: '请选择至少一个访问模式',
        trigger: 'change',
        type: 'array',
        min: 1
      }
    ],
    'nfs.server': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'nfs' && !value) {
            callback(new Error('请输入 NFS 服务器'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    'nfs.path': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'nfs') {
            if (!value) {
              callback(new Error('请输入 NFS 路径'))
            } else if (!value.startsWith('/')) {
              callback(new Error('路径必须以 / 开头'))
            } else {
              callback()
            }
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    'hostPath.path': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'hostPath') {
            if (!value) {
              callback(new Error('请输入主机路径'))
            } else if (!value.startsWith('/')) {
              callback(new Error('路径必须以 / 开头'))
            } else {
              callback()
            }
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    'csi.driver': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'csi' && !value) {
            callback(new Error('请输入 CSI 驱动'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    'csi.volumeHandle': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'csi' && !value) {
            callback(new Error('请输入卷句柄'))
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ],
    'local.path': [
      {
        validator: (rule, value, callback) => {
          if (formData.value.sourceType === 'local') {
            if (!value) {
              callback(new Error('请输入本地路径'))
            } else if (!value.startsWith('/')) {
              callback(new Error('路径必须以 / 开头'))
            } else {
              callback()
            }
          } else {
            callback()
          }
        },
        trigger: 'blur'
      }
    ]
  }

  // 监听表单数据变化
  watch(
    formData,
    (val) => {
      emit('update:modelValue', val)
    },
    { deep: true }
  )

  // 监听外部数据变化
  watch(
    () => props.modelValue,
    (val) => {
      formData.value = val
    },
    { deep: true }
  )

  // 监听存储类型变化，自动调整 RWX
  watch(
    () => [formData.value.sourceType, formData.value.csi.driver],
    () => {
      if (!supportsRWX.value && formData.value.accessModes.includes('ReadWriteMany')) {
        formData.value.accessModes = formData.value.accessModes.filter(
          (mode) => mode !== 'ReadWriteMany'
        )
      }
    }
  )

  // 常用挂载选项
  const commonMountOptions = [
    { label: 'hard', value: 'hard' },
    { label: 'soft', value: 'soft' },
    { label: 'noatime', value: 'noatime' },
    { label: 'nodiratime', value: 'nodiratime' },
    { label: 'ro', value: 'ro' },
    { label: 'nfsvers=4', value: 'nfsvers=4' },
    { label: 'nfsvers=3', value: 'nfsvers=3' }
  ]

  // 切换挂载选项
  const toggleMountOption = (option: string) => {
    const index = formData.value.mountOptions.indexOf(option)
    if (index > -1) {
      formData.value.mountOptions.splice(index, 1)
    } else {
      formData.value.mountOptions.push(option)
    }
  }

  // 添加挂载选项
  const addMountOption = () => {
    const option = newMountOption.value.trim()
    if (option && !formData.value.mountOptions.includes(option)) {
      formData.value.mountOptions.push(option)
      newMountOption.value = ''
    }
  }

  // 验证表单
  const validate = () => {
    return formRef.value?.validate()
  }

  // 重置表单
  const resetFields = () => {
    return formRef.value?.resetFields()
  }

  defineExpose({
    validate,
    resetFields
  })
</script>

<style lang="scss" scoped>
  @use '@/views/cluster/resource/styles/resource-form.scss' as *;
  @use '@/assets/styles/core/components/unified-collapse.scss' as *;

  .pv-form {
    .form-alert-wrapper {
      margin-bottom: 16px;
    }

    .help-content {
      font-size: 13px;
      line-height: 1.8;

      p {
        margin: 0 0 4px;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }

    .field-tip {
      @include form-item-tip;
    }

    .capacity-hint {
      margin-left: 12px;
      font-size: 13px;
      color: #909399;
    }

    .inline-help {
      margin-left: 4px;
      color: #909399;
      cursor: help;
      vertical-align: middle;
    }

    .select-option {
      display: flex;
      align-items: center;
      gap: 6px;

      &.danger {
        color: #f56c6c;
      }
    }

    .sc-option {
      display: flex;
      align-items: center;
      gap: 8px;
      width: 100%;
    }

    .key-value-list {
      display: flex;
      flex-direction: column;
      gap: 8px;
      width: 100%;

      .key-value-item {
        display: flex;
        align-items: center;
        gap: 8px;

        .separator {
          color: #909399;
          font-weight: 500;
        }
      }
    }

    .mount-options-wrapper {
      width: 100%;

      .quick-options {
        display: flex;
        flex-wrap: wrap;
        align-items: center;
        gap: 8px;
        margin-bottom: 12px;
        padding-bottom: 12px;
        border-bottom: 1px dashed #e4e7ed;

        .quick-label {
          font-size: 12px;
          color: #909399;
        }

        .quick-tag {
          cursor: pointer;
          transition: all 0.2s;

          &:hover {
            transform: scale(1.05);
          }
        }
      }
    }

    .mount-options {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      align-items: center;
    }

    .node-affinity-list {
      width: 100%;

      .affinity-item {
        margin-bottom: 12px;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }

    :deep(.el-checkbox),
    :deep(.el-radio) {
      margin-right: 20px;

      .el-checkbox__label,
      .el-radio__label {
        display: inline-flex;
        align-items: center;
        gap: 4px;
      }
    }

    :deep(.el-radio-button) {
      .el-radio-button__inner {
        display: inline-flex;
        align-items: center;
        gap: 4px;
      }
    }

    :deep(.el-form-item__label) {
      font-weight: 500;
      color: #606266;
    }
  }

  .tooltip-content {
    max-width: 300px;
    line-height: 1.6;

    p {
      margin: 0 0 8px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    strong {
      color: #409eff;
    }
  }
</style>
