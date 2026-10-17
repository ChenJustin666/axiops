import type { NavigationGuardNext, RouteLocationNormalized, Router } from 'vue-router'
import NProgress from 'nprogress'
import { ElMessage } from 'element-plus'
import { useSettingStore } from '@/store/modules/setting'
import { useUserStore } from '@/store/modules/user'
import { useMenuStore } from '@/store/modules/menu'
import { useWorktabStore } from '@/store/modules/worktab'
import { setWorktab } from '@/utils/navigation'
import { setPageTitle } from '@/utils/router'
import { loadingService } from '@/utils/ui'
import { getCurrentAxiOpsUserApi } from '@/api/axiops/auth'
import { HttpError, isHttpError } from '@/utils/http/error'
import { IframeRouteManager, MenuProcessor, RoutePermissionValidator, RouteRegistry } from '../core'

const PUBLIC_PAGES = new Set([
  '/auth/login',
  '/auth/register',
  '/auth/forget-password',
  '/auth/reset-password',
  '/403',
  '/404',
  '/500'
])

let routeRegistry: RouteRegistry | null = null
let pendingLoading = false
const menuProcessor = new MenuProcessor()

export function getPendingLoading(): boolean {
  return pendingLoading
}

export function resetPendingLoading(): void {
  pendingLoading = false
}

export function setupBeforeEachGuard(router: Router): void {
  routeRegistry = new RouteRegistry(router)

  router.beforeEach(async (to, _from, next) => {
    const settingStore = useSettingStore()
    if (settingStore.showNprogress) NProgress.start()

    try {
      await guardRoute(to, next, router)
    } catch (error) {
      pendingLoading = false
      loadingService.hideLoading()
      console.error('[AxiOps Router] navigation failed:', error)

      if (isAuthenticationError(error)) {
        await useUserStore().logOut()
        next({ name: 'Login', query: { redirect: to.fullPath }, replace: true })
        return
      }

      ElMessage.error(error instanceof Error ? error.message : '页面加载失败')
      next({ name: 'Exception500', replace: true })
    }
  })
}

async function guardRoute(
  to: RouteLocationNormalized,
  next: NavigationGuardNext,
  router: Router
): Promise<void> {
  const userStore = useUserStore()

  if (PUBLIC_PAGES.has(to.path)) {
    if (to.path === '/auth/login' && userStore.isLogin && userStore.accessToken) {
      next({ path: '/', replace: true })
      return
    }
    next()
    return
  }

  if (!userStore.isLogin || !userStore.accessToken) {
    next({ name: 'Login', query: { redirect: to.fullPath } })
    return
  }

  if (!routeRegistry?.isRegistered()) {
    await registerAuthorizedRoutes(to, next, router)
    return
  }

  if (to.path === '/') {
    next({ path: useMenuStore().getHomePath(), replace: true })
    return
  }

  if (to.matched.length === 0) {
    next({ name: 'Exception404' })
    return
  }

  setWorktab(to)
  setPageTitle(to)
  next()
}

async function registerAuthorizedRoutes(
  to: RouteLocationNormalized,
  next: NavigationGuardNext,
  router: Router
): Promise<void> {
  pendingLoading = true
  loadingService.showLoading()

  const userStore = useUserStore()
  const currentUser = await getCurrentAxiOpsUserApi()
  userStore.setUserInfo({
    userId: currentUser.id,
    userName: currentUser.username,
    nickName: currentUser.nickname,
    avatar: currentUser.avatar,
    email: currentUser.email,
    roles: currentUser.roles || []
  })
  userStore.checkAndClearWorktabs()

  const menuList = await menuProcessor.getMenuList()
  if (!menuProcessor.validateMenuList(menuList)) {
    throw new HttpError('当前账号没有可访问的菜单，请联系管理员授权', 403)
  }

  routeRegistry?.register(menuList)
  const menuStore = useMenuStore()
  menuStore.setMenuList(menuList)
  menuStore.addRemoveRouteFns(routeRegistry?.getRemoveRouteFns() || [])
  IframeRouteManager.getInstance().save()
  useWorktabStore().validateWorktabs(router)

  const homePath = menuStore.getHomePath()
  const targetPath = to.path === '/' ? homePath : to.path
  const validation = RoutePermissionValidator.validatePath(targetPath, menuList, homePath)

  // Re-enter navigation after dynamic routes are installed so Vue Router can match them.
  next({
    path: validation.path,
    query: validation.hasPermission ? to.query : {},
    hash: validation.hasPermission ? to.hash : '',
    replace: true
  })
}

function isAuthenticationError(error: unknown): boolean {
  if (!isHttpError(error)) return false
  return [401, 100091, 100092, 100093, 100094, 100095, 100096, 100097, 100098, 100002].includes(
    error.code
  )
}

export function resetRouterState(delay: number): void {
  setTimeout(() => {
    routeRegistry?.unregister()
    IframeRouteManager.getInstance().clear()
    const menuStore = useMenuStore()
    menuStore.removeAllDynamicRoutes()
    menuStore.setMenuList([])
  }, delay)
}
