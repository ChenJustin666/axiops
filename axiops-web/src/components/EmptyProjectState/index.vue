<template>
  <div class="empty-project-state">
    <!-- 图标 -->
    <div class="empty-icon">
      <component :is="iconComponent" :size="iconSize" :stroke-width="1.2" />
    </div>

    <!-- 文字 -->
    <div class="empty-text">
      <p class="title">{{ title }}</p>
      <p class="description">{{ description }}</p>
    </div>

    <!-- 操作按钮 -->
    <ElButton type="primary" @click="handleOpenProjectSelector">
      <FolderSearch :size="16" style="margin-right: 6px" />
      {{ buttonText }}
    </ElButton>
  </div>
</template>

<script setup lang="ts">
  import { computed } from 'vue'
  import { FolderOpen, FolderSearch } from 'lucide-vue-next'
  import { mittBus } from '@/utils/sys'

  defineOptions({ name: 'EmptyProjectState' })

  interface Props {
    /** 标题文字 */
    title?: string
    /** 描述文字 */
    description?: string
    /** 按钮文字 */
    buttonText?: string
    /** 图标组件 */
    icon?: any
    /** 图标大小 */
    iconSize?: number
  }

  const props = withDefaults(defineProps<Props>(), {
    title: '请先选择项目',
    description: '选择一个项目后即可查看和管理相关数据',
    buttonText: '选择项目',
    icon: undefined,
    iconSize: 64
  })

  const iconComponent = computed(() => props.icon || FolderOpen)

  /**
   * 打开项目选择器下拉框
   */
  const handleOpenProjectSelector = () => {
    console.log('[EmptyProjectState] 点击了选择项目按钮')
    console.log('[EmptyProjectState] 准备触发 openProjectSelector 事件')
    mittBus.emit('openProjectSelector')
    console.log('[EmptyProjectState] openProjectSelector 事件已触发')
  }
</script>

<style lang="scss" scoped>
  .empty-project-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 60px 20px;
    min-height: 300px;

    .empty-icon {
      color: #c0c4cc;
      margin-bottom: 20px;
    }

    .empty-text {
      text-align: center;
      margin-bottom: 24px;

      .title {
        margin: 0 0 8px 0;
        font-size: 15px;
        font-weight: 500;
        color: #606266;
      }

      .description {
        margin: 0;
        font-size: 13px;
        color: #909399;
      }
    }
  }
</style>
