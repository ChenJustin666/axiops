import { AppRouteRecord } from '@/types/router'
import { systemRoutes } from './system'
import { axiopsRoutes } from './axiops'

/**
 * 导出所有模块化路由
 */
export const routeModules: AppRouteRecord[] = [...axiopsRoutes, systemRoutes]
