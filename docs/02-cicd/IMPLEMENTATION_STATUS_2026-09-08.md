# CI/CD 实施状态（2026-09-08）

## 2026-09-08 收口补充

本轮完成项目概览、服务工作台和工作流运行详情的 UI/交互收口，同时补齐服务生命周期、输入校验和 UTF-8 响应约束。Vite 生产构建已通过（6260 modules），后端定向测试与 `go vet` 已通过。本轮没有 SQL 变更；重启和手工验收步骤记录在 [WORK_RECORD_2026-09-08_UI_BACKEND_POLISH.md](WORK_RECORD_2026-09-08_UI_BACKEND_POLISH.md)。

## 本阶段结论

AxiOps CI/CD 已完成 Zadig 风格项目工作台、三类交付服务、服务 Revision、环境矩阵、统一代码源、构建与部署工作流、运行详情、发布预览、审批、通知和权限范围控制的实现收口。当前剩余工作主要是连接真实基础设施进行端到端验收，以及把测试节点从本地 Shell runner 替换为受限执行器后再进入生产发布。

## 已交付能力

- 项目、服务、环境、工作流四层模型，支持 K8s YAML、Helm Chart 和主机应用。
- 服务三栏工作台：服务目录、服务×环境发布矩阵、Revision/YAML/Helm/环境引用工作区。
- 服务创建向导、代码库同步、模板起始定义、批量 YAML 同步和 Revision Diff。
- 一次构建、多环境晋级；运行快照记录 Commit、构建配置、镜像和 digest。
- 发布策略选择、资源预览、环境验证、生产审批、独立运行 Job 和运行详情。
- 统一平台代码源，支持 Gitea、GitLab、GitHub、Gitee 和内网 HTTP，服务创建时选择已授权仓库。
- 项目和服务范围权限、集群和环境范围权限、菜单与操作可见性控制。
- 构建/工作流日志、通知 outbox、飞书/钉钉/企业微信/邮件扩展边界。
- 服务停用采用可恢复生命周期，不破坏历史数据；后端同时阻断停用服务的 Revision 和构建入口。
- 清理未被路由或组件引用的旧 `ServiceDetailDrawer.vue`；项目详情统一使用三栏 `ServiceWorkbenchPanel.vue`。

## 发布前剩余项

- 使用测试集群、镜像仓库、真实 Git、通知渠道和主机 Agent 完成浏览器端到端验收。
- 将测试节点从宿主机 Shell runner 替换为 Kubernetes Job、专用 Agent 或沙箱执行器，并设置网络、凭据、CPU、内存和时限边界。
- 补充多集群有序发布的失败暂停、继续、回滚演练和审计核对。
- 在具备完整依赖缓存的 CI runner 上执行全仓库测试；仓库旧模块当前仍有既有 Vue 类型错误，不应与 CI/CD 定向结果混淆。
- 旧 `/api/v1/releases` 仅保留早期 `release_record` 兼容接口，新发布入口统一走工作流运行、部署快照和 Worker 执行链路。

## 数据库、重启和验收入口

- 当前迁移：`028_access_department_workflow_scope.sql`。
- 本阶段无新增 SQL，不需要补迁移。
- 需要重启 CI/CD API 和 Workflow Worker；Platform API 不需要重启；前端开发服务刷新即可。
- 详细手工流程见 [ACCEPTANCE_RUNBOOK.md](ACCEPTANCE_RUNBOOK.md) 和 [WORK_RECORD_2026-09-08_FINALIZATION.md](WORK_RECORD_2026-09-08_FINALIZATION.md)。
