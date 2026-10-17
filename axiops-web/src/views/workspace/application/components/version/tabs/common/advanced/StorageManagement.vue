<template>
  <div class="storage-management">
    <!-- 顶部操作栏 -->
    <div class="toolbar">
      <div class="toolbar-left">
        <ElRadioGroup v-model="activeTab" size="small">
          <ElRadioButton value="volumes">
            存储卷 <span v-if="volumes.length" class="count">({{ volumes.length }})</span>
          </ElRadioButton>
          <ElRadioButton value="mounts">
            挂载配置 <span v-if="totalMountsCount" class="count">({{ totalMountsCount }})</span>
          </ElRadioButton>
        </ElRadioGroup>
      </div>

      <div class="toolbar-actions">
        <ElButton v-if="!editing" type="primary" size="small" @click="startEdit">
          <Edit :size="14" />
          编辑
        </ElButton>
        <template v-else>
          <ElButton type="primary" size="small" :loading="saving" @click="handleSave">
            <Save :size="14" />
            保存
          </ElButton>
          <ElButton size="small" @click="handleCancel">取消</ElButton>
        </template>
        <!-- 刷新按钮 -->
        <ElButton
          size="small"
          :loading="loading"
          :disabled="editing"
          @click="handleRefresh"
          title="刷新数据"
        >
          <RefreshCw :size="14" />
        </ElButton>
      </div>
    </div>

    <!-- 加载状态 -->
    <div v-if="loading" v-loading="loading" class="loading-wrapper"></div>

    <!-- 主内容 -->
    <div v-else class="config-wrapper">
      <!-- 存储卷配置 -->
      <div v-show="activeTab === 'volumes'" class="config-section">
        <div class="section-header">
          <span class="section-title">存储卷列表</span>
          <ElButton v-if="editing" type="primary" size="small" @click="handleAddVolume">
            <Plus :size="14" />
            添加存储卷
          </ElButton>
        </div>

        <ElTable :data="volumes" border size="small" class="data-table">
          <ElTableColumn prop="name" label="卷名称" width="150">
            <template #default="{ row }">
              <ElInput v-model="row.name" placeholder="卷名称" :disabled="!editing" size="small" />
            </template>
          </ElTableColumn>

          <ElTableColumn prop="type" label="类型" width="130">
            <template #default="{ row }">
              <ElSelect
                v-model="row.type"
                placeholder="类型"
                :disabled="!editing"
                size="small"
                @change="handleVolumeTypeChange(row)"
              >
                <ElOption label="EmptyDir" value="emptyDir" />
                <ElOption label="HostPath" value="hostPath" />
                <ElOption label="ConfigMap" value="configMap" />
                <ElOption label="Secret" value="secret" />
                <ElOption label="PVC" value="persistentVolumeClaim" />
                <ElOption label="NFS" value="nfs" />
              </ElSelect>
            </template>
          </ElTableColumn>

          <ElTableColumn label="配置详情" min-width="300">
            <template #default="{ row }">
              <!-- EmptyDir -->
              <div v-if="row.type === 'emptyDir'" class="config-detail">
                <ElInput
                  v-model="row.emptyDir.sizeLimit"
                  placeholder="容量限制（如: 1Gi）"
                  :disabled="!editing"
                  size="small"
                  style="width: 150px"
                />
                <ElSelect
                  v-model="row.emptyDir.medium"
                  placeholder="介质"
                  :disabled="!editing"
                  size="small"
                  style="width: 100px"
                  clearable
                >
                  <ElOption label="默认" value="" />
                  <ElOption label="Memory" value="Memory" />
                </ElSelect>
              </div>

              <!-- HostPath -->
              <div v-else-if="row.type === 'hostPath'" class="config-detail">
                <ElInput
                  v-model="row.hostPath.path"
                  placeholder="主机路径"
                  :disabled="!editing"
                  size="small"
                  style="flex: 1"
                />
                <ElSelect
                  v-model="row.hostPath.type"
                  placeholder="类型"
                  :disabled="!editing"
                  size="small"
                  style="width: 140px"
                  clearable
                >
                  <ElOption label="Directory" value="Directory" />
                  <ElOption label="DirectoryOrCreate" value="DirectoryOrCreate" />
                  <ElOption label="File" value="File" />
                  <ElOption label="FileOrCreate" value="FileOrCreate" />
                </ElSelect>
              </div>

              <!-- ConfigMap - 远程选择 -->
              <div v-else-if="row.type === 'configMap'" class="config-detail">
                <ElSelect
                  v-model="row.configMap.name"
                  placeholder="选择 ConfigMap"
                  :disabled="!editing"
                  size="small"
                  style="flex: 1"
                  filterable
                  :loading="loadingConfigMaps"
                  @focus="loadConfigMapList"
                >
                  <ElOption
                    v-for="cm in configMapList"
                    :key="cm.name"
                    :label="cm.name"
                    :value="cm.name"
                  >
                    <div class="resource-option">
                      <span>{{ cm.name }}</span>
                      <span class="option-meta">{{ cm.dataCount }} 项</span>
                    </div>
                  </ElOption>
                </ElSelect>
              </div>

              <!-- Secret - 远程选择 -->
              <div v-else-if="row.type === 'secret'" class="config-detail">
                <ElSelect
                  v-model="row.secret.secretName"
                  placeholder="选择 Secret"
                  :disabled="!editing"
                  size="small"
                  style="flex: 1"
                  filterable
                  :loading="loadingSecrets"
                  @focus="loadSecretList"
                >
                  <ElOption v-for="s in secretList" :key="s.name" :label="s.name" :value="s.name">
                    <div class="resource-option">
                      <span>{{ s.name }}</span>
                      <ElTag size="small" type="info">{{ s.type }}</ElTag>
                    </div>
                  </ElOption>
                </ElSelect>
              </div>

              <!-- PVC - 远程选择 -->
              <div v-else-if="row.type === 'persistentVolumeClaim'" class="config-detail">
                <ElSelect
                  v-model="row.persistentVolumeClaim.claimName"
                  placeholder="选择 PVC"
                  :disabled="!editing"
                  size="small"
                  style="flex: 1"
                  filterable
                  :loading="loadingPVCs"
                  @focus="loadPVCList"
                >
                  <ElOption
                    v-for="pvc in pvcList"
                    :key="pvc.name"
                    :label="pvc.name"
                    :value="pvc.name"
                  >
                    <div class="resource-option">
                      <span>{{ pvc.name }}</span>
                      <span class="option-meta">{{ pvc.capacity }} | {{ pvc.status }}</span>
                    </div>
                  </ElOption>
                </ElSelect>
                <ElCheckbox
                  v-model="row.persistentVolumeClaim.readOnly"
                  :disabled="!editing"
                  size="small"
                  >只读</ElCheckbox
                >
              </div>

              <!-- NFS -->
              <div v-else-if="row.type === 'nfs'" class="config-detail">
                <ElInput
                  v-model="row.nfs.server"
                  placeholder="NFS 服务器"
                  :disabled="!editing"
                  size="small"
                  style="width: 140px"
                />
                <ElInput
                  v-model="row.nfs.path"
                  placeholder="路径"
                  :disabled="!editing"
                  size="small"
                  style="flex: 1"
                />
              </div>

              <span v-else class="text-muted">请选择类型</span>
            </template>
          </ElTableColumn>

          <ElTableColumn v-if="editing" label="操作" width="60" align="center">
            <template #default="{ $index }">
              <ElButton size="small" text type="danger" @click="handleDeleteVolume($index)">
                <Trash2 :size="14" />
              </ElButton>
            </template>
          </ElTableColumn>

          <template #empty>
            <div class="empty-state">
              <span>暂无存储卷</span>
              <ElButton v-if="editing" type="primary" link size="small" @click="handleAddVolume">
                添加一个
              </ElButton>
            </div>
          </template>
        </ElTable>
      </div>

      <!-- 卷挂载配置 -->
      <div v-show="activeTab === 'mounts'" class="config-section">
        <div class="section-header">
          <div class="container-selector">
            <span class="selector-label">容器：</span>
            <ElSelect
              v-model="selectedContainer"
              placeholder="选择容器"
              style="width: 200px"
              size="small"
            >
              <ElOptionGroup v-if="initContainers.length > 0" label="Init 容器">
                <ElOption
                  v-for="c in initContainers"
                  :key="c.containerName"
                  :label="c.containerName"
                  :value="c.containerName"
                >
                  <div class="container-option">
                    <span>{{ c.containerName }}</span>
                    <ElTag size="small" type="warning">Init</ElTag>
                  </div>
                </ElOption>
              </ElOptionGroup>
              <ElOptionGroup v-if="mainContainers.length > 0" label="Main 容器">
                <ElOption
                  v-for="c in mainContainers"
                  :key="c.containerName"
                  :label="c.containerName"
                  :value="c.containerName"
                >
                  <div class="container-option">
                    <span>{{ c.containerName }}</span>
                    <ElTag size="small" type="primary">Main</ElTag>
                  </div>
                </ElOption>
              </ElOptionGroup>
            </ElSelect>
            <ElTag
              v-if="currentContainerType"
              :type="currentContainerType === 'init' ? 'warning' : 'primary'"
              size="small"
            >
              {{ currentContainerType === 'init' ? 'Init 容器' : 'Main 容器' }}
            </ElTag>
          </div>
          <ElButton
            v-if="editing && selectedContainer"
            type="primary"
            size="small"
            :disabled="volumes.length === 0"
            @click="handleAddMount"
          >
            <Plus :size="14" />
            添加挂载
          </ElButton>
        </div>

        <ElTable
          v-if="selectedContainer"
          :data="currentMounts"
          border
          size="small"
          class="data-table"
        >
          <ElTableColumn prop="name" label="存储卷" width="150">
            <template #default="{ row }">
              <ElSelect
                v-model="row.name"
                placeholder="选择存储卷"
                :disabled="!editing"
                size="small"
              >
                <ElOption
                  v-for="vol in volumes"
                  :key="vol.name"
                  :label="vol.name"
                  :value="vol.name"
                >
                  <div class="volume-option">
                    <span>{{ vol.name }}</span>
                    <ElTag size="small" type="info">{{ vol.type }}</ElTag>
                  </div>
                </ElOption>
              </ElSelect>
            </template>
          </ElTableColumn>

          <ElTableColumn prop="mountPath" label="挂载路径" min-width="180">
            <template #default="{ row }">
              <ElInput
                v-model="row.mountPath"
                placeholder="容器内挂载路径"
                :disabled="!editing"
                size="small"
              />
            </template>
          </ElTableColumn>

          <ElTableColumn prop="subPath" label="子路径" width="140">
            <template #default="{ row }">
              <ElInput v-model="row.subPath" placeholder="可选" :disabled="!editing" size="small" />
            </template>
          </ElTableColumn>

          <ElTableColumn prop="readOnly" label="只读" width="60" align="center">
            <template #default="{ row }">
              <ElCheckbox v-model="row.readOnly" :disabled="!editing" size="small" />
            </template>
          </ElTableColumn>

          <ElTableColumn v-if="editing" label="操作" width="60" align="center">
            <template #default="{ $index }">
              <ElButton size="small" text type="danger" @click="handleDeleteMount($index)">
                <Trash2 :size="14" />
              </ElButton>
            </template>
          </ElTableColumn>

          <template #empty>
            <div class="empty-state">
              <span v-if="volumes.length === 0">请先添加存储卷</span>
              <template v-else>
                <span>暂无挂载配置</span>
                <ElButton v-if="editing" type="primary" link size="small" @click="handleAddMount">
                  添加一个
                </ElButton>
              </template>
            </div>
          </template>
        </ElTable>

        <ElEmpty v-else description="请选择一个容器" :image-size="60" />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { ElMessage, ElMessageBox } from 'element-plus'
  import { Edit, Save, Plus, Trash2, RefreshCw } from 'lucide-vue-next'
  import {
    getStorageConfigApi,
    updateStorageConfigApi,
    getConfigMapListApi,
    getSecretListApi,
    getPVCListApi,
    type OnecProjectVersion,
    type ProjectWorkspace,
    type ProjectCluster,
    type VolumeConfig,
    type VolumeMountConfig,
    type ConfigMapListItem,
    type SecretListItem,
    type PVCListItem
  } from '@/api'

  interface Props {
    version: OnecProjectVersion
    workspace: ProjectWorkspace | null
    cluster: ProjectCluster | null
    refreshTrigger?: number
  }

  const props = defineProps<Props>()
  const emit = defineEmits<{ success: [] }>()

  // ==================== 状态 ====================
  const loading = ref(false)
  const saving = ref(false)
  const editing = ref(false)
  const activeTab = ref('volumes')

  const volumes = ref<VolumeConfig[]>([])
  const volumeMounts = ref<VolumeMountConfig[]>([])
  const selectedContainer = ref('')
  const originalData = ref<any>(null)

  // 远程资源列表
  const configMapList = ref<ConfigMapListItem[]>([])
  const secretList = ref<SecretListItem[]>([])
  const pvcList = ref<PVCListItem[]>([])
  const loadingConfigMaps = ref(false)
  const loadingSecrets = ref(false)
  const loadingPVCs = ref(false)

  // 缓存标记，避免重复加载
  const configMapsLoaded = ref(false)
  const secretsLoaded = ref(false)
  const pvcsLoaded = ref(false)

  // ==================== 计算属性 ====================
  const workloadId = computed(() => props.workspace?.id || 0)
  const clusterUuid = computed(() => props.cluster?.uuid || '')
  const namespace = computed(() => props.workspace?.namespace || '')

  const initContainers = computed(() =>
    volumeMounts.value.filter((c) => c.containerType === 'init')
  )
  const mainContainers = computed(() =>
    volumeMounts.value.filter((c) => c.containerType === 'main')
  )

  const currentContainerType = computed(() => {
    const container = volumeMounts.value.find((c) => c.containerName === selectedContainer.value)
    return container?.containerType || ''
  })

  const currentMounts = computed(() => {
    const container = volumeMounts.value.find((c) => c.containerName === selectedContainer.value)
    return container?.mounts || []
  })

  const totalMountsCount = computed(() => {
    return volumeMounts.value.reduce((sum, vm) => sum + (vm.mounts?.length || 0), 0)
  })

  // ==================== 远程资源加载 ====================
  const loadConfigMapList = async () => {
    if (configMapsLoaded.value || !workloadId.value) return
    // 防止并发加载
    if (loadingConfigMaps.value) return

    loadingConfigMaps.value = true
    try {
      const response = await getConfigMapListApi({ workloadId: workloadId.value })
      configMapList.value = response.items || []
      configMapsLoaded.value = true
    } catch (error) {
      console.error('[存储管理] 加载 ConfigMap 列表失败:', error)
    } finally {
      loadingConfigMaps.value = false
    }
  }

  const loadSecretList = async () => {
    if (secretsLoaded.value || !workloadId.value) return
    // 防止并发加载
    if (loadingSecrets.value) return

    loadingSecrets.value = true
    try {
      const response = await getSecretListApi({ workloadId: workloadId.value })
      secretList.value = response.items || []
      secretsLoaded.value = true
    } catch (error) {
      console.error('[存储管理] 加载 Secret 列表失败:', error)
    } finally {
      loadingSecrets.value = false
    }
  }

  const loadPVCList = async () => {
    if (pvcsLoaded.value || !clusterUuid.value || !namespace.value) return
    // 防止并发加载
    if (loadingPVCs.value) return

    loadingPVCs.value = true
    try {
      const response = await getPVCListApi({
        clusterUuid: clusterUuid.value,
        namespace: namespace.value
      })
      pvcList.value = response.items || []
      pvcsLoaded.value = true
    } catch (error) {
      console.error('[存储管理] 加载 PVC 列表失败:', error)
    } finally {
      loadingPVCs.value = false
    }
  }

  // ==================== 核心方法 ====================
  const loadConfig = async () => {
    // 防止重复加载（首次加载时 onMounted 和 watch 可能同时触发）
    if (loading.value) return

    loading.value = true
    try {
      const response = await getStorageConfigApi(props.version.id)
      volumes.value = response.volumes || []
      volumeMounts.value = response.volumeMounts || []

      // 默认选中第一个容器
      if (volumeMounts.value.length > 0 && !selectedContainer.value) {
        const firstMain = mainContainers.value[0]
        const firstInit = initContainers.value[0]
        selectedContainer.value = firstMain?.containerName || firstInit?.containerName || ''
      }

      // 备份原始数据
      originalData.value = {
        volumes: JSON.parse(JSON.stringify(volumes.value)),
        volumeMounts: JSON.parse(JSON.stringify(volumeMounts.value))
      }
    } catch (error) {
      console.error('[存储管理] 加载失败:', error)
      ElMessage.error('加载存储配置失败')
    } finally {
      loading.value = false
    }
  }

  const startEdit = () => {
    editing.value = true
  }

  const handleCancel = () => {
    if (originalData.value) {
      volumes.value = JSON.parse(JSON.stringify(originalData.value.volumes))
      volumeMounts.value = JSON.parse(JSON.stringify(originalData.value.volumeMounts))
    }
    editing.value = false
    ElMessage.info('已取消编辑')
  }

  /** 刷新数据 */
  const handleRefresh = async () => {
    // 重置缓存
    configMapsLoaded.value = false
    secretsLoaded.value = false
    pvcsLoaded.value = false
    await loadConfig()
    ElMessage.success('刷新成功')
  }

  const handleSave = async () => {
    try {
      await ElMessageBox.confirm('确定要保存存储配置吗？', '确认保存', {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      })

      saving.value = true

      await updateStorageConfigApi(props.version.id, {
        volumes: volumes.value,
        volumeMounts: volumeMounts.value
      })

      ElMessage.success('存储配置保存成功')
      editing.value = false
      await loadConfig()
      emit('success')
    } catch (error: any) {
      if (error !== 'cancel') {
        console.error('[存储管理] 保存失败:', error)
        ElMessage.error('保存失败：' + (error.message || '未知错误'))
      }
    } finally {
      saving.value = false
    }
  }

  // 存储卷操作
  const handleAddVolume = () => {
    volumes.value.push({
      name: '',
      type: 'emptyDir',
      emptyDir: { sizeLimit: '', medium: '' }
    })
  }

  const handleDeleteVolume = (index: number) => {
    volumes.value.splice(index, 1)
  }

  const handleVolumeTypeChange = (row: VolumeConfig) => {
    const type = row.type

    // 重置配置
    Object.keys(row).forEach((key) => {
      if (key !== 'name' && key !== 'type') {
        delete (row as any)[key]
      }
    })

    // 初始化对应类型的默认配置
    switch (type) {
      case 'emptyDir':
        row.emptyDir = { sizeLimit: '', medium: '' }
        break
      case 'hostPath':
        row.hostPath = { path: '', type: '' }
        break
      case 'configMap':
        row.configMap = { name: '' }
        break
      case 'secret':
        row.secret = { secretName: '' }
        break
      case 'persistentVolumeClaim':
        row.persistentVolumeClaim = { claimName: '', readOnly: false }
        break
      case 'nfs':
        row.nfs = { server: '', path: '', readOnly: false }
        break
    }
  }

  // 挂载操作
  const handleAddMount = () => {
    const container = volumeMounts.value.find((c) => c.containerName === selectedContainer.value)
    if (container) {
      if (!container.mounts) {
        container.mounts = []
      }
      container.mounts.push({
        name: '',
        mountPath: '',
        subPath: '',
        readOnly: false
      })
    }
  }

  const handleDeleteMount = (index: number) => {
    const container = volumeMounts.value.find((c) => c.containerName === selectedContainer.value)
    if (container && container.mounts) {
      container.mounts.splice(index, 1)
    }
  }

  // ==================== 生命周期 ====================
  watch(
    () => props.refreshTrigger,
    (newVal) => {
      if (newVal && newVal > 0) {
        editing.value = false
        // 重置缓存
        configMapsLoaded.value = false
        secretsLoaded.value = false
        pvcsLoaded.value = false
        loadConfig()
      }
    }
  )

  onMounted(() => {
    loadConfig()
  })

  defineExpose({
    editing,
    hasUnsavedChanges: () => {
      if (!editing.value || !originalData.value) return false
      const current = JSON.stringify({ volumes: volumes.value, volumeMounts: volumeMounts.value })
      const original = JSON.stringify(originalData.value)
      return current !== original
    }
  })
</script>

<style lang="scss" scoped>
  .storage-management {
    height: 100%;
    display: flex;
    flex-direction: column;
    background: #fff;

    .toolbar {
      display: flex;
      align-items: center;
      justify-content: space-between;
      padding: 12px 16px;
      border-bottom: 1px solid var(--el-border-color-lighter);

      .toolbar-left {
        .count {
          color: #909399;
          font-size: 12px;
        }

        :deep(.el-radio-group) {
          .el-radio-button__inner {
            padding: 8px 16px;
          }
        }
      }

      .toolbar-actions {
        display: flex;
        gap: 8px;

        :deep(.el-button) {
          display: inline-flex;
          align-items: center;
          gap: 4px;
        }
      }
    }

    .loading-wrapper {
      flex: 1;
      min-height: 150px;
    }

    .config-wrapper {
      flex: 1;
      padding: 16px;
      overflow-y: auto;

      .config-section {
        .section-header {
          display: flex;
          align-items: center;
          justify-content: space-between;
          margin-bottom: 12px;

          .section-title {
            font-size: 14px;
            font-weight: 500;
            color: #303133;
          }

          .container-selector {
            display: flex;
            align-items: center;
            gap: 10px;

            .selector-label {
              font-size: 13px;
              color: #606266;
            }
          }
        }

        .data-table {
          .config-detail {
            display: flex;
            align-items: center;
            gap: 8px;
          }

          .text-muted {
            color: #909399;
            font-size: 12px;
          }

          .empty-state {
            display: flex;
            align-items: center;
            justify-content: center;
            gap: 8px;
            color: #909399;
            font-size: 13px;
            padding: 10px 0;
          }
        }
      }
    }

    .container-option,
    .volume-option,
    .resource-option {
      display: flex;
      justify-content: space-between;
      align-items: center;
      width: 100%;

      .option-meta {
        font-size: 12px;
        color: #909399;
      }
    }

    :deep(.el-empty) {
      padding: 30px 0;
    }
  }
</style>
