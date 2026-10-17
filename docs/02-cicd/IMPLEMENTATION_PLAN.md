# CI/CD 实施计划与验收

> Zadig 交互对齐基线：本计划的前端和服务配置重构以 [ZADIG_ALIGNMENT.md](ZADIG_ALIGNMENT.md) 为准。当前已有的三种 source_type 仅是前置能力，不能替代服务目录、组件面板、Revision 历史、环境服务矩阵和多服务同步。

> CI 设计基线：流水线模板实例化工作流和节点，Job 只在运行时按服务、阶段和环境目标展开；变量按平台 -> 项目 -> 服务 -> 环境 -> 运行参数分层，YAML/Helm 共用流水线骨架但使用强类型部署节点，详见 [CI_DESIGN.md](CI_DESIGN.md)。

## Zadig 对齐优先级

1. P0：项目工作区信息架构、服务三步向导、服务目录/编辑器/组件三栏页面。
2. P1：YAML 资源解析、单目录和同级多目录同步、Revision Diff、环境更新。
3. P2：Helm values、模板库、模板版本和应用同步。
4. P3：工作流节点配置抽屉、测试节点、运行详情和 AI 根因分析。
5. P4：权限范围、真实多集群、代码源、通知和并发 E2E 验收。

## 企业增强路线（设计已确认，代码按阶段实施）

1. 交付正确性：服务依赖图、影响分析、可配置晋升链和 artifact 门禁。
2. 发布治理：多集群目标批次、流量染色能力检测、指标门禁和自动回滚。
3. 安全合规：字段级权限、风险豁免、SBOM/签名、变更窗口和生产镜像限制。
4. 规模运营：成本中心、日志/指标/Trace 关联、2000+ 服务搜索和批量任务预估。
5. 运维流程：变更日历、冻结窗口、灾备演练和 RTO/RPO 报告。

详细模型、接口方向和验收场景见 [ENTERPRISE_OPTIMIZATION.md](ENTERPRISE_OPTIMIZATION.md)。

## 阶段 0：模型冻结和兼容迁移

交付：`delivery_type`、项目环境、服务 revision、变量组、工作流 DAG 的迁移和 API；旧项目进入 `unclassified`，不自动部署。

已完成子项：`007_cicd_project_delivery.sql`、新项目类型必填校验、旧项目单次类型确认 API、
`008_cicd_service_revision.sql`、项目类型派生服务类型、不可变服务 revision API 及范围授权校验。
新增完成子项：`009_cicd_project_environment.sql`、项目环境创建/查询、K8s 集群二次授权、生产环境
专用权限、环境 ID 列表范围、默认变量组、Secret 凭据引用和 revision 乐观锁。
新增完成子项：`010_cicd_environment_service_override.sql`、环境服务 revision/镜像/副本目标状态、
项目归属与类型兼容校验、环境和服务双重授权、覆盖 revision 乐观锁。
已完成子项：环境克隆、工作流 DAG、运行快照、生产审批、任务租约、系统回滚运行、主机 Agent 注册中心，
以及 Helm Chart Git 导入预览和不可变 revision 快照。
Helm 官方 SDK 渲染和发布 provider 已完成；阶段 0 仅剩真实基础设施验收。

验收：

- 新建项目必须选择 YAML、Helm 或主机类型。
- 已确认类型的项目不能通过普通更新或重复确认改为另一种类型。
- 旧项目未确认类型时无法创建发布运行。
- 用户只能查询被授权项目下的服务、环境和工作流。
- 每次配置更新生成新 revision，历史运行仍可读取原始快照。
- revision 编号在并发写入下唯一，服务类型和历史 revision 都不能由普通服务更新接口改写。
- YAML/Helm 环境不能绑定主机组，主机环境不能绑定集群；无集群权限不能创建 K8s 环境。
- 项目管理员不能创建生产环境；Secret 变量不能保存明文；并发变量更新不能静默覆盖。

## 阶段 1：K8s YAML 最小闭环

交付：Git YAML 解析、组件提取、环境渲染预览、构建产物注入、apply provider、资源状态回传、清单回滚。

当前进度：已完成 SQL revision 解析、Namespace/Kind/Secret 安全校验、Server-Side Apply/Diff、部署快照、
系统回滚、真实多集群 client factory、`axiops-workflow-worker`，以及 `kube_job` 命令提交、取消和终态观察。
已完成统一仓库凭据短时 Secret 注入、有界日志快照、结构化镜像 digest/制品结果回传、Git YAML 导入、
代码源管理 API、Git Webhook 幂等入口和构建日志 SSE。
服务配置入口已按 Zadig 方式支持手工 YAML、公共代码库同步和模板起始定义；代码库导入可浏览仓库目录，
按服务目录选择多个 YAML 文件，预览固定 Commit 后写入不可变 Revision。
构建并发治理、可选 MinIO 日志归档和镜像仓库 digest 校验代码已完成；待真实集群、对象存储和镜像仓库端到端验收。

验收：

- 从 Git 导入 Deployment/StatefulSet 等资源后可识别组件和镜像替换点。
- 测试环境可执行 build -> yaml_deploy，页面显示 DAG、日志、渲染 diff、image digest 和资源状态。
- 失败时停止后续节点；回滚恢复到上一成功 rendered manifest。
- 无集群或环境范围权限的用户不能预览、运行或读取日志。

## 阶段 2：Helm Chart 交付

交付：Chart/values 解析、变量分层、`helm template` diff、upgrade/rollback provider、revision 审计。

当前进度：已完成 Git Chart 文件边界校验、Chart v2 application 元数据和 values 解析、敏感默认值拒绝、
commit 一致性校验、确定性文件快照及不可变 revision 写入 API。官方 `helm.sh/helm/v3 v3.18.5` 已接入 Worker，
完成模板渲染、values 分层、Secret 运行时解密/快照脱敏、upgrade/install、执行恢复、release revision 快照和系统回滚。
发布前资源 Diff API 和项目环境配置前端已完成；待真实集群验证 server-side 行为、Helm 删除项和回滚演练。

验收：

- 环境 values 覆盖不污染其他环境或 Chart 默认 values。
- 发布前可查看渲染资源差异；生产发布必须满足审批策略。
- `upgrade --install --atomic --wait` 失败后环境保持一致；回滚可定位 Helm revision。

## 阶段 3：主机应用交付

交付：Agent 注册/心跳、主机组、制品仓库、签名任务、滚动批次、脚本白名单、探活和回滚。

当前进度：已完成 Agent 注册/心跳/主动拉取、主机组范围隔离、制品 HTTPS 与 SHA-256 校验、探活和失败回滚，
以及主机服务 Git 定义预览和不可变 revision。导入只接受同一 commit 内受路径和扩展名约束的脚本引用，
运行时按 `linux/windows` 精确筛选在线 Agent。滚动发布支持固定数量/百分比、批次间隔、事务放行、失败停止和重复终态幂等上报；待真实主机组端到端验收。

验收：

- Agent 仅主动出站连接，平台不保存 SSH 私钥，也不直接执行任意 Shell。
- 制品 checksum 校验通过后才执行；日志按主机和任务可追踪。
- 批次探活失败时停止剩余主机，可回滚到上一成功制品。

## 阶段 4：企业治理和体验

交付：工作流 DAG 编辑器、审批、通知 outbox、审计、效能指标、项目内前端，以及用户、部门、角色、菜单和资源范围授权前端。

当前进度：通知 outbox、审批/审计后端、构建效能与治理 API、项目列表、权限中心、代码源、三类服务定义导入、构建配置、环境变量与服务覆盖、资源 Diff、工作流可视化画布、工作流执行/历史/节点详情/构建日志、通知订阅与投递结果前端已完成。项目工作区已按“项目 -> 服务 -> 环境 -> 工作流”组织。待完成真实环境 E2E。

验收：

- DAG 保存时拒绝环、未连接节点和项目类型不兼容节点。
- 飞书、钉钉、企业微信、邮件、Webhook 仅通过 outbox 投递；失败可重试且不会重复通知。
- 前端在桌面和移动端均能稳定展示运行状态、日志、diff 和审批结果。
- 权限前端可配置项目、服务、环境、集群、构建和工作流范围；只读用户看不到写操作，后端仍必须复核每次请求。

## 每阶段通用质量门槛

- 代码格式化、目标单元测试、API 集成测试、迁移向前兼容检查全部通过。
- 涉及前端时，使用 Playwright 覆盖创建、运行、失败、无权限和生产审批流程。
- 每一阶段更新 `PROGRESS.md`，未实际执行的测试只能记录为阻塞，不能标记为通过。
# 近期完成记录

- 环境克隆已完成并覆盖源环境查看权限、目标环境创建权限、目标集群二次鉴权和 Secret 不复制策略。
- DAG 校验器、持久化、运行快照和 provider 调度已完成；下一步补齐工作流可视化编辑和真实基础设施 E2E。
# Code source consolidation and AI RCA follow-up

The code source consolidation is complete for the current stage: platform
connections, project reuse, HTTP internal endpoints, Gitea/GitLab/GitHub/Gitee
adapters, repository selection and encrypted credentials. No migration is
needed because the existing `code_source.project_id` column already supports
the platform value `0`.

The next RCA stage is deliberately separate from provider code:

1. Add a build/workflow RCA read model with bounded logs, failed node, commit
   and selected repository files.
2. Add local deterministic classifiers for image pull, compile, test, probe,
   RBAC and resource failures.
3. Add an optional model adapter with redaction, timeout, audit hash and a
   configurable internal endpoint.
4. Add `POST /api/v1/builds/{id}/root-cause-analysis` and enforce the caller's
   build, project, code-source and log scopes.
5. Display evidence, confidence and remediation separately; AI cannot deploy,
   approve, rotate credentials or roll back by itself.

## Follow-up backlog: test stage and version management

These items are intentionally not part of the current code-source fix:

- Add a first-class `test` workflow node with test command/image, timeout,
  report artifact and pass/fail gate. It must run after build and before deploy
  when configured, and expose bounded logs and failure details in the run view.
- Add immutable service revision and artifact version selection to workflow
  runs. A run must record the source commit, image digest or host artifact
  checksum and the exact revision used for deployment.
- Add retry policy and test-result retention without allowing a failed test to
  be bypassed by a developer-scoped read-only user. Production workflows keep
  approval and permission checks unchanged.
