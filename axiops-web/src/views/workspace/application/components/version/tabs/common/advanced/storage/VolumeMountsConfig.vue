<template>
  <div class="volume-mounts-config">
    <div class="section-header">
      <div class="header-left">
        <FolderOpen :size="18" />
        <h4>卷挂载配置</h4>
      </div>
    </div>

    <ElAlert type="info" :closable="false" show-icon>
      <template #title>配置容器如何挂载存储卷到指定路径</template>
    </ElAlert>

    <!-- 容器选择 -->
    <div class="container-selector">
      <label>选择容器：</label>
      <ElSelect v-model="selectedContainer" placeholder="选择容器">
        <!-- 使用 containers props 而不是从 volumeMounts 中提取 -->
        <ElOption
          v-for="containerName in containerList"
          :key="containerName"
          :label="containerName"
          :value="containerName"
        >
          <div class="container-option">
            <span>{{ containerName }}</span>
            <ElTag v-if="getContainerMountsCount(containerName) > 0" size="small" type="success">
              {{ getContainerMountsCount(containerName) }} 个挂载
            </ElTag>
            <ElTag v-else size="small" type="info">无挂载</ElTag>
          </div>
        </ElOption>
      </ElSelect>
    </div>

    <ElEmpty v-if="!selectedContainer" description="请选择容器" :image-size="60" />

    <div v-else-if="currentContainerMounts.length === 0" class="empty-mounts">
      <ElEmpty description="该容器暂无挂载配置" :image-size="60">
        <ElButton v-if="editing" type="primary" size="small" @click="addMount">
          添加挂载点
        </ElButton>
      </ElEmpty>
    </div>

    <div v-else class="mounts-list">
      <div v-for="(mount, index) in currentContainerMounts" :key="index" class="mount-item">
        <div class="mount-fields">
          <div class="form-item">
            <label>卷名</label>
            <ElSelect v-model="mount.name" :disabled="!editing" placeholder="选择存储卷">
              <ElOption v-for="vol in volumes" :key="vol.name" :label="vol.name" :value="vol.name">
                <div class="volume-option">
                  <span>{{ vol.name }}</span>
                  <ElTag size="small" type="info">{{ vol.type }}</ElTag>
                </div>
              </ElOption>
            </ElSelect>
          </div>
          <div class="form-item">
            <label>挂载路径</label>
            <ElInput v-model="mount.mountPath" :readonly="!editing" placeholder="/app/data" />
          </div>
          <div class="form-item">
            <label>子路径</label>
            <ElInput v-model="mount.subPath" :readonly="!editing" placeholder="可选" />
          </div>
          <div class="form-item-switch">
            <label>只读</label>
            <ElSwitch v-model="mount.readOnly" :disabled="!editing" />
          </div>
          <ElButton v-if="editing" type="danger" text circle @click="removeMount(index)">
            <Trash2 :size="16" />
          </ElButton>
        </div>
      </div>

      <ElButton v-if="editing" type="primary" text @click="addMount" style="margin-top: 12px">
        <Plus :size="14" style="margin-right: 4px" />
        添加挂载点
      </ElButton>
    </div>
  </div>
</template>

<script setup lang="ts">
  import { ref, computed, watch, onMounted } from 'vue'
  import { FolderOpen, Plus, Trash2 } from 'lucide-vue-next'
  import type { VolumeConfig, VolumeMountConfig } from '@/api'

  // ==================== Props ====================
  interface Props {
    editing: boolean
    volumes: VolumeConfig[]
    containers: string[] // 新增：接收所有容器列表
  }

  const props = defineProps<Props>()

  // ==================== Model ====================
  const volumeMountsValue = defineModel<VolumeMountConfig[]>('volumeMounts', { default: () => [] })

  // ==================== 状态 ====================
  const selectedContainer = ref('')

  // ==================== 计算属性 ====================

  /**
   * 容器列表：优先使用 props.containers，否则从 volumeMounts 中提取
   */
  const containerList = computed(() => {
    // 如果父组件传递了容器列表，使用它
    if (props.containers && props.containers.length > 0) {
      return props.containers
    }
    // 否则从 volumeMounts 中提取（兜底方案）
    return volumeMountsValue.value.map((vm) => vm.containerName)
  })

  /**
   * 当前选中容器的挂载配置
   */
  const currentContainerMounts = computed(() => {
    if (!selectedContainer.value) return []
    const container = volumeMountsValue.value.find(
      (vm) => vm.containerName === selectedContainer.value
    )
    return container?.mounts || []
  })

  // ==================== 方法 ====================

  /**
   * 获取指定容器的挂载点数量
   */
  const getContainerMountsCount = (containerName: string): number => {
    const container = volumeMountsValue.value.find((vm) => vm.containerName === containerName)
    return container?.mounts?.length || 0
  }

  /**
   * 确保容器在 volumeMounts 中有对应条目
   */
  const ensureContainerExists = (containerName: string): VolumeMountConfig => {
    let container = volumeMountsValue.value.find((vm) => vm.containerName === containerName)
    if (!container) {
      container = {
        containerName,
        mounts: []
      }
      volumeMountsValue.value.push(container)
    }
    if (!container.mounts) {
      container.mounts = []
    }
    return container
  }

  /**
   * 添加挂载点
   */
  const addMount = () => {
    if (!selectedContainer.value) return

    const container = ensureContainerExists(selectedContainer.value)
    container.mounts.push({
      name: '',
      mountPath: '',
      subPath: '',
      readOnly: false
    })
  }

  /**
   * 删除挂载点
   */
  const removeMount = (index: number) => {
    const container = volumeMountsValue.value.find(
      (vm) => vm.containerName === selectedContainer.value
    )
    if (container?.mounts) {
      container.mounts.splice(index, 1)
    }
  }

  // ==================== 生命周期 ====================

  /**
   * 监听容器列表变化，自动选择第一个容器
   */
  watch(
    containerList,
    (newVal) => {
      if (newVal && newVal.length > 0 && !selectedContainer.value) {
        selectedContainer.value = newVal[0]
      }
      // 如果当前选中的容器不在列表中，重置选择
      if (selectedContainer.value && !newVal.includes(selectedContainer.value)) {
        selectedContainer.value = newVal.length > 0 ? newVal[0] : ''
      }
    },
    { immediate: true }
  )

  /**
   * 监听 volumeMounts 变化，确保所有容器都有对应条目
   */
  watch(
    () => props.containers,
    (newContainers) => {
      if (!newContainers || newContainers.length === 0) return

      // 确保每个容器都在 volumeMounts 中
      newContainers.forEach((containerName) => {
        const exists = volumeMountsValue.value.some((vm) => vm.containerName === containerName)
        if (!exists) {
          volumeMountsValue.value.push({
            containerName,
            mounts: []
          })
        }
      })
    },
    { immediate: true }
  )

  onMounted(() => {
    // 初始化时，如果有容器列表但没有选中容器，选择第一个
    if (containerList.value.length > 0 && !selectedContainer.value) {
      selectedContainer.value = containerList.value[0]
    }
  })
</script>

<style lang="scss" scoped>
  .volume-mounts-config {
    padding: 20px;

    .section-header {
      display: flex;
      justify-content: space-between;
      align-items: center;
      margin-bottom: 16px;

      .header-left {
        display: flex;
        align-items: center;
        gap: 10px;

        h4 {
          margin: 0;
          font-size: 16px;
          font-weight: 600;
          color: #303133;
        }
      }
    }

    :deep(.el-alert) {
      margin-bottom: 16px;
    }

    .container-selector {
      display: flex;
      align-items: center;
      gap: 12px;
      padding: 12px 16px;
      background: #f5f7fa;
      border-radius: 6px;
      margin-bottom: 16px;

      label {
        font-size: 14px;
        font-weight: 500;
        color: #606266;
        flex-shrink: 0;
      }

      :deep(.el-select) {
        flex: 1;
        max-width: 400px;
      }

      .container-option {
        display: flex;
        justify-content: space-between;
        align-items: center;
        width: 100%;
      }
    }

    .empty-mounts {
      padding: 40px 0;
    }

    .mounts-list {
      .mount-item {
        padding: 16px;
        background: #f9fafb;
        border: 1px solid #e4e7ed;
        border-radius: 6px;
        margin-bottom: 12px;
        transition: all 0.2s;

        &:hover {
          border-color: #409eff;
        }

        &:last-child {
          margin-bottom: 0;
        }

        .mount-fields {
          display: grid;
          grid-template-columns: 1fr 1.5fr 1fr auto auto;
          gap: 12px;
          align-items: end;

          .form-item {
            label {
              display: block;
              margin-bottom: 6px;
              font-size: 13px;
              font-weight: 500;
              color: #606266;
            }

            .volume-option {
              display: flex;
              justify-content: space-between;
              align-items: center;
              width: 100%;
            }
          }

          .form-item-switch {
            display: flex;
            flex-direction: column;
            gap: 6px;

            label {
              font-size: 13px;
              font-weight: 500;
              color: #606266;
            }
          }
        }
      }
    }
  }
</style>
