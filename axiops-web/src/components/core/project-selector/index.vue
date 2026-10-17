<template>
  <div class="art-project-selector">
    <ElSelect
      ref="selectRef"
      v-model="selectedProjectId"
      filterable
      placeholder="选择项目支持搜索"
      :loading="loading"
      @change="handleProjectChange"
      @visible-change="handleVisibleChange"
      class="project-select"
      popper-class="art-project-popper"
    >
      <ElOption v-for="item in projects" :key="item.id" :label="item.name" :value="item.id">
        <div class="option-item">
          <div class="option-left">
            <span class="project-icon" :class="item.isSystem === 1 ? 'system' : 'normal'">
              <ArtSvgIcon v-if="item.isSystem === 1" icon="ri:shield-user-line" :size="16" />
              <ArtSvgIcon v-else icon="ri:folder-open-line" :size="16" />
            </span>
            <span class="project-name">{{ item.name }}</span>
          </div>
          <span class="project-badge" :class="item.isSystem === 1 ? 'system' : 'normal'">
            {{ item.isSystem === 1 ? '平台' : '普通' }}
          </span>
        </div>
      </ElOption>

      <template #empty>
        <div class="empty">
          <template v-if="loading">
            <div class="loading-container">
              <ElIcon class="is-loading">
                <Loading />
              </ElIcon>
              <span class="loading-text">加载项目中...</span>
            </div>
          </template>
          <template v-else>
            <div class="no-data">暂无项目数据</div>
          </template>
        </div>
      </template>
    </ElSelect>
  </div>
</template>

<script setup lang="ts">
  import { ref, onMounted, onUnmounted, watch, computed, nextTick } from 'vue'
  import { ElSelect, ElOption, ElMessage, ElIcon } from 'element-plus'
  import { Loading } from '@element-plus/icons-vue'
  import { useProjectStore } from '@/store/modules/project'
  import { usePlatformStore } from '@/store/modules/platform'
  import { mittBus } from '@/utils/sys'

  defineOptions({ name: 'ArtProjectSelector' })

  const projectStore = useProjectStore()
  const platformStore = usePlatformStore()

  const projects = computed(() => projectStore.projects)
  const selectedProject = computed(() => projectStore.selectedProject)
  const loading = computed(() => projectStore.loading)

  const selectedProjectId = ref<number | undefined>()
  const selectRef = ref<InstanceType<typeof ElSelect> | null>(null)

  watch(selectedProject, (newVal) => {
    if (newVal) {
      selectedProjectId.value = newVal.id
    } else {
      selectedProjectId.value = undefined
    }
  })

  const openSelector = async () => {
    await projectStore.fetchProjects()
    await nextTick()

    if (selectRef.value) {
      ;(selectRef.value as any).expanded = true
    }
  }

  onMounted(async () => {
    await projectStore.fetchProjects()

    if (selectedProject.value) {
      selectedProjectId.value = selectedProject.value.id
    }

    // 注册事件监听
    mittBus.on('openProjectSelector', openSelector)
  })

  onUnmounted(() => {
    mittBus.off('openProjectSelector', openSelector)
  })

  const handleVisibleChange = async (visible: boolean) => {
    if (visible) {
      await projectStore.fetchProjects()
    }
  }

  const resolveNextPlatformId = async (projectId?: number) => {
    const platforms = await platformStore.fetchProjectPlatforms(projectId)
    if (platforms.length === 0) {
      return null
    }
    if (
      platformStore.currentPlatformId &&
      platforms.some((item) => item.id === platformStore.currentPlatformId)
    ) {
      return platformStore.currentPlatformId
    }
    const defaultPlatform = platforms.find((item) => item.isDefault === 1 && item.isEnable === 1)
    if (defaultPlatform) {
      return defaultPlatform.id
    }
    return platforms.find((item) => item.isEnable === 1)?.id || null
  }

  const reloadWithPlatform = (platformId: number | null) => {
    const url = new URL(window.location.href)
    url.searchParams.delete('platformId')
    const hashPath = url.hash.replace(/^#/, '').split('?')[0] || '/'
    const query = new URLSearchParams(url.hash.split('?')[1] || '')
    if (platformId) {
      query.set('platformId', String(platformId))
    } else {
      query.delete('platformId')
    }
    const queryString = query.toString()
    url.hash = `${hashPath}${queryString ? `?${queryString}` : ''}`
    window.history.replaceState(null, '', url.toString())
    window.location.reload()
  }

  const handleProjectChange = async (id: number | undefined) => {
    if (id) {
      const project = projects.value.find((p) => p.id === id)
      if (project) {
        const nextPlatformId = await resolveNextPlatformId(project.id)
        if (!nextPlatformId) {
          selectedProjectId.value = selectedProject.value?.id
          ElMessage.warning('当前项目未授权任何平台')
          return
        }

        projectStore.selectProject(project)
        mittBus.emit('projectChanged', project)

        ElMessage({
          message: `已切换到「${project.name}」项目`,
          type: 'success',
          duration: 2000
        })
        reloadWithPlatform(nextPlatformId)
      }
    }
  }
</script>

<style lang="scss" scoped>
  .art-project-selector {
    .project-select {
      width: 240px;

      :deep(.el-input__wrapper) {
        min-height: 36px;
        background: #fff;
        border: 1px solid var(--el-border-color);
        border-radius: 6px;
        box-shadow: none;
        transition:
          border-color 0.2s,
          box-shadow 0.2s;

        &:hover {
          border-color: var(--el-border-color-darker);
        }

        &.is-focus {
          border-color: var(--el-color-primary);
          box-shadow: 0 0 0 1px var(--el-color-primary);
        }
      }

      :deep(.el-input__inner) {
        color: var(--el-text-color-primary);
        font-size: 14px;
        font-weight: 500;

        &::placeholder {
          color: var(--el-text-color-placeholder);
        }
      }

      :deep(.el-select__selected-item) {
        font-weight: 500;
      }
    }
  }
</style>

<style lang="scss">
  .art-project-popper {
    min-width: 320px !important;
    padding: 4px !important;
    border-radius: 6px !important;
    box-shadow: var(--el-box-shadow-light) !important;
    z-index: 3000 !important;

    .el-select-dropdown {
      min-height: 220px !important;
      max-height: 320px !important;
      overflow: hidden !important;
    }

    .el-select-dropdown__wrap {
      min-height: 220px !important;
      max-height: 320px !important;
      overflow-y: auto !important;
      overscroll-behavior: contain;
      scrollbar-gutter: stable;
    }

    .el-select-dropdown__item {
      display: flex !important;
      align-items: center !important;
      height: 44px !important;
      line-height: 44px !important;
      padding: 0 !important;
      margin: 0 !important;
      overflow: hidden !important;
      background-color: transparent !important;

      &.hover,
      &.is-hovering {
        background-color: transparent !important;

        .option-item {
          background-color: #f5f7fa;
        }
      }

      &.selected,
      &.is-selected {
        background-color: transparent !important;
        font-weight: normal !important;

        .option-item {
          background-color: #ecf5ff;

          .project-name {
            color: #409eff;
            font-weight: 500;
          }
        }
      }

      .option-item {
        display: flex !important;
        align-items: center !important;
        justify-content: space-between !important;
        width: 100%;
        height: 36px !important;
        padding: 0 12px !important;
        border-radius: 4px;
        transition: background-color 0.2s;
        gap: 10px;
        box-sizing: border-box;

        .option-left {
          display: flex !important;
          align-items: center !important;
          flex: 1;
          min-width: 0;
          gap: 10px;

          .project-icon {
            display: flex !important;
            align-items: center !important;
            justify-content: center !important;
            flex-shrink: 0;
            width: 24px;
            height: 24px;
            line-height: 1;
            border-radius: 6px;

            &.system {
              color: #0f766e;
              background: linear-gradient(135deg, #dcfce7, #dbeafe);
              box-shadow: inset 0 0 0 1px rgb(15 118 110 / 14%);
            }

            &.normal {
              color: #2563eb;
              background: linear-gradient(135deg, #eff6ff, #e0f2fe);
              box-shadow: inset 0 0 0 1px rgb(37 99 235 / 12%);
            }

            svg {
              display: block;
              width: 16px;
              height: 16px;
            }
          }

          .project-name {
            font-size: 14px;
            color: #303133;
            line-height: 1;
            white-space: nowrap;
            overflow: hidden;
            text-overflow: ellipsis;
            margin: 0 !important;
            padding: 0 !important;
            vertical-align: middle;
          }
        }

        .project-badge {
          display: inline-flex !important;
          align-items: center !important;
          justify-content: center !important;
          padding: 4px 10px !important;
          border-radius: 4px;
          font-size: 12px;
          font-weight: 600;
          line-height: 1 !important;
          white-space: nowrap;
          flex-shrink: 0;
          height: auto !important;
          margin: 0 !important;
          vertical-align: middle;

          &.system {
            background-color: #eef3ff;
            color: #4f6bed;
          }

          &.normal {
            background-color: #f0f9ff;
            color: #409eff;
          }
        }
      }
    }

    .empty {
      padding: 20px;
      text-align: center;
      color: #909399;
      font-size: 14px;

      .loading-container {
        display: flex;
        flex-direction: column;
        align-items: center;
        gap: 12px;

        .el-icon {
          font-size: 24px;
          color: #409eff;
        }

        .loading-text {
          font-size: 14px;
          color: #606266;
        }
      }

      .no-data {
        color: #909399;
      }
    }
  }
</style>
