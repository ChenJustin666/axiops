/**
 * 平台介绍页面配置
 * Platform Introduction Page Configuration
 */

export interface PlatformFeature {
  icon: string
  title: string
  description: string
}

export interface PlatformHighlight {
  value: string
  suffix?: string
  label: string
}

export interface PlatformConfig {
  id: number
  name: string
  enName: string // 新增英文名，增加设计感
  subtitle: string
  description: string
  themeColor: string // 主题色
  bgGradient: string // 背景弥散光
  icon: string
  highlights: PlatformHighlight[]
  features: PlatformFeature[]
  capabilities: string[]
}

export const platformConfigs: Record<number, PlatformConfig> = {
  // ==========================================
  // ID 1: 容器云平台 (Kube Nova)
  // ==========================================
  1: {
    id: 1,
    name: '容器云平台',
    enName: 'Container Cloud',
    subtitle: '企业级云原生基础设施底座',
    description:
      '基于 Kubernetes 构建的高性能容器管理平台。提供从集群全生命周期管理、应用编排、可观测性到多云调度的全栈能力，助您构建弹性、可靠的数字化基础设施。',
    themeColor: '#3b82f6', // 蓝色系
    bgGradient:
      'radial-gradient(circle at 10% 20%, rgba(59, 130, 246, 0.15) 0%, transparent 40%), radial-gradient(circle at 90% 80%, rgba(37, 99, 235, 0.1) 0%, transparent 40%)',
    icon: 'ri:ship-2-line',
    highlights: [
      { value: '99.99', suffix: '%', label: '服务可用性' },
      { value: '1000', suffix: '+', label: '节点规模支持' },
      { value: 'ms', suffix: '级', label: '弹性伸缩响应' },
      { value: '0', suffix: '故障', label: '自动化运维' }
    ],
    features: [
      {
        icon: 'ri:cloud-windy-line',
        title: '多云集群统管',
        description: '统一纳管公有云、私有云及边缘集群，实现跨云资源的全局视图与统一调度。'
      },
      {
        icon: 'ri:database-2-line',
        title: '企业级镜像中心',
        description: '集成 Harbor 高可用方案，支持镜像安全扫描、漏洞阻断及多地跨数据中心同步。'
      },
      {
        icon: 'ri:shield-keyhole-line',
        title: '多租户安全隔离',
        description: '基于 Namespace 的硬隔离机制，结合 RBAC 权限体系，确保业务数据的绝对安全。'
      },
      {
        icon: 'ri:dashboard-3-line',
        title: '全栈可观测性',
        description: '内置 Prometheus 监控体系，提供从基础设施到应用链路的端到端可视化监控。'
      },
      {
        icon: 'ri:cpu-line',
        title: '智能资源调度',
        description: '支持 CPU/GPU 异构资源调度，通过潮汐混部技术提升资源利用率至 60% 以上。'
      },
      {
        icon: 'ri:apps-line',
        title: '应用全生命周期',
        description: '提供可视化的工作负载管理，支持一键部署、滚动更新、灰度发布及自动回滚。'
      }
    ],
    capabilities: [
      '多集群联邦管理',
      'GPU 算力虚拟化',
      'HPA/VPA 弹性伸缩',
      '服务网格集成',
      '存储卷动态供给',
      '网络策略(NetworkPolicy)',
      '节点自动扩缩容',
      '审计日志留存'
    ]
  },

  // ==========================================
  // ID 2: 微服务平台 (Microservice)
  // ==========================================
  2: {
    id: 2,
    name: '微服务治理',
    enName: 'Microservice Governance',
    subtitle: '新一代服务网格治理中心',
    description:
      '基于 Istio 架构的非侵入式微服务治理平台。提供流量精细化控制、服务熔断降级、全链路追踪等核心能力，解决微服务架构下的复杂治理难题。',
    themeColor: '#10b981', // 绿色系
    bgGradient:
      'radial-gradient(circle at 10% 20%, rgba(16, 185, 129, 0.15) 0%, transparent 40%), radial-gradient(circle at 90% 80%, rgba(5, 150, 105, 0.1) 0%, transparent 40%)',
    icon: 'ri:links-line',
    highlights: [
      { value: '100', suffix: '%', label: '无侵入接入' },
      { value: 'L7', suffix: '层', label: '流量精细管控' },
      { value: 'Sec', suffix: '级', label: '故障定位速度' },
      { value: 'TLS', suffix: '', label: '全链路加密' }
    ],
    features: [
      {
        icon: 'ri:flow-chart',
        title: '智能流量路由',
        description: '支持基于权重、Header、Cookie 的灰度发布策略，实现金丝雀发布与蓝绿部署。'
      },
      {
        icon: 'ri:shield-flash-line',
        title: '服务熔断限流',
        description: '可视化的熔断器与限流器配置，自动隔离异常服务，防止雪崩效应，保障系统稳健。'
      },
      {
        icon: 'ri:treasure-map-line',
        title: '全链路追踪',
        description: '集成 Jaeger 链路追踪，自动串联微服务调用链，快速定位性能瓶颈与异常节点。'
      },
      {
        icon: 'ri:eye-2-line',
        title: '服务拓扑感知',
        description: '自动生成微服务依赖拓扑图，实时监控服务间的调用关系、QPS、延迟与错误率。'
      },
      {
        icon: 'ri:lock-password-line',
        title: '零信任安全',
        description: '默认开启 mTLS 双向认证，提供服务间的细粒度访问控制策略，构建零信任网络。'
      },
      {
        icon: 'ri:equalizer-line',
        title: '统一配置中心',
        description: '支持配置动态下发与热加载，解决微服务架构下配置文件分散、难以管理的问题。'
      }
    ],
    capabilities: [
      'Istio 服务网格',
      '流量镜像(Mirroring)',
      '故障注入演练',
      '离群实例驱逐',
      '服务访问黑白名单',
      '调用链分析',
      '网关(Gateway)管理',
      'WASM 插件扩展'
    ]
  },

  // ==========================================
  // ID 3: DevOps 平台
  // ==========================================
  3: {
    id: 3,
    name: 'DevOps 平台',
    enName: 'Delivery Command Center',
    subtitle: '企业级云原生交付控制平面',
    description:
      '面向平台工程打造的 DevOps 指挥中枢，统一编排 Jenkins 与 Tekton 双引擎，将代码、制品、环境、凭证、质量门禁和发布策略收束到一条可治理、可审计、可持续优化的交付链路。',
    themeColor: '#5D87FF', // 蓝色系
    bgGradient:
      'linear-gradient(135deg, rgba(93, 135, 255, 0.16) 0%, rgba(29, 132, 255, 0.1) 48%, rgba(56, 192, 252, 0.08) 100%)',
    icon: 'ri:rocket-2-line',
    highlights: [
      { value: '双', suffix: '引擎', label: 'Jenkins 与 Tekton 统一编排' },
      { value: '14', suffix: '天', label: '交付态势滚动洞察' },
      { value: '全链路', suffix: '', label: '构建发布审计闭环' },
      { value: '项目级', suffix: '', label: '权限与资源隔离' }
    ],
    features: [
      {
        icon: 'ri:radar-line',
        title: '交付驾驶舱',
        description:
          '聚合构建量、成功率、失败风险、平均时长与资源承载，把全局交付态势压缩成一屏可决策视图。'
      },
      {
        icon: 'ri:node-tree',
        title: '双引擎编排',
        description:
          '同时支持 Jenkins 生态与 Tekton 云原生执行模型，传统流水线迁移和 Kubernetes 原生构建可以并行演进。'
      },
      {
        icon: 'ri:flow-chart',
        title: '可视化流水线',
        description:
          '用 DAG 方式编排构建、扫描、部署、审批和通知，复杂发布流程可以被清晰表达、复用和审计。'
      },
      {
        icon: 'ri:shield-check-line',
        title: '质量门禁前移',
        description:
          '把静态代码、镜像漏洞、测试结果和人工审批纳入流水线关键节点，让风险在进入生产之前被阻断。'
      },
      {
        icon: 'ri:links-line',
        title: '项目级渠道治理',
        description:
          '按项目绑定真实可用渠道实例，统一管理构建引擎、代码仓库、制品库、主机和 Kubernetes 部署目标。'
      },
      {
        icon: 'ri:lock-password-line',
        title: '凭证与配置中枢',
        description:
          '区分系统级与项目级凭证，联动配置中心和运行资源，敏感信息不散落在流水线脚本里。'
      }
    ],
    capabilities: [
      'Jenkins / Tekton 双引擎',
      '可视化 DAG 流水线',
      '项目级渠道绑定',
      '构建渠道健康分析',
      '静态代码与镜像安全扫描',
      '环境与配置统一治理',
      '凭证分级授权',
      '交付趋势与风险排行'
    ]
  },

  // ==========================================
  // ID 4: 门户平台
  // ==========================================
  4: {
    id: 4,
    name: '门户平台',
    enName: 'Portal Hub',
    subtitle: '统一租户、权限与公共能力管理中心',
    description:
      '统一维护项目、成员、平台绑定、用户、角色、菜单、权限和告警中心。容器平台、DevOps 平台等模块系统通过门户项目完成租户隔离与权限边界收敛。',
    themeColor: '#2563eb',
    bgGradient:
      'radial-gradient(circle at 12% 22%, rgba(37, 99, 235, 0.14) 0%, transparent 38%), radial-gradient(circle at 88% 78%, rgba(20, 184, 166, 0.12) 0%, transparent 40%)',
    icon: 'ri:shield-user-line',
    highlights: [
      { value: '统一', suffix: '', label: '项目主数据入口' },
      { value: '多', suffix: '平台', label: '项目访问范围绑定' },
      { value: '成员级', suffix: '', label: '租户权限隔离' },
      { value: '公共', suffix: '', label: '告警与权限能力' }
    ],
    features: [
      {
        icon: 'ri:folder-settings-line',
        title: '统一项目管理',
        description: '项目增删改查、成员维护与平台绑定都在门户收口，模块系统不再维护项目主数据。'
      },
      {
        icon: 'ri:team-line',
        title: '项目成员边界',
        description: '按项目关联用户，普通用户只能访问自己加入的项目，支撑多租户隔离。'
      },
      {
        icon: 'ri:apps-2-line',
        title: '平台访问绑定',
        description: '项目可以绑定容器、微服务、DevOps 等平台，模块系统按绑定关系加载项目。'
      },
      {
        icon: 'ri:menu-line',
        title: '动态菜单治理',
        description: '统一维护用户、角色、部门、权限、菜单和平台访问，动态菜单按平台加载。'
      },
      {
        icon: 'ri:alarm-warning-line',
        title: '告警中心',
        description: '告警规则、渠道、分组和事件收敛到门户，作为公共通知能力复用。'
      },
      {
        icon: 'ri:link-m',
        title: '模块系统联动',
        description: 'DevOps 通过 portalProjectUuid 关联门户项目，Mongo 仅保存模块扩展数据。'
      }
    ],
    capabilities: [
      '统一项目主数据',
      '项目成员绑定',
      '项目平台绑定',
      '用户与角色管理',
      '动态菜单管理',
      '接口权限管理',
      '告警规则与通知',
      '模块系统数据同步'
    ]
  }
}

export function getPlatformConfig(platformId: number): PlatformConfig | null {
  return platformConfigs[platformId] || null
}

export function getDefaultPlatformConfig(): PlatformConfig {
  return platformConfigs[1]
}
