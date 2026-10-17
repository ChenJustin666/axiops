<template>
  <section class="art-card" :class="[{ 'is-empty': isEmpty }, sizeClass]" :style="cardStyle">
    <div class="art-card-header">
      <div class="header-left">
        <slot name="header-left">
          <ElButton v-if="showAdd" type="primary" :icon="Plus" @click="emit('add')">
            {{ addText }}
          </ElButton>
        </slot>
      </div>
      <div class="header-right">
        <slot name="header-right-before" />
        <button
          v-if="shouldShow('search') && showSearchBar !== undefined"
          type="button"
          class="header-tool"
          :class="{ active: showSearchBar }"
          title="搜索"
          @click="emit('update:showSearchBar', !showSearchBar)"
        >
          <ArtSvgIcon icon="ri:search-line" />
        </button>
        <button
          v-if="shouldShow('refresh') && showRefresh"
          type="button"
          class="header-tool"
          title="刷新"
          @click="emit('refresh')"
        >
          <ArtSvgIcon icon="ri:refresh-line" :class="{ 'is-loading': loading }" />
        </button>
        <ElDropdown v-if="shouldShow('size')" @command="handleCardSizeChange">
          <button type="button" class="header-tool" title="卡片密度">
            <ArtSvgIcon icon="ri:arrow-up-down-fill" />
          </button>
          <template #dropdown>
            <ElDropdownMenu>
              <ElDropdownItem
                v-for="item in cardSizeOptions"
                :key="item.value"
                :command="item.value"
                :class="{ active: cardSize === item.value }"
              >
                {{ item.label }}
              </ElDropdownItem>
            </ElDropdownMenu>
          </template>
        </ElDropdown>
        <button
          v-if="shouldShow('fullscreen')"
          type="button"
          class="header-tool"
          :class="{ active: isFullScreen }"
          title="全屏"
          @click="toggleFullScreen"
        >
          <ArtSvgIcon :icon="isFullScreen ? 'ri:fullscreen-exit-line' : 'ri:fullscreen-line'" />
        </button>
        <slot name="header-right" />
      </div>
    </div>

    <slot name="summary" />

    <div v-loading="loading" class="art-card-body">
      <slot name="body-before" />
      <slot v-if="isEmpty && !loading" name="empty">
        <ElEmpty :description="emptyText" :image-size="emptyImageSize">
          <slot name="empty-actions" />
        </ElEmpty>
      </slot>
      <div v-else class="art-card-grid">
        <template v-for="(item, index) in data" :key="getItemKey(item, index)">
          <slot :item="item" :index="index" />
        </template>
      </div>
    </div>

    <div v-if="showPagination" class="art-card-pagination" :class="paginationAlign">
      <ElPagination
        v-bind="mergedPaginationOptions"
        :total="pagination?.total"
        :page-size="pagination?.size"
        :current-page="pagination?.current"
        :disabled="loading"
        @size-change="(size: number) => emit('pagination:size-change', size)"
        @current-change="(page: number) => emit('pagination:current-change', page)"
      />
    </div>
  </section>
</template>

<script setup lang="ts">
  import { computed, onMounted, onUnmounted, ref } from 'vue'
  import { Plus } from '@element-plus/icons-vue'

  defineOptions({ name: 'ArtCard' })

  type CardSize = 'small' | 'default' | 'large'

  interface PaginationConfig {
    current: number
    size: number
    total: number
  }

  interface PaginationOptions {
    pageSizes?: number[]
    align?: 'left' | 'center' | 'right'
    layout?: string
    background?: boolean
    hideOnSinglePage?: boolean
    size?: 'small' | 'default' | 'large'
    pagerCount?: number
  }

  const props = withDefaults(
    defineProps<{
      data?: Record<string, any>[]
      loading?: boolean
      rowKey?: string | ((row: Record<string, any>, index: number) => string | number)
      emptyText?: string
      emptyImageSize?: number
      minCardWidth?: number
      gap?: number
      showAdd?: boolean
      addText?: string
      showRefresh?: boolean
      showSearchBar?: boolean
      layout?: string
      fullClass?: string
      defaultSize?: CardSize
      searchGap?: number
      pagination?: PaginationConfig
      paginationOptions?: PaginationOptions
    }>(),
    {
      data: () => [],
      loading: false,
      rowKey: 'id',
      emptyText: '暂无数据',
      emptyImageSize: 120,
      minCardWidth: 300,
      gap: 12,
      showAdd: true,
      addText: '新增',
      showRefresh: true,
      showSearchBar: undefined,
      layout: 'search,refresh,size,fullscreen',
      fullClass: 'art-page-view',
      defaultSize: 'default',
      searchGap: 12,
      pagination: undefined,
      paginationOptions: undefined
    }
  )

  const emit = defineEmits<{
    (e: 'add'): void
    (e: 'refresh'): void
    (e: 'update:showSearchBar', value: boolean): void
    (e: 'pagination:size-change', size: number): void
    (e: 'pagination:current-change', page: number): void
  }>()

  const cardSize = ref<CardSize>(props.defaultSize)
  const isFullScreen = ref(false)
  const originalOverflow = ref('')

  const isEmpty = computed(() => props.data.length === 0)

  const layoutItems = computed(() => props.layout.split(',').map((item) => item.trim()))

  const shouldShow = (name: string) => layoutItems.value.includes(name)

  const cardSizeOptions: Array<{ label: string; value: CardSize }> = [
    { label: '紧凑', value: 'small' },
    { label: '默认', value: 'default' },
    { label: '宽松', value: 'large' }
  ]

  const sizeClass = computed(() => `is-${cardSize.value}`)

  const cardStyle = computed(() => ({
    '--art-card-min-width': `${props.minCardWidth}px`,
    '--art-card-gap': `${props.gap}px`,
    marginTop: props.showSearchBar ? `${props.searchGap}px` : '0'
  }))

  const mergedPaginationOptions = computed(() => ({
    pageSizes: [15, 30, 50, 100],
    align: 'center' as const,
    background: true,
    layout: 'total, sizes, prev, pager, next, jumper',
    hideOnSinglePage: false,
    size: 'default' as const,
    pagerCount: 7,
    ...props.paginationOptions
  }))

  const paginationAlign = computed(() => mergedPaginationOptions.value.align || 'center')

  const showPagination = computed(
    () => !!props.pagination && props.pagination.total > 0 && !isEmpty.value
  )

  const getItemKey = (item: Record<string, any>, index: number) => {
    if (typeof props.rowKey === 'function') return props.rowKey(item, index)
    return item?.[props.rowKey] ?? index
  }

  const handleCardSizeChange = (size: CardSize) => {
    cardSize.value = size
  }

  const toggleFullScreen = () => {
    const el = document.querySelector(`.${props.fullClass}`)
    if (!el) return

    isFullScreen.value = !isFullScreen.value
    if (isFullScreen.value) {
      originalOverflow.value = document.body.style.overflow
      document.body.style.overflow = 'hidden'
      el.classList.add('el-full-screen')
      return
    }
    document.body.style.overflow = originalOverflow.value
    el.classList.remove('el-full-screen')
  }

  const handleEscapeKey = (event: KeyboardEvent) => {
    if (event.key === 'Escape' && isFullScreen.value) toggleFullScreen()
  }

  onMounted(() => {
    document.addEventListener('keydown', handleEscapeKey)
  })

  onUnmounted(() => {
    document.removeEventListener('keydown', handleEscapeKey)
    if (isFullScreen.value) {
      document.body.style.overflow = originalOverflow.value
      const el = document.querySelector(`.${props.fullClass}`)
      if (el) el.classList.remove('el-full-screen')
    }
  })
</script>

<style scoped lang="scss">
  .art-card {
    display: flex;
    flex: 1;
    flex-direction: column;
    min-height: 0;
    padding: var(--art-card-padding);
    background: var(--el-bg-color);
    border: 1px solid var(--el-border-color-light);
    border-radius: 8px;
  }

  .art-card.is-small {
    --art-card-padding: 12px;
    --art-card-gap: 10px;
  }

  .art-card.is-default {
    --art-card-padding: 16px;
  }

  .art-card.is-large {
    --art-card-padding: 20px;
    --art-card-gap: 16px;
  }

  .art-card-header {
    display: flex;
    flex-shrink: 0;
    align-items: center;
    justify-content: space-between;
    min-height: 36px;
    margin-bottom: 12px;
  }

  .header-left,
  .header-right {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: 8px;
  }

  .header-tool {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 36px;
    height: 36px;
    color: var(--art-text-gray-700);
    cursor: pointer;
    background: var(--el-fill-color-lighter);
    border: 1px solid transparent;
    border-radius: 8px;
    transition:
      color 0.2s ease,
      background 0.2s ease,
      border-color 0.2s ease;

    &:hover,
    &.active {
      color: #fff;
      background: var(--el-color-primary);
      border-color: var(--el-color-primary);
    }

    .is-loading {
      animation: art-card-spin 0.8s linear infinite;
    }
  }

  :deep(.el-dropdown-menu__item.active) {
    color: var(--el-color-primary);
    background: var(--el-color-primary-light-9);
  }

  .art-card-body {
    container: art-card-body / inline-size;
    flex: 1;
    min-height: 0;
    overflow: auto;
  }

  .art-card-grid {
    display: grid;
    grid-template-columns: repeat(1, minmax(0, 1fr));
    gap: var(--art-card-gap);
    min-height: 0;
    padding-bottom: 2px;
  }

  .art-card-grid > :deep(*) {
    min-width: 0;
  }

  @container art-card-body (min-width: 620px) {
    .art-card-grid {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @container art-card-body (min-width: 900px) {
    .art-card-grid {
      grid-template-columns: repeat(3, minmax(0, 1fr));
    }
  }

  @container art-card-body (min-width: 1180px) {
    .art-card-grid {
      grid-template-columns: repeat(4, minmax(0, 1fr));
    }
  }

  @container art-card-body (min-width: 1500px) {
    .art-card-grid {
      grid-template-columns: repeat(5, minmax(0, 1fr));
    }
  }

  .art-card-pagination {
    display: flex;
    flex-shrink: 0;
    padding-bottom: 14px;
    margin-top: 12px;

    &.left {
      justify-content: flex-start;
    }

    &.center {
      justify-content: center;
    }

    &.right {
      justify-content: flex-end;
    }
  }

  @keyframes art-card-spin {
    from {
      transform: rotate(0deg);
    }

    to {
      transform: rotate(360deg);
    }
  }
</style>
