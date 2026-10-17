<!--
  AxiOps 页面标题区（Page Header）

  统一替代各页面自行书写的 .page-header。此前 25 个页面的标题区各不相同：
  标题 24px / 25px 混用、下间距 18px / 24px / 0 混用、副标题 --art-text-gray-500 / 600 混用、
  有的是纯文字、有的套白卡片、有的加下边框。这里收敛成一套。

  用法：
    <ArtPageHeader title="持续交付" subtitle="项目、服务、环境与工作流">
      <template #actions>
        <ElButton type="primary" :icon="Plus">新建项目</ElButton>
      </template>
    </ArtPageHeader>

  也可用 slot 覆盖标题（例如标题前要放图标）：
    <ArtPageHeader eyebrow="ACCESS CONTROL">
      <template #title>权限中心</template>
      <template #subtitle>统一管理角色能力…</template>
    </ArtPageHeader>
-->
<template>
  <header class="ax-page-header" :class="{ 'is-bordered': bordered, 'is-compact': compact }">
    <div class="ax-page-header__main">
      <span v-if="eyebrow" class="ax-page-header__eyebrow">{{ eyebrow }}</span>

      <h1 class="ax-page-header__title">
        <slot name="title">{{ title }}</slot>
      </h1>

      <p v-if="subtitle || $slots.subtitle" class="ax-page-header__subtitle">
        <slot name="subtitle">{{ subtitle }}</slot>
      </p>
    </div>

    <div v-if="$slots.actions || $slots.extra" class="ax-page-header__actions">
      <slot name="actions" />
      <slot name="extra" />
    </div>
  </header>
</template>

<script setup lang="ts">
  defineOptions({ name: 'ArtPageHeader' })

  withDefaults(
    defineProps<{
      /** 主标题 */
      title?: string
      /** 副标题（一句话说明本页用途） */
      subtitle?: string
      /** 标题上方的小标签，全大写，例如 ACCESS CONTROL */
      eyebrow?: string
      /** 底部加一条分隔线（适合内容区无卡片的页面） */
      bordered?: boolean
      /** 紧凑模式：减小上下留白，适合列表页顶部 */
      compact?: boolean
    }>(),
    {
      title: '',
      subtitle: '',
      eyebrow: '',
      bordered: false,
      compact: false
    }
  )
</script>

<style scoped lang="scss">
  .ax-page-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--ax-space-5);
    margin-bottom: var(--ax-space-5);

    &.is-bordered {
      padding-bottom: var(--ax-space-4);
      border-bottom: 1px solid var(--ax-line);
    }

    &.is-compact {
      margin-bottom: var(--ax-space-4);
    }
  }

  .ax-page-header__main {
    min-width: 0;
  }

  .ax-page-header__eyebrow {
    display: block;
    margin-bottom: var(--ax-space-1);
    color: var(--ax-blue-deep);
    font-size: 11px;
    font-weight: 700;
    letter-spacing: var(--ax-tracking-caps);
    text-transform: uppercase;
  }

  .ax-page-header__title {
    display: flex;
    align-items: center;
    gap: var(--ax-space-3);
    margin: 0;
    color: var(--ax-ink);
    font-size: var(--ax-text-page-title);
    font-weight: 600;
    line-height: var(--ax-leading-tight);
  }

  .ax-page-header__subtitle {
    max-width: 76ch;
    margin: var(--ax-space-1) 0 0;
    color: var(--ax-slate);
    font-size: var(--ax-text-sm);
    line-height: var(--ax-leading-snug);
  }

  .ax-page-header__actions {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    gap: var(--ax-space-2);
  }

  // 窄屏：操作区换行到标题下方
  @media (max-width: 720px) {
    .ax-page-header {
      flex-direction: column;
      align-items: stretch;
      gap: var(--ax-space-3);
    }

    .ax-page-header__actions {
      justify-content: flex-start;
    }
  }
</style>
