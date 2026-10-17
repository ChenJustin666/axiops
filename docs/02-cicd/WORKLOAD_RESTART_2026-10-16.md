# 工作负载滚动重启

2026-10-16：源码、隔离测试和浏览器受控验收完成。运行中的本地 API/Worker 尚未更新，本页描述新版本行为。监控/日志旧模块的资源范围接入仍是后续工作，不因本批重启完成而视为全平台权限完成。

## 如何使用

1. 在权限中心给操作账号授予目标范围的「部署 → 滚动重启」（`deployment:restart`），并允许查看对应环境、服务和工作流。可以限定项目、服务、环境与环境编码；测试权限不会自动覆盖生产。仅有查看、编辑或发布权限不能重启。
2. 进入「项目 → 环境 → 运行资源 → 工作负载」。平台已有有效 YAML 部署快照且账号有权限的 Deployment、StatefulSet、DaemonSet 行显示「重启」。独立环境资源页使用相同入口。
3. 点击重启，核对环境、集群、Namespace、资源名和当前镜像。生产环境另勾选确认，再提交。
4. 提交后打开执行记录，展示实际账号、阶段耗时、日志与状态。工作负载实际就绪后才显示成功；超过一分钟以分、秒显示。记录也保存在对应工作流历史中，不需要重新构建。
5. 失败时查看执行日志以及该资源的「YAML / 事件」、Pod 状态和容器日志。网络响应不明时点击「确认原请求结果」，使用同一个请求标识查询/重试，不重复创建重启。

这是重启当前运行配置，不是发布尚未部署的编辑器草稿。容器读取的外部 ConfigMap/Secret 内容可能已变化，应用启动后读取何种内容仍由挂载和应用行为决定。

## 目标与安全边界

- 归属取环境内每个服务最新部署快照，校验摘要、服务/工作流有效且工作流仍绑定该环境。准备中或失败的新快照不会回退到旧快照授予操作权。多服务归属冲突时拒绝。
- 服务端解析集群与 Namespace，预览固定 UID、generation、配置指纹、快照和工作流版本；客户端不能指定其他集群。提交与执行前检查当前权限和目标；页面切环境会丢弃旧预览。
- 与发布、回滚按服务互斥。重复请求固定操作者、环境及全部参数；同一个 key 换目标或换账号拒绝。
- 仅修改 Pod 模板的 `axiops.io/restart-id` 注解，使用 UID/resourceVersion 条件防止覆盖并发修改，保留其他注解、镜像和配置。不是删除 Pod，也不自动扩容。
- 使用现有持久化工作流任务与租约恢复。集群接受请求但 HTTP 响应丢失时，恢复先读取同一个标记，避免再次触发。已接受的操作可继续观察；撤销权限后不允许发起新修改。
- 目标重建、环境改绑、其他配置/副本/更新变化会停止或标记失败，不跟随新目标。HPA 在此期间调整副本也可能触发保守的配置变化检查，需要查看实际状态。
- 单副本是否中断取决于更新策略、资源容量及就绪探针，不能承诺零中断。重启采用独立权限与生产确认，本批不扩建审批系统。

## 支持范围与结果

| 情况 | 行为 |
| --- | --- |
| 平台 YAML 部署的 Deployment / StatefulSet / DaemonSet | 按控制器 RollingUpdate 策略执行，并检查 observedGeneration、更新数量与可用数量；StatefulSet 还需 currentRevision=updateRevision |
| Helm、平台外创建或无有效快照的资源 | 本批不提供重启入口 |
| 独立 Pod、Job、CronJob | 不提供通用重启入口 |
| 零副本、暂停的 Deployment、Recreate / OnDelete、StatefulSet 非零 partition、DaemonSet 无目标节点 | 拒绝并显示具体原因 |
| 控制器报告失败 / 目标漂移 | 记录失败，保留日志与资源 |
| 等待超过 10 分钟 | 平台任务失败，说明检查实际资源；Kubernetes 自身的更新可能继续，不删除资源或自动回滚 |
| 取消操作 | 未提交则停止；已提交的滚动重启不可撤回，继续确认状态或超时，不删除工作负载 |

## 验证与运行状态

- Go 回归包含 workflow、projectenv、envworkload、shared/authz、platform/role 和两个命令包；隔离 MySQL schema 验证提交幂等、双向冲突、操作者、环境改绑和权限撤销。Kubernetes 写操作全部使用本地 fake API。
- 单测覆盖三类控制器策略/就绪判断、旧失败条件、UID/版本变化、超时、响应丢失恢复、撤权不写入及取消。
- 54 项相关前端回归通过。浏览器以真实登录及受控读接口验证只读入口、准确目标、生产确认、相同请求重试、账号/耗时/日志、500px 布局、错误目标阻断；真实业务写入 0、页面异常 0。截图位于 `local/dev/browser-check/workload-restart-*.png`。
- 全仓旧前端测试仍有 9 项失败（VM 测试缺少 require/localStorage 等上下文）；全仓 vue-tsc 仍有既有诊断，不能宣称全仓检查通过。本次修改的 Vue/API 文件无类型诊断，ESLint 通过。
- 新二进制为 `bin/axiops-cicd-api-restart.exe`、`bin/axiops-workflow-worker-restart.exe`。无新迁移；启用接口前需配套更新 API、Worker 和前述平台权限后端，避免旧 Worker 接到不支持的节点类型。
- 先前自动审批审核拒绝替换本地后端进程（`blocked by policy`），本批没有重试或绕过；当前访问页面的后端尚未使用这些二进制。浏览器受控验收不代表真实集群重启已完成。

主要证据：`local/dev/restart-regression.log`、`restart-readiness-tests.log`、`restart-worker-final.log`、`restart-frontend-focused.log`、`restart-browser.log`、`restart-frontend-{lint,types,build}.log`。
