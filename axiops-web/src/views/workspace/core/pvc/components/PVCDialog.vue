<template>
  <ElDialog
    :model-value="visible"
    :title="dialogType === 'add' ? '创建 PVC' : '编辑 PVC'"
    width="850px"
    top="5vh"
    :close-on-click-modal="false"
    @update:model-value="handleClose"
    @closed="handleClosed"
  >
    <!-- 编辑模式切换 -->
    <div class="mode-switch">
      <ElRadioGroup v-model="editMode" size="small">
        <ElRadioButton value="form">
          <div class="radio-content">
            <FormInput :size="13" />
            <span>表单编辑</span>
          </div>
        </ElRadioButton>
        <ElRadioButton value="yaml">
          <div class="radio-content">
            <Code :size="13" />
            <span>YAML 编辑</span>
          </div>
        </ElRadioButton>
      </ElRadioGroup>
    </div>

    <!-- 加载状态 -->
    <div v-if="dataLoading" style="text-align: center; padding: 30px">
      <ElSkeleton :rows="6" animated />
      <div style="margin-top: 12px; color: #909399; font-size: 13px">正在加载 PVC 配置...</div>
    </div>

    <!-- 表单模式 -->
    <div v-else-if="editMode === 'form'" class="pvc-form-container">
      <ElForm
        ref="formRef"
        :model="formData"
        :rules="formRules"
        label-width="140px"
        class="pvc-form"
      >
        <ElCollapse v-model="activeNames" class="form-collapse">
          <!-- 基础信息 -->
          <ElCollapseItem name="basic" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><InfoFilled /></ElIcon>
                <span>基础信息</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="名称" prop="name" required>
                <ElInput
                  v-model="formData.name"
                  placeholder="例如: data-pvc"
                  :disabled="dialogType === 'edit'"
                  maxlength="63"
                  show-word-limit
                >
                  <template #suffix>
                    <ElTooltip
                      content="小写字母、数字、连字符，以字母或数字开头结尾"
                      placement="top"
                    >
                      <HelpCircle :size="13" class="help-icon" />
                    </ElTooltip>
                  </template>
                </ElInput>
              </ElFormItem>

              <ElFormItem label="绑定模式" prop="bindingMode" required>
                <ElRadioGroup v-model="formData.bindingMode" :disabled="dialogType === 'edit'">
                  <ElRadio value="dynamic">
                    <div class="radio-option-content">
                      <span>动态制备</span>
                      <ElTooltip placement="top">
                        <template #content>
                          <div>由 StorageClass 自动创建 PV</div>
                          <div>适合大多数场景（推荐）</div>
                        </template>
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                  <ElRadio value="static">
                    <div class="radio-option-content">
                      <span>静态绑定</span>
                      <ElTooltip placement="top">
                        <template #content>
                          <div>绑定到已存在的 PV</div>
                          <div>适合预先创建好 PV 的场景</div>
                        </template>
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 动态制备 - 存储配置 -->
          <ElCollapseItem
            v-if="formData.bindingMode === 'dynamic'"
            name="storage"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Coin /></ElIcon>
                <span>存储配置</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="存储类" prop="storageClassName">
                <ElSelect
                  v-model="formData.storageClassName"
                  placeholder="留空使用默认存储类"
                  clearable
                  filterable
                  :disabled="dialogType === 'edit'"
                  :loading="storageClassLoading"
                  @change="handleStorageClassChange"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="sc in storageClassList"
                    :key="sc.name"
                    :label="sc.name"
                    :value="sc.name"
                  >
                    <div class="sc-option">
                      <div class="sc-option-main">
                        <span class="sc-name">{{ sc.name }}</span>
                        <ElTag v-if="sc.isDefault" type="success" size="small">默认</ElTag>
                      </div>
                      <div class="sc-option-sub">
                        <span class="sc-provisioner">{{ sc.provisioner }}</span>
                        <ElTag
                          v-if="sc.allowVolumeExpansion"
                          type="info"
                          size="small"
                          effect="plain"
                        >
                          支持扩容
                        </ElTag>
                      </div>
                    </div>
                  </ElOption>
                </ElSelect>
                <div v-if="selectedStorageClass" class="info-tip">
                  <Info :size="12" />
                  <span>
                    {{ selectedStorageClass.provisioner }}
                    {{
                      selectedStorageClass.allowVolumeExpansion
                        ? ' • 支持在线扩容'
                        : ' • 不支持扩容'
                    }}
                    {{
                      selectedStorageClass.reclaimPolicy
                        ? ` • 回收策略: ${selectedStorageClass.reclaimPolicy}`
                        : ''
                    }}
                  </span>
                </div>
              </ElFormItem>

              <ElFormItem label="存储容量" prop="requestStorage" required>
                <ElInput
                  v-model="formData.requestStorage"
                  placeholder="例如: 10Gi, 500Mi"
                  :disabled="dialogType === 'edit' && !canExpandVolume"
                >
                  <template #suffix>
                    <ElTooltip placement="top">
                      <template #content>
                        <div>支持的单位: Ki, Mi, Gi, Ti, Pi</div>
                        <div v-if="dialogType === 'edit'">
                          {{ canExpandVolume ? '支持在线扩容' : '此存储类不支持扩容' }}
                        </div>
                      </template>
                      <HelpCircle :size="13" class="help-icon" />
                    </ElTooltip>
                  </template>
                </ElInput>
                <div v-if="dialogType === 'edit' && !canExpandVolume" class="warning-tip">
                  <AlertCircle :size="12" />
                  <span>当前存储类不支持扩容，容量无法修改</span>
                </div>
              </ElFormItem>

              <ElFormItem label="访问模式" prop="accessModes" required>
                <ElCheckboxGroup v-model="formData.accessModes" :disabled="dialogType === 'edit'">
                  <ElCheckbox value="ReadWriteOnce">
                    <div class="checkbox-option-content">
                      <span>ReadWriteOnce</span>
                      <ElTooltip content="单节点读写" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElCheckbox>
                  <ElCheckbox value="ReadOnlyMany">
                    <div class="checkbox-option-content">
                      <span>ReadOnlyMany</span>
                      <ElTooltip content="多节点只读" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElCheckbox>
                  <ElCheckbox value="ReadWriteMany">
                    <div class="checkbox-option-content">
                      <span>ReadWriteMany</span>
                      <ElTooltip content="多节点读写" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElCheckbox>
                  <ElCheckbox value="ReadWriteOncePod">
                    <div class="checkbox-option-content">
                      <span>ReadWriteOncePod</span>
                      <ElTooltip content="单 Pod 读写 (Kubernetes 1.22+)" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElCheckbox>
                </ElCheckboxGroup>
              </ElFormItem>

              <ElFormItem label="卷模式" prop="volumeMode">
                <ElRadioGroup v-model="formData.volumeMode" :disabled="dialogType === 'edit'">
                  <ElRadio value="Filesystem">
                    <div class="radio-option-content">
                      <span>Filesystem</span>
                      <ElTooltip content="文件系统模式（默认）" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                  <ElRadio value="Block">
                    <div class="radio-option-content">
                      <span>Block</span>
                      <ElTooltip content="块设备模式" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 静态绑定 - PV 选择 -->
          <ElCollapseItem
            v-if="formData.bindingMode === 'static'"
            name="pvselect"
            class="collapse-section-item"
          >
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Coin /></ElIcon>
                <span>PV 选择</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="选择方式" prop="pvSelectMode">
                <ElRadioGroup v-model="formData.pvSelectMode" :disabled="dialogType === 'edit'">
                  <ElRadio value="name">
                    <div class="radio-option-content">
                      <span>指定 PV 名称</span>
                      <ElTooltip content="直接选择一个可用的 PV" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                  <ElRadio value="selector">
                    <div class="radio-option-content">
                      <span>标签选择器</span>
                      <ElTooltip content="通过标签自动匹配符合条件的 PV" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                </ElRadioGroup>
              </ElFormItem>

              <!-- 指定 PV 名称 -->
              <ElFormItem
                v-if="formData.pvSelectMode === 'name'"
                label="选择 PV"
                prop="selectedPV"
                required
              >
                <ElSelect
                  v-model="formData.selectedPV"
                  placeholder="选择可用的 PV"
                  clearable
                  filterable
                  :loading="pvLoading"
                  :disabled="dialogType === 'edit'"
                  @change="handlePVChange"
                  style="width: 100%"
                >
                  <ElOption
                    v-for="pv in availablePVList"
                    :key="pv.name"
                    :label="pv.name"
                    :value="pv.name"
                  >
                    <div class="pv-option">
                      <div class="pv-option-main">
                        <span class="pv-name">{{ pv.name }}</span>
                        <ElTag type="success" size="small">{{ pv.status }}</ElTag>
                      </div>
                      <div class="pv-option-sub">
                        <span>容量: {{ pv.capacity }}</span>
                        <span>访问: {{ pv.accessModes }}</span>
                        <span v-if="pv.storageClass">SC: {{ pv.storageClass }}</span>
                      </div>
                    </div>
                  </ElOption>
                </ElSelect>
                <div v-if="selectedPV" class="info-tip">
                  <Info :size="12" />
                  <span>
                    容量: {{ selectedPV.capacity }} • 访问模式: {{ selectedPV.accessModes }} •
                    回收策略:
                    {{ selectedPV.reclaimPolicy }}
                  </span>
                </div>
                <div v-if="availablePVList.length === 0 && !pvLoading" class="warning-tip">
                  <AlertCircle :size="12" />
                  <span>当前集群中没有可用的 PV，请先创建 PV 或切换为动态制备模式</span>
                </div>
              </ElFormItem>

              <!-- 标签选择器 -->
              <ElFormItem
                v-if="formData.pvSelectMode === 'selector'"
                label="标签选择器"
                prop="selector"
              >
                <ElInput
                  v-model="formData.selector"
                  type="textarea"
                  :rows="4"
                  placeholder="用于匹配 PV 的标签选择器，YAML 格式：&#10;matchLabels:&#10;  environment: production&#10;  zone: cn-north-1"
                  :disabled="dialogType === 'edit'"
                />
                <div class="info-tip" style="margin-top: 8px">
                  <Info :size="12" />
                  <span>PVC 将自动绑定到匹配此选择器的第一个可用 PV</span>
                </div>
              </ElFormItem>

              <!-- 当选择了 PV 后，自动填充这些字段并设为只读 -->
              <ElFormItem label="存储容量" prop="requestStorage" required>
                <ElInput
                  v-model="formData.requestStorage"
                  placeholder="将自动从所选 PV 获取"
                  :disabled="true"
                >
                  <template #suffix>
                    <ElTooltip content="静态绑定时容量由 PV 决定" placement="top">
                      <HelpCircle :size="13" class="help-icon" />
                    </ElTooltip>
                  </template>
                </ElInput>
              </ElFormItem>

              <ElFormItem label="访问模式" prop="accessModes" required>
                <ElCheckboxGroup v-model="formData.accessModes" :disabled="true">
                  <ElCheckbox value="ReadWriteOnce">ReadWriteOnce</ElCheckbox>
                  <ElCheckbox value="ReadOnlyMany">ReadOnlyMany</ElCheckbox>
                  <ElCheckbox value="ReadWriteMany">ReadWriteMany</ElCheckbox>
                  <ElCheckbox value="ReadWriteOncePod">ReadWriteOncePod</ElCheckbox>
                </ElCheckboxGroup>
                <div class="info-tip" style="margin-top: 8px">
                  <Info :size="12" />
                  <span>静态绑定时访问模式由所选 PV 决定</span>
                </div>
              </ElFormItem>

              <ElFormItem label="卷模式" prop="volumeMode">
                <ElRadioGroup v-model="formData.volumeMode" :disabled="dialogType === 'edit'">
                  <ElRadio value="Filesystem">Filesystem</ElRadio>
                  <ElRadio value="Block">Block</ElRadio>
                </ElRadioGroup>
              </ElFormItem>
            </div>
          </ElCollapseItem>

          <!-- 高级选项 -->
          <ElCollapseItem name="advanced" class="collapse-section-item">
            <template #title>
              <div class="collapse-title">
                <ElIcon class="title-icon"><Setting /></ElIcon>
                <span>高级选项（可选）</span>
              </div>
            </template>
            <div class="collapse-content">
              <ElFormItem label="数据源">
                <ElRadioGroup v-model="formData.dataSourceType" :disabled="dialogType === 'edit'">
                  <ElRadio value="none">无</ElRadio>
                  <ElRadio value="pvc">
                    <div class="radio-option-content">
                      <span>从 PVC 克隆</span>
                      <ElTooltip content="从现有 PVC 克隆数据" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                  <ElRadio value="snapshot">
                    <div class="radio-option-content">
                      <span>从快照恢复</span>
                      <ElTooltip content="从 VolumeSnapshot 恢复数据" placement="top">
                        <HelpCircle :size="11" class="help-icon" />
                      </ElTooltip>
                    </div>
                  </ElRadio>
                </ElRadioGroup>
              </ElFormItem>

              <ElFormItem
                v-if="formData.dataSourceType === 'pvc'"
                label="源 PVC"
                prop="dataSourcePVC"
              >
                <ElInput
                  v-model="formData.dataSourcePVC"
                  placeholder="输入源 PVC 名称（需在同一命名空间）"
                  :disabled="dialogType === 'edit'"
                />
              </ElFormItem>

              <ElFormItem
                v-if="formData.dataSourceType === 'snapshot'"
                label="快照名称"
                prop="dataSourceSnapshot"
              >
                <ElInput
                  v-model="formData.dataSourceSnapshot"
                  placeholder="输入 VolumeSnapshot 名称"
                  :disabled="dialogType === 'edit'"
                />
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
                      <ElInput v-model="row.value" placeholder="注解值" />
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
    <div v-else-if="editMode === 'yaml'" class="yaml-content">
      <ArtYamlEditor v-model="yamlContent" height="600px" @change="handleYamlChange" />
    </div>

    <!-- 底部按钮 -->
    <template #footer>
      <ElButton @click="handleCancel">取消</ElButton>
      <ElButton type="primary" :loading="submitting" @click="handleSubmit">
        {{ dialogType === 'add' ? '创建' : '更新' }}
      </ElButton>
    </template>
  </ElDialog>
</template>

<script setup lang="ts">
  import { ref, watch, nextTick, computed } from 'vue'
  import { ElMessage, ElIcon } from 'element-plus'
  import { InfoFilled, Coin, Setting, PriceTag } from '@element-plus/icons-vue'
  import yaml from 'js-yaml'
  import {
    Database,
    HardDrive,
    Settings,
    Tag,
    FileText,
    FormInput,
    Code,
    Plus,
    Trash2,
    HelpCircle,
    Info,
    AlertCircle
  } from 'lucide-vue-next'
  import {
    createPVCApi,
    updatePVCApi,
    getPVCYamlApi,
    type PVCListItem,
    type ProjectWorkspace
  } from '@/api/workload/core'
  import {
    getStorageClassListApi,
    getPVListApi,
    type StorageClassListItem,
    type PVListItem
  } from '@/api/workload/cluster-resource'
  import ArtYamlEditor from '@/components/yaml-editor-pro/index.vue'
  import { createDns1123LabelRules } from '../../utils/k8s-validator'

  interface Props {
    visible: boolean
    dialogType: 'add' | 'edit'
    pvcData?: PVCListItem
    workspace: ProjectWorkspace | null
  }

  interface KeyValue {
    key: string
    value: string
  }

  const props = withDefaults(defineProps<Props>(), {
    pvcData: undefined
  })

  const emit = defineEmits(['close', 'success'])

  const formRef = ref()
  const editMode = ref<'form' | 'yaml'>('form')
  const submitting = ref(false)
  const dataLoading = ref(false)
  const storageClassLoading = ref(false)
  const pvLoading = ref(false)
  const yamlContent = ref('')
  const canExpandVolume = ref(false)
  const activeNames = ref<string[]>(['basic', 'storage', 'pvselect', 'advanced', 'metadata'])

  // StorageClass 相关
  const storageClassList = ref<StorageClassListItem[]>([])
  const selectedStorageClass = computed(() => {
    if (!formData.value.storageClassName) {
      return storageClassList.value.find((sc) => sc.isDefault)
    }
    return storageClassList.value.find((sc) => sc.name === formData.value.storageClassName)
  })

  // PV 相关
  const availablePVList = ref<PVListItem[]>([])
  const selectedPV = computed(() => {
    if (!formData.value.selectedPV) return null
    return availablePVList.value.find((pv) => pv.name === formData.value.selectedPV)
  })

  const formData = ref({
    name: '',
    bindingMode: 'dynamic' as 'dynamic' | 'static', // 绑定模式
    storageClassName: '', // 动态制备时使用
    pvSelectMode: 'name' as 'name' | 'selector', // 静态绑定的选择方式
    selectedPV: '', // 静态绑定 - 指定 PV 名称
    selector: '', // 静态绑定 - 标签选择器
    requestStorage: '10Gi',
    accessModes: ['ReadWriteOnce'] as string[],
    volumeMode: 'Filesystem',
    dataSourceType: 'none',
    dataSourcePVC: '',
    dataSourceSnapshot: '',
    labels: [] as KeyValue[],
    annotations: [] as KeyValue[]
  })

  const formRules = {
    name: createDns1123LabelRules(true, 'PVC 名称'),
    requestStorage: [
      { required: true, message: '请输入存储容量', trigger: 'blur' },
      {
        pattern: /^\d+(\.\d+)?(Ki|Mi|Gi|Ti|Pi|K|M|G|T|P)$/,
        message: '格式不正确，例如: 10Gi, 500Mi',
        trigger: 'blur'
      }
    ],
    accessModes: [
      {
        type: 'array',
        required: true,
        message: '请至少选择一个访问模式',
        trigger: 'change'
      }
    ],
    selectedPV: [
      {
        validator: (_rule: any, value: any, callback: any) => {
          if (
            formData.value.bindingMode === 'static' &&
            formData.value.pvSelectMode === 'name' &&
            !value
          ) {
            callback(new Error('请选择一个 PV'))
          } else {
            callback()
          }
        },
        trigger: 'change'
      }
    ]
  }

  // 加载 StorageClass 列表
  const loadStorageClasses = async () => {
    if (!props.workspace?.clusterUuid) return

    storageClassLoading.value = true
    try {
      const response = await getStorageClassListApi({
        clusterUuid: props.workspace.clusterUuid
      })

      if (response && response.items) {
        storageClassList.value = response.items
      } else if (Array.isArray(response)) {
        storageClassList.value = response
      }
    } catch (error) {
      console.error('加载 StorageClass 列表失败:', error)
      storageClassList.value = []
    } finally {
      storageClassLoading.value = false
    }
  }

  // 加载可用的 PV 列表
  const loadAvailablePVs = async () => {
    if (!props.workspace?.clusterUuid) return

    pvLoading.value = true
    try {
      const response = await getPVListApi({
        clusterUuid: props.workspace.clusterUuid,
        status: 'Available' // 只获取可用的 PV
      })

      if (response && response.items) {
        availablePVList.value = response.items
      } else if (Array.isArray(response)) {
        availablePVList.value = response
      }
    } catch (error) {
      console.error('加载 PV 列表失败:', error)
      availablePVList.value = []
    } finally {
      pvLoading.value = false
    }
  }

  // StorageClass 变化处理
  const handleStorageClassChange = (value: string) => {
    const sc = storageClassList.value.find((item) => item.name === value)
    if (sc) {
      canExpandVolume.value = sc.allowVolumeExpansion
    } else {
      // 使用默认 StorageClass 的配置
      const defaultSC = storageClassList.value.find((item) => item.isDefault)
      canExpandVolume.value = defaultSC?.allowVolumeExpansion || false
    }
  }

  // PV 变化处理
  const handlePVChange = (value: string) => {
    const pv = availablePVList.value.find((item) => item.name === value)
    if (pv) {
      // 自动填充 PV 的相关信息
      formData.value.requestStorage = pv.capacity
      const modes = pv.accessModes.split(',').map((m: string) => m.trim())
      formData.value.accessModes = modes
    }
  }

  const handleClose = (val: boolean) => {
    if (!val) {
      emit('close')
    }
  }

  // Labels 管理
  const addLabel = () => {
    formData.value.labels.push({ key: '', value: '' })
  }

  const removeLabel = (index: number) => {
    formData.value.labels.splice(index, 1)
  }

  // Annotations 管理
  const addAnnotation = () => {
    formData.value.annotations.push({ key: '', value: '' })
  }

  const removeAnnotation = (index: number) => {
    formData.value.annotations.splice(index, 1)
  }

  // 工具函数
  const objectToArray = (obj?: Record<string, string>): KeyValue[] => {
    if (!obj) return []
    return Object.entries(obj)
      .filter(([key]) => key)
      .map(([key, value]) => ({ key, value }))
  }

  const arrayToObject = (arr: KeyValue[]): Record<string, string> => {
    const obj: Record<string, string> = {}
    arr.forEach((item) => {
      if (item.key && item.key.trim()) {
        obj[item.key.trim()] = item.value
      }
    })
    return obj
  }

  // 表单转 YAML
  const formToYaml = () => {
    const pvcObj: any = {
      apiVersion: 'v1',
      kind: 'PersistentVolumeClaim',
      metadata: {
        name: formData.value.name || 'pvc-name',
        namespace: props.workspace?.namespace || 'default'
      },
      spec: {
        accessModes: formData.value.accessModes,
        resources: {
          requests: {
            storage: formData.value.requestStorage || '10Gi'
          }
        }
      }
    }

    // 动态制备模式
    if (formData.value.bindingMode === 'dynamic') {
      // StorageClass（可选）
      if (formData.value.storageClassName) {
        pvcObj.spec.storageClassName = formData.value.storageClassName
      }
    }

    // 静态绑定模式
    if (formData.value.bindingMode === 'static') {
      if (formData.value.pvSelectMode === 'name' && formData.value.selectedPV) {
        // 指定 PV 名称
        pvcObj.spec.volumeName = formData.value.selectedPV
      } else if (formData.value.pvSelectMode === 'selector' && formData.value.selector) {
        // 标签选择器
        try {
          const selectorObj = yaml.load(formData.value.selector)
          pvcObj.spec.selector = selectorObj
        } catch (error) {
          console.error('标签选择器解析失败:', error)
        }
      }
    }

    // VolumeMode
    if (formData.value.volumeMode) {
      pvcObj.spec.volumeMode = formData.value.volumeMode
    }

    // DataSource
    if (formData.value.dataSourceType === 'pvc' && formData.value.dataSourcePVC) {
      pvcObj.spec.dataSource = {
        kind: 'PersistentVolumeClaim',
        name: formData.value.dataSourcePVC
      }
    } else if (formData.value.dataSourceType === 'snapshot' && formData.value.dataSourceSnapshot) {
      pvcObj.spec.dataSource = {
        kind: 'VolumeSnapshot',
        apiGroup: 'snapshot.storage.k8s.io',
        name: formData.value.dataSourceSnapshot
      }
    }

    // Labels
    const labelsObj = arrayToObject(formData.value.labels)
    if (Object.keys(labelsObj).length > 0) {
      pvcObj.metadata.labels = labelsObj
    }

    // Annotations
    const annotationsObj = arrayToObject(formData.value.annotations)
    if (Object.keys(annotationsObj).length > 0) {
      pvcObj.metadata.annotations = annotationsObj
    }

    yamlContent.value = yaml.dump(pvcObj, { indent: 2 })
  }

  // YAML 转表单
  const yamlToForm = (yamlStr: string) => {
    try {
      const obj = yaml.load(yamlStr) as any

      formData.value.name = obj.metadata?.name || ''
      formData.value.requestStorage = obj.spec?.resources?.requests?.storage || '10Gi'
      formData.value.accessModes = obj.spec?.accessModes || ['ReadWriteOnce']
      formData.value.volumeMode = obj.spec?.volumeMode || 'Filesystem'

      // 判断绑定模式
      if (obj.spec?.volumeName) {
        // 静态绑定 - 指定 PV 名称
        formData.value.bindingMode = 'static'
        formData.value.pvSelectMode = 'name'
        formData.value.selectedPV = obj.spec.volumeName
        formData.value.storageClassName = ''
      } else if (obj.spec?.selector) {
        // 静态绑定 - 标签选择器
        formData.value.bindingMode = 'static'
        formData.value.pvSelectMode = 'selector'
        formData.value.selector = yaml.dump(obj.spec.selector, { indent: 2 })
        formData.value.storageClassName = ''
      } else {
        // 动态制备
        formData.value.bindingMode = 'dynamic'
        formData.value.storageClassName = obj.spec?.storageClassName || ''
      }

      // DataSource
      if (obj.spec?.dataSource) {
        if (obj.spec.dataSource.kind === 'PersistentVolumeClaim') {
          formData.value.dataSourceType = 'pvc'
          formData.value.dataSourcePVC = obj.spec.dataSource.name || ''
        } else if (obj.spec.dataSource.kind === 'VolumeSnapshot') {
          formData.value.dataSourceType = 'snapshot'
          formData.value.dataSourceSnapshot = obj.spec.dataSource.name || ''
        }
      } else {
        formData.value.dataSourceType = 'none'
      }

      // Labels
      formData.value.labels = objectToArray(obj.metadata?.labels)

      // Annotations
      formData.value.annotations = objectToArray(obj.metadata?.annotations)

      // 检查是否支持扩容（仅动态制备时）
      if (formData.value.bindingMode === 'dynamic') {
        const sc = storageClassList.value.find((s) => s.name === formData.value.storageClassName)
        canExpandVolume.value = sc?.allowVolumeExpansion || false
      }
    } catch (error) {
      console.error('YAML 解析失败:', error)
      throw error
    }
  }

  // 加载 YAML
  const loadPVCYAML = async () => {
    if (!props.workspace || !props.pvcData) {
      return
    }

    dataLoading.value = true
    try {
      const yamlStr = await getPVCYamlApi({
        clusterUuid: props.workspace.clusterUuid!,
        namespace: props.workspace.namespace!,
        name: props.pvcData.name
      })

      yamlContent.value = yamlStr
      yamlToForm(yamlStr)
    } catch (error: any) {
      console.error('加载失败:', error)
      emit('close')
    } finally {
      dataLoading.value = false
    }
  }

  // 监听对话框打开
  watch(
    () => props.visible,
    async (val) => {
      if (val) {
        if (props.dialogType === 'edit' && props.pvcData && props.workspace) {
          // 编辑模式：先加载 StorageClass 列表，再加载 PVC 数据
          await loadStorageClasses()
          await loadPVCYAML()
        } else if (props.dialogType === 'add') {
          // 新建模式：加载 StorageClass 和 PV 列表
          await loadStorageClasses()
          await loadAvailablePVs()
          await nextTick()
          formToYaml()
        }
      }
    },
    { immediate: true }
  )

  // 监听绑定模式变化
  watch(
    () => formData.value.bindingMode,
    (newMode) => {
      if (newMode === 'static' && availablePVList.value.length === 0) {
        loadAvailablePVs()
      }
      // 切换模式时清空相关字段
      if (newMode === 'dynamic') {
        formData.value.selectedPV = ''
        formData.value.selector = ''
      } else {
        formData.value.storageClassName = ''
      }
    }
  )

  // 监听编辑模式切换
  watch(editMode, (newMode, oldMode) => {
    if (dataLoading.value) return

    if (newMode === 'yaml' && oldMode === 'form') {
      formToYaml()
    } else if (newMode === 'form' && oldMode === 'yaml') {
      try {
        yamlToForm(yamlContent.value)
        ElMessage.success('已切换到表单模式')
      } catch {
        ElMessage.error('YAML 格式错误，无法切换到表单模式')
        nextTick(() => {
          editMode.value = 'yaml'
        })
      }
    }
  })

  // 监听表单数据变化
  watch(
    () => formData.value,
    () => {
      if (editMode.value === 'form' && !dataLoading.value) {
        formToYaml()
      }
    },
    { deep: true }
  )

  const handleYamlChange = (content: string) => {
    yamlContent.value = content
  }

  // 提交
  const handleSubmit = async () => {
    if (!props.workspace) {
      return
    }

    if (editMode.value === 'form') {
      try {
        await formRef.value?.validate()
      } catch {
        return
      }

      formToYaml()
    }

    submitting.value = true
    try {
      if (props.dialogType === 'add') {
        await createPVCApi({
          clusterUuid: props.workspace.clusterUuid!,
          namespace: props.workspace.namespace!,
          yamlStr: yamlContent.value
        })
        ElMessage.success('创建成功')
      } else {
        await updatePVCApi({
          clusterUuid: props.workspace.clusterUuid!,
          namespace: props.workspace.namespace!,
          name: formData.value.name,
          yamlStr: yamlContent.value
        })
        ElMessage.success('更新成功')
      }

      emit('success')
      emit('close')
    } finally {
      submitting.value = false
    }
  }

  const handleCancel = () => {
    emit('close')
  }

  const handleClosed = () => {
    formRef.value?.resetFields()
    formData.value = {
      name: '',
      bindingMode: 'dynamic',
      storageClassName: '',
      pvSelectMode: 'name',
      selectedPV: '',
      selector: '',
      requestStorage: '10Gi',
      accessModes: ['ReadWriteOnce'],
      volumeMode: 'Filesystem',
      dataSourceType: 'none',
      dataSourcePVC: '',
      dataSourceSnapshot: '',
      labels: [],
      annotations: []
    }
    yamlContent.value = ''
    editMode.value = 'form'
    canExpandVolume.value = false
    storageClassList.value = []
    availablePVList.value = []
  }
</script>

<style lang="scss" scoped>
  .mode-switch {
    margin-bottom: 16px;
    display: flex;
    justify-content: flex-end;

    .radio-content {
      display: flex;
      align-items: center;
      gap: 4px;
      font-size: 12px;
    }
  }

  .pvc-form-container {
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

  .pvc-form {
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

  .help-icon {
    cursor: help;
    color: #909399;
    transition: color 0.2s;

    &:hover {
      color: #606266;
    }
  }

  .sc-option {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 0;

    .sc-option-main {
      display: flex;
      align-items: center;
      gap: 8px;

      .sc-name {
        font-weight: 500;
        color: #303133;
      }
    }

    .sc-option-sub {
      display: flex;
      align-items: center;
      gap: 8px;
      font-size: 12px;
      color: #909399;

      .sc-provisioner {
        font-family: var(--ax-font-mono);
      }
    }
  }

  .info-tip {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 8px;
    padding: 8px 12px;
    background: #f0f9ff;
    border: 1px solid #bfdbfe;
    border-radius: 4px;
    font-size: 12px;
    color: #1e40af;

    svg {
      flex-shrink: 0;
      color: #3b82f6;
    }
  }

  .warning-tip {
    display: flex;
    align-items: center;
    gap: 4px;
    margin-top: 8px;
    padding: 8px 12px;
    background: #fef3c7;
    border: 1px solid #fde68a;
    border-radius: 4px;
    font-size: 12px;
    color: #92400e;

    svg {
      flex-shrink: 0;
      color: #f59e0b;
    }
  }

  .pv-option {
    display: flex;
    flex-direction: column;
    gap: 4px;
    padding: 4px 0;

    .pv-option-main {
      display: flex;
      align-items: center;
      gap: 8px;

      .pv-name {
        font-weight: 500;
        color: #303133;
        font-family: var(--ax-font-mono);
      }
    }

    .pv-option-sub {
      display: flex;
      align-items: center;
      gap: 12px;
      font-size: 12px;
      color: #909399;
    }
  }

  .labels-list,
  .annotations-list {
    display: flex;
    flex-direction: column;
    gap: 8px;
    margin-bottom: 12px;
  }

  .label-row,
  .annotation-row {
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 12px;
    background: white;
    border-radius: 4px;
    border: 1px solid #dcdfe6;
    transition: border-color 0.2s;

    &:hover {
      border-color: #409eff;
    }

    .separator {
      color: #909399;
      font-weight: 500;
      flex-shrink: 0;
    }
  }

  .yaml-content {
    min-height: 500px;
  }

  :deep(.el-form-item) {
    margin-bottom: 16px;
  }

  :deep(.el-form-item__label) {
    font-size: 13px;
    font-weight: 500;
    color: #303133;
  }

  :deep(.el-checkbox) {
    margin-right: 16px;
    margin-bottom: 8px;
  }

  :deep(.el-radio) {
    margin-right: 16px;
    margin-bottom: 8px;
  }

  :deep(.el-select-dropdown__item) {
    height: auto;
    padding: 8px 12px;
  }

  .radio-option-content {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }

  .checkbox-option-content {
    display: inline-flex;
    align-items: center;
    gap: 4px;
  }
</style>
