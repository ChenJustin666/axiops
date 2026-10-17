<template>
  <ElPopover
    v-model:visible="popoverVisible"
    placement="bottom-end"
    :width="360"
    trigger="click"
    :show-arrow="false"
    :offset="10"
    popper-class="platform-selector-popover"
    :teleported="true"
  >
    <template #reference>
      <div class="platform-trigger" :class="{ 'is-active': popoverVisible }">
        <div class="platform-trigger__badge" :style="{ '--platform-accent': currentVisual.accent }">
          <ArtSvgIcon
            v-if="currentPlatform?.platformIcon || currentVisual.icon"
            :icon="currentPlatform?.platformIcon || currentVisual.icon"
            :size="18"
          />
          <span v-else>{{ currentVisual.shortName }}</span>
        </div>
        <span class="platform-trigger__name" :title="currentPlatform?.platformName || '未选择平台'">
          {{ currentPlatformDisplayName }}
        </span>
        <ArtSvgIcon icon="ri:arrow-down-s-line" :size="18" class="platform-trigger__arrow" />
      </div>
    </template>

    <div class="platform-dropdown">
      <div v-if="currentPlatform" class="platform-dropdown__current">
        <div class="section-title">已进入</div>
        <div
          class="platform-card is-current"
          :style="{ '--platform-accent': currentVisual.accent }"
        >
          <div class="platform-card__badge">
            <ArtSvgIcon
              v-if="currentPlatform.platformIcon || currentVisual.icon"
              :icon="currentPlatform.platformIcon || currentVisual.icon"
              :size="20"
            />
            <span v-else>{{ currentVisual.shortName }}</span>
          </div>
          <div class="platform-card__content">
            <span class="platform-card__name">{{ currentPlatform.platformName }}</span>
            <span class="platform-card__desc">
              {{ currentPlatform.platformDesc || currentPlatform.platformCode }}
            </span>
          </div>
          <ElTag size="small" effect="dark" type="primary">当前</ElTag>
        </div>
      </div>

      <div v-if="switchablePlatforms.length > 0" class="platform-dropdown__switch">
        <div class="section-title">切换平台</div>
        <div class="platform-list">
          <button
            v-for="platform in switchablePlatforms"
            :key="platform.id"
            type="button"
            class="platform-card platform-card--action"
            :class="{ 'is-disabled': platform.isEnable !== 1 }"
            :style="{ '--platform-accent': getPlatformVisual(platform).accent }"
            @click="handleSelect(platform)"
          >
            <div class="platform-card__badge">
              <ArtSvgIcon
                v-if="platform.platformIcon || getPlatformVisual(platform).icon"
                :icon="platform.platformIcon || getPlatformVisual(platform).icon"
                :size="18"
              />
              <span v-else>{{ getPlatformVisual(platform).shortName }}</span>
            </div>
            <div class="platform-card__content">
              <span class="platform-card__name">{{ platform.platformName }}</span>
              <span class="platform-card__desc">
                {{ platform.platformDesc || platform.platformCode }}
              </span>
            </div>
            <ArtSvgIcon
              v-if="platform.isEnable === 1"
              icon="ri:arrow-right-up-line"
              :size="16"
              class="platform-card__action"
            />
            <ElTag v-else size="small" effect="plain" type="info">禁用</ElTag>
          </button>
        </div>
      </div>

      <div v-if="platformList.length === 0" class="platform-dropdown__empty">暂无可用平台</div>
    </div>
  </ElPopover>
</template>

<script setup lang="ts">
  import { computed, ref } from 'vue'
  import { storeToRefs } from 'pinia'
  import { usePlatformStore } from '@/store/modules/platform'
  import type { PlatformSysPlatform } from '@/api/portal/platform'

  defineOptions({ name: 'ArtPlatformSelector' })

  const platformStore = usePlatformStore()
  const { platformList, currentPlatformId } = storeToRefs(platformStore)
  const popoverVisible = ref(false)

  const currentPlatform = computed(() => {
    return platformList.value.find((item) => item.id === currentPlatformId.value) || null
  })

  const currentVisual = computed(() => platformStore.getPlatformVisual(currentPlatform.value))
  const compactPlatformNameMap: Record<string, string> = {
    container: '容器',
    microservice: '微服务',
    devops: 'DevOps',
    portal: '门户'
  }

  const currentPlatformDisplayName = computed(() => {
    const code = String(currentPlatform.value?.platformCode || '').toLowerCase()
    if (compactPlatformNameMap[code]) {
      return compactPlatformNameMap[code]
    }

    const name = String(currentPlatform.value?.platformName || '').trim()
    if (!name) {
      return '平台'
    }
    return name.length > 6 ? name.slice(0, 6) : name
  })

  const switchablePlatforms = computed(() => {
    return platformList.value.filter((item) => item.id !== currentPlatformId.value)
  })

  const getPlatformVisual = (platform: PlatformSysPlatform) =>
    platformStore.getPlatformVisual(platform)

  const platformHomePathMap: Record<string, string> = {
    portal: '/dashboard/console',
    container: '/dashboard/console',
    microservice: '/dashboard/console',
    devops: '/devops/dashboard/cockpit'
  }

  const getPlatformHomePath = (platform: PlatformSysPlatform) => {
    const code = String(platform.platformCode || '').toLowerCase()
    return platformHomePathMap[code] || '/dashboard/console'
  }

  const buildPlatformHomeUrl = (platform: PlatformSysPlatform) => {
    const url = new URL(window.location.href)
    url.searchParams.delete('platformId')
    url.hash = `${getPlatformHomePath(platform)}?platformId=${platform.id}`
    return url.toString()
  }

  const openPlatformHome = (platform: PlatformSysPlatform) => {
    const link = document.createElement('a')
    link.href = buildPlatformHomeUrl(platform)
    link.target = '_blank'
    link.rel = 'noopener noreferrer'
    document.body.appendChild(link)
    link.click()
    link.remove()
  }

  const handleSelect = (platform: PlatformSysPlatform) => {
    if (platform.id === currentPlatformId.value || platform.isEnable !== 1) {
      return
    }

    popoverVisible.value = false
    openPlatformHome(platform)
  }
</script>

<style lang="scss" scoped>
  .platform-trigger {
    display: inline-flex;
    align-items: center;
    gap: 10px;
    min-width: 108px;
    max-width: 146px;
    height: 40px;
    padding: 0 10px 0 8px;
    cursor: pointer;
    background: #fff;
    border: 1px solid var(--el-border-color);
    border-radius: 10px;
    transition:
      border-color 0.2s ease,
      box-shadow 0.2s ease,
      transform 0.2s ease;

    &:hover,
    &.is-active {
      border-color: var(--el-color-primary);
      box-shadow: 0 0 0 3px rgba(93, 135, 255, 0.12);
    }

    &__badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      flex: 0 0 28px;
      width: 28px;
      height: 28px;
      color: #fff;
      background: var(--platform-accent, #5d87ff);
      border-radius: 8px;
      box-shadow: 0 8px 18px color-mix(in srgb, var(--platform-accent, #5d87ff) 30%, transparent);
    }

    &__name {
      flex: 1;
      min-width: 0;
      overflow: hidden;
      font-size: 13px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      line-height: 1;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &__arrow {
      color: var(--el-text-color-secondary);
      transition: transform 0.2s ease;
    }

    &.is-active &__arrow {
      transform: rotate(180deg);
    }
  }

  .platform-dropdown {
    display: flex;
    flex-direction: column;
    gap: 16px;
    min-height: 420px;
    max-height: min(620px, calc(100vh - 120px));
    overflow: hidden;

    &__current {
      flex: 0 0 auto;
    }

    &__switch {
      display: flex;
      flex: 1 1 auto;
      flex-direction: column;
      min-height: 0;
    }

    &__empty {
      padding: 12px 4px 4px;
      font-size: 13px;
      color: var(--el-text-color-secondary);
      text-align: center;
    }
  }

  .section-title {
    margin-bottom: 10px;
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }

  .platform-list {
    display: flex;
    flex: 1 1 auto;
    flex-direction: column;
    gap: 10px;
    min-height: 240px;
    max-height: 100%;
    padding-top: 2px;
    overflow-y: auto;
    overscroll-behavior: contain;
    scrollbar-gutter: stable;
  }

  .platform-card {
    display: flex;
    align-items: center;
    gap: 12px;
    width: 100%;
    min-height: 72px;
    padding: 12px 14px;
    text-align: left;
    background: linear-gradient(
      135deg,
      color-mix(in srgb, var(--platform-accent, #5d87ff) 11%, #ffffff) 0%,
      #ffffff 72%
    );
    border: 1px solid color-mix(in srgb, var(--platform-accent, #5d87ff) 14%, #dcdfe6);
    border-radius: 14px;

    &.is-current {
      box-shadow: 0 16px 30px rgba(15, 23, 42, 0.08);
    }

    &--action {
      cursor: pointer;
      transition:
        transform 0.2s ease,
        border-color 0.2s ease,
        box-shadow 0.2s ease;

      &:hover:not(.is-disabled) {
        border-color: color-mix(in srgb, var(--platform-accent, #5d87ff) 42%, #dcdfe6);
        box-shadow: 0 12px 24px rgba(15, 23, 42, 0.08);
      }
    }

    &.is-disabled {
      opacity: 0.58;
      cursor: not-allowed;
      filter: grayscale(0.18);
    }

    &__badge {
      display: inline-flex;
      align-items: center;
      justify-content: center;
      flex: 0 0 38px;
      width: 38px;
      height: 38px;
      color: #fff;
      background: var(--platform-accent, #5d87ff);
      border-radius: 12px;
      box-shadow: 0 10px 20px color-mix(in srgb, var(--platform-accent, #5d87ff) 28%, transparent);
    }

    &__content {
      display: flex;
      flex: 1;
      flex-direction: column;
      min-width: 0;
      gap: 4px;
    }

    &__name {
      overflow: hidden;
      font-size: 15px;
      font-weight: 600;
      color: var(--el-text-color-primary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &__desc {
      overflow: hidden;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    &__action {
      color: var(--el-text-color-secondary);
    }
  }
</style>

<style lang="scss">
  .platform-selector-popover {
    z-index: 3000 !important;
  }
</style>
