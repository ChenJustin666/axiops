# CI/CD 与发布模块

本模块文档按产品、数据模型、API、实施计划和运行手册拆分。当前轮新增：

- [HELM_DELIVERY_2026-10-17.md](HELM_DELIVERY_2026-10-17.md)：Helm 服务/Chart、环境 values、构建镜像、生产发布及回滚的当前状态与验收边界
- [HELM_GUIDE_2026-10-17.md](HELM_GUIDE_2026-10-17.md)：创建服务、配置不同环境、发布、选择性同步与回滚的操作教程
- [HELM_TEMPLATE_REUSE_RESEARCH_2026-10-17.md](HELM_TEMPLATE_REUSE_RESEARCH_2026-10-17.md)：Zadig、Helm、Argo CD、Flux 调研；共享 Chart 与独立服务/环境 Release 的结构建议
- [HELM_DELIVERY_2026-10-16.md](HELM_DELIVERY_2026-10-16.md)：Helm 阶段日志的历史记录

- [PLATFORM_AUTHORIZATION_2026-10-16.md](PLATFORM_AUTHORIZATION_2026-10-16.md)：统一权限契约、本批基础修复、监控/日志/AI/集群的权限边界与接入缺口
- [NEXT_DELIVERY_PLAN_2026-10-16.md](NEXT_DELIVERY_PLAN_2026-10-16.md)：分批进度、实施顺序和验收标准；旧监控/日志资源范围及后端运行更新仍待完成
- [WORKLOAD_RESTART_2026-10-16.md](WORKLOAD_RESTART_2026-10-16.md)：滚动重启的权限、操作步骤、恢复与并发语义、受控验收及运行限制
- [ENVIRONMENT_CONFIGURATION_REVIEW_2026-10-02.md](ENVIRONMENT_CONFIGURATION_REVIEW_2026-10-02.md)：环境配置与发布重新评估，两个入口共用工作台、表单/YAML边界和三阶段实施计划（方案，尚未实现）
- [MIGRATION_ORDER.md](MIGRATION_ORDER.md)：完整迁移顺序和手动执行命令
- [HOST_AGENT_RUNBOOK.md](HOST_AGENT_RUNBOOK.md)：受限网络主机 Agent 安装、制品和故障处理
- [API_HOST_AGENT_NOTIFICATION.md](API_HOST_AGENT_NOTIFICATION.md)：Webhook、SSE、通知和 Agent API
- [IMPLEMENTATION_STATUS_2026-08-12.md](IMPLEMENTATION_STATUS_2026-08-12.md)：完成项、外部阻断和验证记录
- [IMPLEMENTATION_STATUS_2026-09-07.md](IMPLEMENTATION_STATUS_2026-09-07.md)：CI/CD 前端重构、乱码修复和最终回归状态
- [IMPLEMENTATION_STATUS_2026-09-08.md](IMPLEMENTATION_STATUS_2026-09-08.md)：服务生命周期、停用保护和最终交付状态
- [WORK_RECORD_2026-09-07_FRONTEND_REBUILD.md](WORK_RECORD_2026-09-07_FRONTEND_REBUILD.md)：项目工作台三栏布局和前端验收记录
- [WORK_RECORD_2026-09-07_SERVICE_WIZARD_RELEASE_DRAFT.md](WORK_RECORD_2026-09-07_SERVICE_WIZARD_RELEASE_DRAFT.md)：服务向导、批量同步和发布草稿记录
- [WORK_RECORD_2026-09-08_FINALIZATION.md](WORK_RECORD_2026-09-08_FINALIZATION.md)：服务启停、发布限制、后端保护和最终验证记录
- [WORK_RECORD_2026-09-08_ZADIG_PORTAL_REVIEW.md](WORK_RECORD_2026-09-08_ZADIG_PORTAL_REVIEW.md)：Zadig Portal 源码核对与工作台可用性修复记录
- [WORK_RECORD_2026-09-08_PROJECT_CARDS.md](WORK_RECORD_2026-09-08_PROJECT_CARDS.md)：Zadig 风格项目卡片列表重构记录
- [WORK_RECORD_2026-09-08_SERVICE_ONBOARDING_REBUILD.md](WORK_RECORD_2026-09-08_SERVICE_ONBOARDING_REBUILD.md)：服务初始化向导、三栏布局与 AxiOps 品牌收口记录
- [WORK_RECORD_2026-09-08_STRICT_ZADIG_LAYOUT.md](WORK_RECORD_2026-09-08_STRICT_ZADIG_LAYOUT.md)：严格按参考图重排项目创建与服务向导页面
- [TEST_NODE_AND_RELEASE_STRATEGY.md](TEST_NODE_AND_RELEASE_STRATEGY.md)：测试节点安全边界和滚动/金丝雀/蓝绿发布策略
- [ACCEPTANCE_RUNBOOK.md](ACCEPTANCE_RUNBOOK.md)：登录账号、权限矩阵和 YAML/Helm/主机验收步骤
- [CODE_SOURCE_AND_AI_RCA.md](CODE_SOURCE_AND_AI_RCA.md)：统一代码源、仓库选择和 AI 根因分析设计
- [ZADIG_ALIGNMENT.md](ZADIG_ALIGNMENT.md)：Zadig 三类项目流程、前端 80% 对齐范围和分阶段重构计划
- [ZADIG_PORTAL_ANALYSIS_2026-09-08.md](ZADIG_PORTAL_ANALYSIS_2026-09-08.md)：对 `reference/zadig-portal` 官方前端源码的分析与重构计划
- [ZADIG_STYLE_PRODUCT_SPEC_2026-09-08.md](ZADIG_STYLE_PRODUCT_SPEC_2026-09-08.md)：按截图冻结的项目、服务、模板、变量、环境和工作流产品规格
- [STRICT_PAGE_REQUIREMENTS_2026-09-08.md](STRICT_PAGE_REQUIREMENTS_2026-09-08.md)：按参考图冻结的页面位置、步骤和验收要求
- [CI_DESIGN.md](CI_DESIGN.md)：CI 参数、变量映射、多节点构建、流水线模板、镜像仓库和部署任务规范
- [ENTERPRISE_OPTIMIZATION.md](ENTERPRISE_OPTIMIZATION.md)：依赖、环境晋升、发布策略、权限、成本合规、规模化检索、变更日历和灾备演练
- CI 运行必须遵循“一次构建、多环境晋级”：构建产生不可变 digest/checksum，环境只保存覆盖和发布目标；Commit、变更文件、构建配置、缓存和制品信息都进入运行快照。
- 前端优化按 [FRONTEND_OPTIMIZATION_PLAN.md](FRONTEND_OPTIMIZATION_PLAN.md) 分阶段实施，环境数量由项目自行选择，支持仅测试和生产的最小链路。

## 产品设计

面向开发、测试和运维提供统一的服务、构建、环境、工作流、制品、发布、审批和回滚体验。产品以 `项目 -> 服务 -> 环境 -> 工作流` 为主线，支持三种互斥的项目交付类型：`K8s YAML`、`Helm Chart`、`主机应用`。页面必须展示阶段状态、实时日志、失败原因、提交信息、制品 digest、渲染差异和发布结果。

K8s YAML/Helm 服务配置遵循 Zadig 的三种入口：手工输入 YAML、从统一代码源同步服务目录、使用模板起始定义。服务创建后以不可变 Revision 保存配置；代码库导入固定 Commit，并在目录浏览器中选择 YAML 文件。模板库的集中管理、变量渲染和模板变更同步属于后续模板中心阶段。

## 需求

- 支持手动、Webhook、定时和审批触发。
- 支持 Jenkins、Kubernetes Job/Tekton 和受限网络 Agent provider。
- 支持 Maven、Gradle、Node、Go、Python、Docker 构建与缓存。
- 支持 K8s YAML、Helm Chart 和 ECS/物理机/虚拟机主机应用三种项目类型。
- 支持飞书、钉钉、企业微信、邮件和 Webhook 通知。
- 所有服务、构建、发布和日志按项目/服务/环境/集群授权。

## 技术设计

AxiOps 负责编排、权限、状态、制品、发布、审计与通知；provider 负责实际执行。构建结果必须保存 commit、build ID、image digest、日志和失败分类。项目类型决定服务解析、环境资源绑定、变量渲染和部署 provider，不允许只用一个 `deploy_type` 字段混合处理三种语义。

产品流程见[PRODUCT.md](PRODUCT.md)，目标数据模型见[DATA_MODEL.md](DATA_MODEL.md)，接口契约见[API.md](API.md)，实施顺序和验收见[IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md)，执行器取舍见[DECISIONS.md](DECISIONS.md)。

## 实施计划

1. 冻结三类项目的领域模型、迁移策略、模板和变量边界。
2. 以 K8s YAML 跑通“项目 -> 服务解析 -> 环境 -> 构建 -> 工作流 -> 部署 -> 回滚”闭环。
3. 实现 Helm Chart 的 values 分层、预览、upgrade/rollback provider。
4. 实现主机 Agent、制品分发、脚本执行、探活和回滚 provider。
5. 完成 DAG、审批、通知、效能度量与项目内前端体验。

## 完成进度

详细记录见 [PROGRESS.md](PROGRESS.md)。当前状态：

- 已完成：构建执行基础、权限第一阶段、命令队列、K8s Job manifest/provider 契约、凭据加密边界。
- 已完成：三类项目产品模型、数据模型、接口、实施计划和验收标准重构。
- 已完成：工作流 DAG、运行快照、生产审批、任务租约、构建 executor 和 YAML 部署 executor/provider 基础。
- 已完成：YAML 发布快照、受范围授权的回滚 API、生产审批、幂等系统运行和回滚 executor。
- 已完成：`axiops-workflow-worker` 进程、真实多集群 client factory、YAML apply/rollback executor 注册和优雅停止。
- 已完成：可选 build command 子循环、`kube_job` 提交/取消/终态观察和崩溃租约回收。
- 已完成：有界构建日志快照、统一仓库凭据短时 Secret 注入、镜像 digest/制品结果协议与回传。
- 已完成：构建日志 SSE、统一代码源、YAML/Helm/主机执行器、主机 Agent、通知 outbox、运行详情和权限中心前端。
- 已完成：服务构建配置、环境变量组、Secret 凭据引用和环境服务 Revision/镜像/副本覆盖前端。
- 已完成：工作流 DAG 可视化查看与编辑、连线环校验、服务端校验和版本乐观锁保存。
- 已完成：项目通知订阅、加密渠道凭据接入和项目范围投递结果追踪。
- 已完成：YAML/Helm 集群 Server-Side 资源预览、Helm Release 删除差异、Secret 脱敏和项目内 Diff 前端。
- 进行中：真实 Git、MySQL、Kubernetes、镜像仓库、主机 Agent、通知渠道和浏览器端到端验收；测试节点生产安全替换。

本模块采用渐进式重构。旧 `build_record` 表和旧服务暂不删除，待新 worker 完成流量切换、历史数据核对和回滚演练后再评估下线。

## 验收标准

- 三类项目分别通过对应的端到端验收，详见[实施计划与验收](IMPLEMENTATION_PLAN.md)。
- 无项目/服务/环境/集群或主机组权限的用户无法查看或操作相关流水线。
# 本轮完成记录

- 环境克隆：`POST /api/v1/environments/{id}/clone`，事务复制非 Secret 变量和服务覆盖，生产环境双向禁止。
- 工作流定义：新增 `workflow` / `workflow_node` 模型、创建读取更新校验 API，以及 DAG 环检测和三类项目部署节点兼容性校验。
- 数据库：新增 `migrations/011_cicd_workflow.sql`、`012_cicd_permission_defaults.sql`、`013_cicd_workflow_run.sql` 和 `014_cicd_workflow_governance.sql`。本轮只修改代码和文档，未连接业务数据库、未执行 SQL。
- worker：新增 `015_cicd_workflow_task_lease.sql`，为工作流节点增加租约和崩溃恢复基础。
- 构建节点：复用已有构建队列和 `kube_job` provider，异步轮询构建终态后再推进工作流。
- YAML 节点：完成不可变 revision 解析、资源白名单、Namespace 隔离、Secret 拒绝和幂等 provider 请求。
- Kubernetes provider：完成 Server-Side Apply/预览接口和 client factory 边界，集群凭据不进入工作流数据模型。
- 部署快照：新增 `016_cicd_deployment_snapshot.sql`；YAML apply 前保存已校验清单及 SHA-256，成功后才标记 `applied`，为审计、差异和受控回滚提供不可变输入。
- YAML 回滚：新增 `017_cicd_deployment_rollback.sql` 和系统回滚运行；API 强制幂等键，按项目/服务/环境/集群授权，生产环境复用工作流审批，executor 只读取已应用且摘要一致的快照。
- 工作流进程：新增 `cmd/workflow-worker` 和 `configs/workflow-worker.yaml`，启动时只校验迁移、不自动执行 DDL；注册 build、YAML deploy、YAML rollback executor，支持任务租约、轮询、错误恢复和优雅停止。
- 多集群连接：按 `cluster_id` 加载 AES-GCM 密文 kubeconfig，设置 QPS/Burst/超时和短时客户端缓存；明文或非 connected 集群拒绝执行。

启动命令（数据库迁移完成后）：

```powershell
go run ./cmd/workflow-worker -f configs/workflow-worker.yaml
```

`CredentialEncryptionKey` 必须通过部署 Secret 注入，并与 `platform-api` 使用同一值。启用 `Build.Enabled` 时还必须配置
专用 `Build.ClusterID` 和 `Build.Namespace`；worker 会提交、取消并观察 Kubernetes Job，将成功、失败或超时同步到
`build_run`。worker 轮询 builder 容器日志并以最多 1 MiB 的幂等快照写入 `build_log`；镜像 digest 和制品信息
必须由 builder 写入 `/dev/termination-log` 的 JSON 协议返回。日志 API 已提供 SSE 和轮询读取；真实镜像仓库
digest 验证及发布闭环仍需在目标环境完成验收。
