declare module 'virtual:iconify-icons' {
  import type { IconifyIcon } from '@iconify/vue'

  export const iconNames: string[]
  export const iconLoaders: Record<string, () => Promise<{ default: Record<string, IconifyIcon> }>>
}

declare module 'virtual:iconify-icons/*' {
  import type { IconifyIcon } from '@iconify/vue'

  const icons: Record<string, IconifyIcon>
  export default icons
}
