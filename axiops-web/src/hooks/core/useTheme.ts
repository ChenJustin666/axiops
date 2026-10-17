/** Centralized AxiOps theme handling. */
import { useSettingStore } from '@/store/modules/setting'
import { SystemThemeEnum } from '@/enums/appEnum'
import AppConfig from '@/config'
import { SystemThemeTypes } from '@/types/store'
import { getLightColor, setElementThemeColor } from '@/utils/ui'
import { usePreferredDark } from '@vueuse/core'

export function useTheme() {
  const settingStore = useSettingStore()

  const disableTransitions = () => {
    const style = document.createElement('style')
    style.setAttribute('id', 'disable-transitions')
    style.textContent = '* { transition: none !important; }'
    document.head.appendChild(style)
  }

  const enableTransitions = () => {
    document.getElementById('disable-transitions')?.remove()
  }

  const setSystemTheme = (_theme: SystemThemeEnum, _themeMode?: SystemThemeEnum) => {
    void _theme
    void _themeMode
    disableTransitions()
    const appliedTheme = SystemThemeEnum.LIGHT
    const el = document.documentElement
    const currentTheme = AppConfig.systemThemeStyles[appliedTheme as keyof SystemThemeTypes]
    if (currentTheme) el.setAttribute('class', currentTheme.className)

    const primary = settingStore.systemThemeColor
    for (let i = 1; i <= 9; i++) {
      el.style.setProperty(`--el-color-primary-light-${i}`, getLightColor(primary, i / 10))
    }
    settingStore.setGlopTheme(appliedTheme, appliedTheme)
    requestAnimationFrame(() => requestAnimationFrame(enableTransitions))
  }

  // Kept for settings-panel API compatibility; the baseline stays light.
  const prefersDark = usePreferredDark()
  const setSystemAutoTheme = () => setSystemTheme(SystemThemeEnum.LIGHT, SystemThemeEnum.LIGHT)
  const switchThemeStyles = (_theme: SystemThemeEnum) => {
    void _theme
    setSystemTheme(SystemThemeEnum.LIGHT, SystemThemeEnum.LIGHT)
  }

  return { setSystemTheme, setSystemAutoTheme, switchThemeStyles, prefersDark }
}

/** Apply the light theme during application bootstrap. */
export function initializeTheme() {
  const settingStore = useSettingStore()
  const actualTheme = SystemThemeEnum.LIGHT
  const el = document.documentElement
  const currentTheme = AppConfig.systemThemeStyles[actualTheme as keyof SystemThemeTypes]
  if (currentTheme) el.setAttribute('class', currentTheme.className)
  setElementThemeColor(settingStore.systemThemeColor)
  settingStore.setGlopTheme(actualTheme, actualTheme)
  el.style.setProperty('--custom-radius', `${settingStore.customRadius}rem`)
}
