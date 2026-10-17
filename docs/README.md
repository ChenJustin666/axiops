# AxiOps 文档中心

> 本目录是 AxiOps 唯一权威文档入口。文档按产品模块组织，不再维护并行的“总方案”和“接续说明”。

## 快速导航

| 模块 | 当前状态 | 文档入口 |
|---|---|---|
| 产品总览与架构 | 已确认，持续演进 | [00-overview](00-overview/README.md) |
| 权限 | 后端与权限中心前端已完成，待数据库迁移和浏览器 E2E | [01-authorization](01-authorization/README.md) |
| CI/CD 与发布 | 优先补齐 Helm；配置与发布链路隔离回归通过，待运行更新、视觉与真实集群验收；主机/构建 Agent 后续 | [02-cicd](02-cicd/README.md) |
| 监控 | 方案已确认，目标服务未实现 | [03-monitoring](03-monitoring/README.md) |
| 日志 | 方案已确认，目标服务未实现 | [04-logging](04-logging/README.md) |
| Kubernetes | 旧能力可用，统一权限和体验待改造 | [05-kubernetes](05-kubernetes/README.md) |
| 资产 | 已有云同步基础，统一资源模型待收敛 | [06-assets](06-assets/README.md) |
| Agent | 方案阶段 | [07-agent](07-agent/README.md) |
| AI 运维 | 安全与产品方案阶段 | [08-ai](08-ai/README.md) |
| 前端 | 权限中心与 CI/CD 项目工作区已重构，其他模块渐进替换 | [09-frontend](09-frontend/README.md) |
| 工程规范 | 持续执行 | [10-engineering](10-engineering/README.md) |

## 阅读顺序

1. 新成员先读[产品总览](00-overview/README.md)和[总体架构](00-overview/ARCHITECTURE.md)。
2. 开发具体功能时只读对应模块首页及其详细设计。
3. 查看当前在做什么，读[全局路线与进度](00-overview/PROGRESS.md)。
4. 每完成功能，只更新对应模块的“完成进度”，并附测试结果。

## 文档规则

- 模块首页固定包含：产品设计、需求、技术设计、实施计划、完成进度、验收标准。
- DESIGN.md 保存详细数据模型、接口和流程；DECISIONS.md 只记录关键取舍。
- 计划状态统一使用：未开始 / 进行中 / 已完成 / 阻塞。
- 不允许新增第二份全局总账；跨模块信息放在 00-overview，实现细节回到所属模块。
