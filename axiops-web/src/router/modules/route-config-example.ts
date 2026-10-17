// CICD route reference. The active routes are registered in axiops.ts.
export const routeConfigExample = [
  {
    path: '/cicd/projects',
    name: 'CICDProjects',
    component: '/cicd/projects/index',
    meta: { title: '持续交付', icon: 'ri:git-merge-line', roles: ['R_SUPER', 'R_ADMIN'] }
  },
  {
    path: '/cicd/projects/:projectId/wizard',
    name: 'ProjectInitializationWizard',
    component: '/cicd/project-wizard/index',
    meta: {
      title: '项目初始化向导',
      isHideTab: true,
      icon: 'ri:guide-line',
      roles: ['R_SUPER', 'R_ADMIN']
    }
  },
  {
    path: '/cicd/projects/:projectId',
    name: 'ProjectWorkspace',
    component: '/cicd/project-workspace/index',
    meta: { title: '项目工作区', icon: 'ri:folders-line', roles: ['R_SUPER', 'R_ADMIN'] }
  }
] as const
