import { AppRouteRecord } from '@/types/router'

// AxiOps currently exposes the unified authorization center from the system module.
// Legacy system pages stay in source until migration acceptance is complete.
export const systemRoutes: AppRouteRecord = {
  path: '/system',
  name: 'AxiOpsSystem',
  component: '/index/index',
  meta: {
    title: '系统管理',
    icon: 'ri:settings-3-line'
  },
  children: [
    {
      path: 'access',
      name: 'AxiOpsAccessCenter',
      component: '/system/access',
      meta: {
        title: '权限中心',
        icon: 'ri:shield-keyhole-line',
        keepAlive: true,
        menuCode: 'system.access',
        access: { resource: 'role_binding', action: 'view' }
      }
    },
    {
      path: 'resources',
      name: 'AxiOpsSystemResources',
      component: '/cicd/resource-center',
      meta: {
        title: '构建资源',
        icon: 'ri:cpu-line',
        keepAlive: true,
        menuCode: 'system.resources'
      }
    }
  ]
}
