import { AppRouteRecord } from '@/types/router'

export const clusterRoutes: AppRouteRecord = {
  name: 'ClusterRoot',
  path: '/cluster-log-entry',
  component: '/index/index',
  meta: {
    title: '日志中心',
    icon: 'ri:file-search-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: '/cluster/log/dashboard',
      name: 'ClusterLogDashboard',
      component: '/cluster/log/dashboard',
      meta: {
        title: '告警大盘',
        icon: 'ri:dashboard-horizontal-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/log/search',
      name: 'ClusterLogSearch',
      component: '/cluster/log/search',
      meta: {
        title: '日志检索',
        icon: 'ri:file-search-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/log/alert',
      name: 'ClusterLogAlert',
      component: '/cluster/log/alert',
      meta: {
        title: '日志告警',
        icon: 'ri:alarm-warning-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/log/config',
      name: 'ClusterLogConfig',
      component: '/cluster/log/config',
      meta: {
        title: '后端配置',
        icon: 'ri:server-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/log/engine',
      name: 'ClusterLogEngine',
      component: '/cluster/log/engine',
      meta: {
        title: '告警中心',
        icon: 'ri:alarm-warning-line',
        keepAlive: false
      }
    }
  ]
}

export const pollingRoutes: AppRouteRecord = {
  name: 'InspectionCenter',
  path: '/cluster/polling',
  component: '/index/index',
  meta: {
    title: '巡检中心',
    icon: 'ri:health-book-line',
    roles: ['R_SUPER', 'R_ADMIN']
  },
  children: [
    {
      path: '/cluster/polling/dashboard',
      name: 'InspectionDashboard',
      component: '/cluster/polling/dashboard',
      meta: {
        title: '巡检大盘',
        icon: 'ri:dashboard-3-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/polling/rule',
      name: 'InspectionRuleLibrary',
      component: '/cluster/polling/rule',
      meta: {
        title: '巡检规则库',
        icon: 'ri:list-check-3',
        keepAlive: false
      }
    },
    {
      path: '/cluster/polling/config',
      name: 'InspectionConfigCenter',
      component: '/cluster/polling/config',
      meta: {
        title: '巡检配置中心',
        icon: 'ri:settings-3-line',
        keepAlive: false
      }
    },
    {
      path: '/cluster/polling/report',
      name: 'InspectionReportCenter',
      component: '/cluster/polling/report',
      meta: {
        title: '巡检报告中心',
        icon: 'ri:file-chart-line',
        keepAlive: false
      }
    }
  ]
}
