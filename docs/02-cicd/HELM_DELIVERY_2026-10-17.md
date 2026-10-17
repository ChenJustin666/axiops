# Helm 交付状态（2026-10-17）

当前主线为 Helm → 主机应用部署 → Agent 主机构建。本轮仅推进 Helm，不扩大监控、日志平台接入或审批系统。

## 本次补充：共享模板与通用工作负载

- 模板库保存不可变版本，历史只读查看、采用旧版本另存新版本；使用方只显示授权范围内的服务与环境。已有服务不会跟随模板保存自动更新。
- 服务记录真实模板来源，显示固定版本及可用更新；用目标环境现有 values 比较、校验、预览后原子升级一个环境。自定义 Chart 替换需确认；不兼容配置、过期 Token、归档模板或未授权凭据导致整笔拒绝。
- 新内置 `helm-workload-v1` 覆盖 Deployment、StatefulSet、DaemonSet、Job、CronJob；可选 Service、Ingress（含 ALB）、ConfigMap、Secret、PVC、HPA、PDB、探针、监控注解、调度、安全及 Namespace 内 RBAC/NetworkPolicy。静态检查增加原生字段/类型、域名/端口、资源数量、卷挂载及配置组合校验。默认探针关闭，默认不创建 Ingress/PVC/HPA 等额外资源。
- 密钥引用支持同一安全凭据的 password、SK、Token、证书等多个字段。正文不进入环境 values 或公共预览；字段映射与身份按环境冻结，普通同步保持目标凭据。
- 普通 Job 与常驻服务区分失败策略：包含普通 Job 的发布关闭自动原子回滚/失败清理，保留诊断资源；回滚预览和执行拒绝重放这类历史。实际升级分支临时新增 Job 时在提交前阻止不安全原子发布。CronJob 发布成功不等于后续定时任务执行成功。
- 新增迁移 **052_delivery_template_versions.sql**；上述版本功能需要配套迁移和进程更新。原环境 values 存储无需新表，但完整本版不能在未执行 052 的库上启动。
- 操作教程与资源支持边界：[共享模板与通用 Chart](HELM_SHARED_TEMPLATES_2026-10-17.md)。下方原有验证数字属于前一批 Helm 基线，不能替代本批结果。

## 产品结构

沿用 YAML 的项目工作台结构，不另造发布系统。

| 位置 | Helm 内容 | 保存影响 |
| --- | --- | --- |
| 服务与配置 / Chart 文件 | Chart.yaml、values.yaml、templates、已打包的依赖 | 创建服务 Chart 新 Revision，不自动更新环境 |
| 服务 / 构建配置 | 代码、分支、编译命令、Dockerfile、镜像仓库、缓存 | 继续复用现有构建能力 |
| 服务与配置 / 目标环境 / 环境 values | 环境独立 values、选用的 Chart Revision、镜像 values 路径 | 预览后保存，只修改指定环境，不部署 |
| 环境 | 集群、Namespace、运行资源及状态 | 发布目标由服务端解析、冻结并在执行前复核 |
| 工作流 | 构建并部署，或选择已有成功构建镜像发布 | 冻结配置与镜像摘要，由 Helm install/upgrade 执行 |
| 运行记录 / 回滚 | 操作人、耗时、阶段日志、镜像、Release revision | 回滚环境中的 Release，不回滚代码仓库 |

教程：[HELM_GUIDE_2026-10-17.md](HELM_GUIDE_2026-10-17.md)。

## 已实现的源码链路

- 服务工作台新增环境 values / Chart 文件双页签。支持文件增删改、Chart 新版本、环境原生 YAML 类型、渲染预览、保存以及未保存离开确认。Chart 模板文件不再用普通 YAML 语法检查器误报 Go 模板；保存时执行后端 Helm 校验。
- Helm 创建服务仅支持模板库 Chart 和代码仓库 Chart 导入，初始化与工作台一致，后端拒绝新建 manual Helm 服务。仓库路径与 Chart 目录分别填写，使用 Helm 导入接口固定 Commit；模板和导入完成后均可编辑 Chart。已有自定义 Helm 服务继续维护，用户已有模板库条目不被默认模板自动覆盖。
- 环境 Helm 设置存于既有 environment_resource 的内部类型 AxiOpsHelmValues，不需要数据库迁移，不通过通用运行资源 API 暴露或删除。环境复制不会生成缺少 values 的 Helm 服务绑定，需明确预览保存后才能发布。
- 构建配置沿用现有入口。成功构建或已有镜像以 SHA256 digest 注入默认 image.repository/image.tag，兼容普通 repository:tag Chart；也支持自定义完整镜像路径。校验实际渲染结果确实使用选定镜像，拒绝其他服务或项目的构建产物。
- 生产使用统一已有镜像选择/发布预览，默认不重新构建；仍可显式选择构建并部署。生产使用目标环境配置，不随镜像复制测试 values；客户端不能通过通用执行接口绕过生产选择和配置冻结校验。
- 配置同步按勾选的顶层 values 项执行，数组整项替换。保留目标其他配置、镜像映射、凭据引用；Chart 版本单独勾选。源/目标配置版本校验，生产额外确认，预览后仍有变化则阻止旧请求保存。
- 官方 Helm install/upgrade 接入实际 post-render：提交前保存实际资源清单快照，注入私有镜像拉取引用，先后校验资源范围。公共镜像也按既有环境规则准备 Namespace。同名 Release 必须归属当前项目、环境和服务，不能静默接管外部 Release。
- 安装/升级采用已有 Atomic + Wait、执行记录、租约恢复；提交前保存阶段日志，日志不可用则不发起新操作。恢复核对原记录，不盲目重复提交。成功需有效 Release revision；保留原有取消/超时语义，不承诺任意中断都能无损恢复。
- Chart/values/构建来源按运行冻结。公共渲染预览替换凭据后重新渲染，防止经 b64enc 等转换的凭据泄露；Secret data/stringData 不展示。Helm 自身的 Release 数据仍由集群权限管理。
- Helm 历史快照记录镜像与实际资源身份，回滚产生新 revision。目标漂移、冲突、历史缺失等阻止操作。保留历史上限为 20；目标历史中所有工作负载镜像必须具有 digest，含可变 tag 的附加容器不能承诺精确回滚。

## 支持边界

- 面向应用型、目标 Namespace 内的 Chart，支持受允许的工作负载及 Service、Ingress、ConfigMap、Secret、PVC、HPA、Pod 等资源；不接管 Namespace、CRD、ClusterRole 等集群范围资源。
- 依赖需预先打包进 Chart，不在部署时任意下载远程依赖。
- 添加 values 键不会自动生成资源。Ingress、PVC、Secret、探针、调度等能力须由所用 Chart 的 templates 实现。
- Helm hook 也检查资源范围，但官方 Helm post-render 不包含 hooks；私有 hook 镜像需在其 Chart 中配置正确的 imagePullSecrets。
- 日志是准备、目标、渲染/快照、等待就绪、成功或失败阶段，不是逐 Pod 实时事件流。Pod 日志/事件继续在环境运行资源入口查看。
- 配置预览可能使用校验占位镜像；它不是可以运行的业务镜像。实际发布应构建或选择已有成功镜像。

## 验证与生效状态

本轮共享模板补充：相关 Go 包与独立数据库测试通过；通用 Chart 五类控制器和配套资源通过严格原生类型解析，Job 不重放策略通过专项测试。前端相关 146 项通过，修改文件 ESLint 通过，Vite 构建通过（2分58秒）；全仓类型诊断仍 1295 条，本轮修改文件 0 条。证据 `local/dev/helm-shared-{final-go,mysql-final,validation-final,job-policy,ui-last,eslint-final,types-final,vite-build}.log`。

扩大整仓前端回归为 304 项/295 通过/9 失败，位于本轮未改的构建配置、项目标题和项目列表测试，原因包括 VM require/localStorage 缺失及旧布局断言；不宣称整仓通过。完整结果 `local/dev/helm-shared-frontend-all.log`。

本轮已编译 CI/CD API、Worker 和带 052 的迁移工具，未启动或执行业务库迁移。浏览器仍未连接，实际页面与真实集群演练待验收。下方为前一批基线证据。

Go 相关包回归、独立临时 MySQL schema 测试、fake Kubernetes 与 provider 测试通过，覆盖配置隔离、原子保存、队列配置冻结、生产已有镜像及显式构建、目标变化、私有拉取、实际 post-render 快照和环境复制。证据：

- local/dev/helm-config-final-go.log
- local/dev/helm-config-final-fixes-go.log
- local/dev/helm-config-workflow-focused.log
- local/dev/helm-config-starter-test.log
- local/dev/helm-config-ownership-final.log

前端相关 68 项测试通过，修改文件 ESLint 通过；Vite 正式打包通过（3分15秒）。全仓类型检查有 1295 条诊断，本次修改文件无诊断，不宣称全仓类型检查通过。证据 local/dev/helm-config-{web-final,lint-final,build-final,types-final}.log。

CI/CD API 与 Workflow Worker 已编译为 bin/axiops-cicd-api-helm.exe、bin/axiops-workflow-worker-helm.exe，仅生成文件，未启动。

没有触发真实业务构建、集群部署、回滚或外部通知，没有数据库迁移。浏览器工具当前无可用浏览器，桌面/窄屏视觉和真实基础设施端到端验收仍待完成。不能把组件测试、SDK/fake 测试当作这两项验收。

此前自动审批拒绝替换本地后端进程，返回 `blocked by policy`；未重试或绕过。运行中 API/Worker 仍为旧版本，这批源码需要配套更新 CI/CD API 和 Workflow Worker 后才能在页面完整生效。
