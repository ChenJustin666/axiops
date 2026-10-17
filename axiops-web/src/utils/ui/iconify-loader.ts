/**
 * Iconify 离线图标加载器。
 *
 * 构建时只注册源码中实际使用的图标名，图标数据按前缀懒加载。
 */

import { addIcon, iconLoaded } from '@iconify/vue'
import { iconLoaders, iconNames } from 'virtual:iconify-icons'

type IconifyIconData = Parameters<typeof addIcon>[1]
type IconifyIconMap = Record<string, IconifyIconData>

const localIconNames = new Set(iconNames)
const iconSetCache = new Map<string, Promise<IconifyIconMap>>()

function iconPrefix(icon: string) {
  return icon.split(':')[0] || ''
}

async function loadIconSet(prefix: string) {
  if (!iconSetCache.has(prefix)) {
    const loader = iconLoaders[prefix]
    iconSetCache.set(prefix, loader ? loader().then((mod) => mod.default) : Promise.resolve({}))
  }

  return iconSetCache.get(prefix)!
}

export function isLocalIconifyIcon(icon?: string) {
  return Boolean(icon && localIconNames.has(icon))
}

export function isIconifyIconLoaded(icon?: string) {
  return Boolean(icon && iconLoaded(icon))
}

export async function ensureIconifyIcon(icon?: string) {
  if (!icon || !localIconNames.has(icon) || iconLoaded(icon)) return

  const prefix = iconPrefix(icon)
  const iconData = (await loadIconSet(prefix))[icon]
  if (iconData && !iconLoaded(icon)) addIcon(icon, iconData)
}
