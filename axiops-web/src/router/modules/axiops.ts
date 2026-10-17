import { AppRouteRecord } from '@/types/router'

export const axiopsRoutes: AppRouteRecord[] = [
  {
    path: '/clusters',
    name: 'AxiOpsClusters',
    component: '/index/index',
    meta: {
      title: '集群管理',
      icon: 'ri:cloud-line',
      menuCode: 'kubernetes',
      access: { resource: 'cluster', action: 'view' }
    },
    children: [
      {
        path: 'management',
        name: 'AxiOpsClusterManagement',
        component: '/clusters/management',
        meta: {
          title: '集群管理',
          icon: 'ri:cloud-line',
          keepAlive: true,
          menuCode: 'kubernetes',
          access: { resource: 'cluster', action: 'view' }
        }
      }
    ]
  },
  {
    path: '/cicd',
    name: 'AxiOpsCICD',
    component: '/index/index',
    meta: {
      title: 'CI/CD Delivery',
      icon: 'ri:git-branch-line',
      menuCode: 'cicd',
      access: { resource: 'project', action: 'view' }
    },
    children: [
      {
        path: 'dashboard',
        name: 'AxiOpsCICDDashboard',
        component: '/cicd/dashboard',
        meta: {
          title: 'Delivery Dashboard',
          icon: 'ri:dashboard-3-line',
          keepAlive: true,
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'projects',
        name: 'AxiOpsCICDProjects',
        component: '/cicd/projects',
        meta: {
          title: '项目交付',
          icon: 'ri:apps-2-line',
          keepAlive: true,
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'templates',
        name: 'AxiOpsTemplateLibrary',
        component: '/cicd/template-library',
        meta: {
          title: '模板库',
          icon: 'ri:file-code-line',
          menuCode: 'cicd',
          access: { resource: 'build_template', action: 'view' }
        }
      },
      {
        path: 'projects/:projectId/workflows/:workflowId',
        name: 'AxiOpsCICDWorkflowJob',
        component: '/cicd/workflow-job',
        meta: {
          title: 'Workflow Job',
          isHide: true,
          activePath: '/cicd/projects',
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'projects/:projectId/workflows/:workflowId/runs/:runId',
        name: 'AxiOpsCICDWorkflowRun',
        component: '/cicd/workflow-run',
        meta: {
          title: 'Workflow Run',
          isHide: true,
          activePath: '/cicd/projects',
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'projects/:projectId/wizard',
        name: 'AxiOpsCICDProjectWizard',
        component: '/cicd/project-workspace',
        meta: {
          title: 'Project Initialization Wizard',
          isHide: true,
          activePath: '/cicd/projects',
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'projects/:projectId',
        name: 'AxiOpsCICDProjectWorkspace',
        component: '/cicd/project-workspace',
        meta: {
          title: 'Project Workspace',
          isHide: true,
          activePath: '/cicd/projects',
          menuCode: 'cicd',
          access: { resource: 'project', action: 'view' }
        }
      },
      {
        path: 'environments/:id',
        name: 'AxiOpsCICDEnvironmentDetail',
        component: '/cicd/environment-detail',
        meta: {
          title: 'Environment Detail',
          isHide: true,
          activePath: '/cicd/projects',
          menuCode: 'cicd',
          access: { resource: 'environment', action: 'view' }
        }
      }
    ]
  }
]
