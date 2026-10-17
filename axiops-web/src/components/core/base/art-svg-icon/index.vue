<!-- 图标组件 -->
<template>
  <Icon v-if="canRenderIcon" :icon="icon" v-bind="bindAttrs" class="art-svg-icon inline" />
</template>

<script setup lang="ts">
  import { Icon } from '@iconify/vue'
  import {
    ensureIconifyIcon,
    isIconifyIconLoaded,
    isLocalIconifyIcon
  } from '@/utils/ui/iconify-loader'

  defineOptions({ name: 'ArtSvgIcon', inheritAttrs: false })

  interface Props {
    /** Iconify icon name */
    icon?: string
  }

  const props = defineProps<Props>()

  const attrs = useAttrs()
  const localIconReady = ref(false)

  const bindAttrs = computed<{ class: string; style: string }>(() => ({
    class: (attrs.class as string) || '',
    style: (attrs.style as string) || ''
  }))

  const canRenderIcon = computed(
    () =>
      Boolean(props.icon) &&
      (!isLocalIconifyIcon(props.icon) || localIconReady.value || isIconifyIconLoaded(props.icon))
  )

  watch(
    () => props.icon,
    async (icon) => {
      if (!icon) {
        localIconReady.value = false
        return
      }

      if (!isLocalIconifyIcon(icon) || isIconifyIconLoaded(icon)) {
        localIconReady.value = true
        return
      }

      localIconReady.value = false
      await ensureIconifyIcon(icon)
      if (props.icon === icon) localIconReady.value = isIconifyIconLoaded(icon)
    },
    { immediate: true }
  )
</script>
