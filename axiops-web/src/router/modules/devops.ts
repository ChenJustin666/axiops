import { AppRouteRecord } from '@/types/router'

export const devopsRoutes: AppRouteRecord[] = [
  {
    path: '/devops/dashboard',
    name: 'DevOpsDashboard',
    component: '/index/index',
    meta: {
      title: 'DevOps 仪表盘',
      icon: 'ri:dashboard-line',
      roles: ['R_SUPER', 'R_ADMIN']
    },
    children: [
      {
        path: '/devops/dashboard/cockpit',
        name: 'DevOpsDashboardCockpit',
        component: '/devops/dashboard/cockpit',
        meta: { title: '驾驶舱', icon: 'ri:dashboard-line', keepAlive: false }
      },
      {
        path: '/devops/dashboard/project',
        name: 'DevOpsDashboardProject',
        component: '/devops/dashboard/project',
        meta: { title: '项目大盘', icon: 'ri:bar-chart-box-line', keepAlive: false }
      }
    ]
  },
  {
    path: '/devops/channel',
    name: 'DevOpsChannelManagement',
    component: '/index/index',
    meta: {
      title: '渠道管理',
      icon: 'ri:git-branch-line',
      roles: ['R_SUPER', 'R_ADMIN']
    },
    children: [
      {
        path: '/devops/channel/group',
        name: 'DevOpsChannelGroup',
        component: '/devops/channel/group',
        meta: { title: '渠道分组', icon: 'ri:node-tree', keepAlive: true }
      },
      {
        path: '/devops/channel/management',
        name: 'DevOpsChannelTypeManage',
        component: '/devops/channel/management',
        meta: { title: '渠道类型', icon: 'ri:settings-3-line', keepAlive: true }
      },
      {
        path: '/devops/channel/center',
        name: 'DevOpsChannelCenter',
        component: '/devops/channel/center',
        meta: { title: '渠道中心', icon: 'ri:server-line', keepAlive: true }
      },
      {
        path: '/devops/channel/host',
        name: 'DevOpsChannelHost',
        component: '/devops/channel/host',
        meta: { title: '主机资产', icon: 'ri:computer-line', keepAlive: true }
      },
      {
        path: '/devops/channel/project-binding',
        name: 'DevOpsProjectChannelBinding',
        component: '/devops/project/channel',
        meta: { title: '渠道绑定', icon: 'ri:links-line', keepAlive: true }
      }
    ]
  },
  {
    path: '/devops/pipeline',
    name: 'DevOpsPipelineConfig',
    component: '/index/index',
    meta: {
      title: '流水线配置',
      icon: 'ri:flow-chart',
      roles: ['R_SUPER', 'R_ADMIN']
    },
    children: [
      {
        path: '/devops/pipeline/jenkins-step',
        name: 'DevOpsJenkinsStep',
        component: '/devops/pipeline/jenkins-step',
        meta: { title: 'Jenkins 步骤', icon: 'ri:braces-line', keepAlive: false }
      },
      {
        path: '/devops/pipeline/tekton-step',
        name: 'DevOpsTektonStep',
        component: '/devops/pipeline/tekton-step',
        meta: { title: 'Tekton Task', icon: 'ri:task-line', keepAlive: false }
      },
      {
        path: '/devops/pipeline/environment',
        name: 'DevOpsPipelineEnvironment',
        component: '/devops/pipeline/environment',
        meta: { title: '流水线环境', icon: 'ri:cloud-line', keepAlive: true }
      },
      {
        path: '/devops/pipeline/jenkins-template',
        name: 'DevOpsJenkinsTemplateLibrary',
        component: '/devops/pipeline/template',
        meta: { title: 'Jenkins 模板库', icon: 'simple-icons:jenkins', keepAlive: true }
      },
      {
        path: '/devops/pipeline/tekton-template',
        name: 'DevOpsTektonTemplateLibrary',
        component: '/devops/pipeline/tekton-template',
        meta: { title: 'Tekton 模板库', icon: 'simple-icons:tekton', keepAlive: false }
      },
      {
        path: '/devops/pipeline/template',
        name: 'DevOpsPipelineTemplate',
        component: '/devops/pipeline/template',
        meta: { title: '流水线模板', icon: 'ri:file-list-3-line', keepAlive: true, isHide: true }
      },
      {
        path: '/devops/pipeline/category',
        name: 'DevOpsStepCategory',
        component: '/devops/pipeline/category',
        meta: { title: '步骤分类', icon: 'ri:folder-line', keepAlive: true }
      },
      {
        path: '/devops/pipeline/instance',
        name: 'DevOpsPipelineInstance',
        component: '/devops/pipeline/instance',
        meta: { title: '流水线', icon: 'ri:play-list-2-line', keepAlive: false }
      },
      {
        path: '/devops/project/system',
        name: 'DevOpsProjectSystem',
        component: '/devops/project/system',
        meta: { title: '应用管理', icon: 'ri:apps-2-line', keepAlive: true }
      }
    ]
  },
  {
    path: '/devops/settings',
    name: 'DevOpsSystemSettings',
    component: '/index/index',
    meta: {
      title: '系统设置',
      icon: 'ri:settings-3-line',
      roles: ['R_SUPER', 'R_ADMIN']
    },
    children: [
      {
        path: '/devops/settings/credential',
        name: 'DevOpsCredentialManagement',
        component: '/devops/settings/credential',
        meta: { title: '凭证管理', icon: 'ri:key-2-line', keepAlive: true }
      },
      {
        path: '/devops/settings/jenkins-credential',
        name: 'DevOpsJenkinsCredential',
        component: '/devops/settings/jenkins-credential',
        meta: { title: 'Jenkins 凭证', icon: 'ri:key-line', keepAlive: true }
      },
      {
        path: '/devops/settings/config-center',
        name: 'DevOpsConfigCenter',
        component: '/devops/settings/config-center',
        meta: { title: '配置中心', icon: 'ri:settings-4-line', keepAlive: true }
      },
      {
        path: '/devops/settings/config-type',
        name: 'DevOpsConfigType',
        component: '/devops/settings/config-type',
        meta: { title: '配置类型', icon: 'ri:list-settings-line', keepAlive: true }
      },
      {
        path: '/devops/settings/channel',
        name: 'DevOpsProjectChannelSettings',
        component: '/devops/settings/channel',
        meta: { title: '渠道设置', icon: 'ri:links-line', keepAlive: true }
      }
    ]
  }
]
