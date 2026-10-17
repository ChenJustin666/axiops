# CI/CD 模块进度

## 2026-09-08 严格按参考图重排项目与服务页

- 新建项目由普通弹窗改为 Zadig 式全屏窄表单，字段和按钮顺序按参考页面重排。
- 服务向导改为四步线性 Step + 蓝色步骤标题；服务来源移入创建弹窗，主体直接保留服务树、编辑器、变量栏。
- 无服务状态和三栏宽度按 Zadig 源码样式基线调整，不再添加参考图之外的介绍区。
- 模板来源增加独立模板目录和只读 YAML 预览，确认模板后再进入服务变量与定义编辑。
- Prettier、ESLint、Vue 模板编译和 Vite 生产构建（6262 modules）通过。本轮无 SQL，后端无需重启。
- 详见 [WORK_RECORD_2026-09-08_STRICT_ZADIG_LAYOUT.md](WORK_RECORD_2026-09-08_STRICT_ZADIG_LAYOUT.md)。

## 2026-09-08 服务初始化向导与 AxiOps 品牌收口

- 服务创建向导完成 Zadig 风格全屏三栏布局：服务树、YAML/Helm 定义编辑器、变量面板。
- 项目初始化阶段状态改为动态显示；引导页接入现有 AxiOps Logo 资产，避免使用占位字标。
- 全局 `ArtLogo` 已统一为 AxiOps A 字标，登录、侧边栏和 CI/CD 向导保持一致品牌识别。
- 手工/模板服务定义保存前支持基础服务变量替换，内置 K8s 起始模板包含 Deployment + Service、镜像、端口、副本和资源限制；增加小屏响应式布局，避免 1280px/1024px 页面裁切。
- 本轮目标组件 Prettier、ESLint、Vue SFC 模板编译、`go test ./internal/cicd/...` 和 Vite 生产构建（6262 modules）通过；未新增 SQL，后端无需重启。生产静态前端需重新构建发布。
- 详见 [WORK_RECORD_2026-09-08_SERVICE_ONBOARDING_REBUILD.md](WORK_RECORD_2026-09-08_SERVICE_ONBOARDING_REBUILD.md)。

## 2026-09-08 项目列表卡片化

- 项目列表改为 Zadig 风格响应式卡片网格，增加类型分类标签和项目元信息展示。
- 创建项目表单严格保持项目元数据边界，服务 YAML、代码库和模板导入继续放在初始化向导的新建服务步骤。
- 前端 Prettier、ESLint 和 Vite 生产构建通过（6262 modules）。本轮无 SQL，后端无需重启。
- 详见 [WORK_RECORD_2026-09-08_PROJECT_CARDS.md](WORK_RECORD_2026-09-08_PROJECT_CARDS.md)。

## 2026-09-08 Zadig 风格产品规格冻结

- 明确不直接复制 Zadig Portal 旧 Vue 2 代码，改为按源码交互和页面结构重写 Vue 3 AxiOps 前端。
- 明确项目创建只填写项目元数据和类型；YAML/代码库/模板导入全部发生在“新建服务”步骤。
- 明确服务三栏布局：左服务树、中 YAML/Helm 编辑器、右变量/组件/构建配置；支持一次创建多个服务。
- 明确模板变量、系统内置变量、项目/环境变量和 Secret 脱敏规则。
- 明确环境可按项目自行选择（例如只有 dev/qa），下一步生成服务×环境绑定和运行时 Job，不复制工作流定义。
- 规格与 P0 重构顺序记录在 [ZADIG_STYLE_PRODUCT_SPEC_2026-09-08.md](ZADIG_STYLE_PRODUCT_SPEC_2026-09-08.md)。本次仅文档变更，无 SQL、无需重启。

## 2026-09-08 Zadig Portal 源码核对与工作台可用性修复

- 确认 `reference/zadig-portal` 为 Zadig 官方 Vue 2 Portal 源码，完成页面、组件和接口映射分析；AxiOps 将按页面交互重建，不直接混用 Vue 2 依赖。
- 修复项目工作台刷新时的全页加载遮罩，避免已有内容被灰色覆盖；保留首次加载提示，并补充工作台宽度约束。
- 计划和验收标准记录在 [ZADIG_PORTAL_ANALYSIS_2026-09-08.md](ZADIG_PORTAL_ANALYSIS_2026-09-08.md)。
- 本次仅前端与文档变更，无 SQL；开发模式可热更新，生产静态包需重新构建。

## 2026-09-08 UI 与后端收口

- 完成项目概览交付就绪度、服务×环境矩阵、Revision 工作区和工作流运行详情的最后一轮体验收口。
- 服务停用改为可恢复生命周期，后端同步阻断停用服务的新 Revision、手动构建和 Webhook 构建。
- 补充项目、服务、集群、Revision 输入校验，并统一 JSON/SSE UTF-8 响应头。
- 验证通过：CI/CD 定向 ESLint、Prettier、Go 测试、Go vet 和 Vite 生产构建（6260 modules）。
- 本轮无 SQL 变更；Platform API 与 CI/CD API 修改后建议执行 `start.cmd` 统一重建启动，详见 [工作记录](WORK_RECORD_2026-09-08_UI_BACKEND_POLISH.md)。

## 2026-09-08 CI/CD 收尾：服务生命周期和交付门禁

- 服务支持 `active/inactive` 生命周期；停用保留 Revision、环境引用、运行记录和审计数据，不执行破坏性删除。
- 服务目录增加停用筛选和灰态展示；停用服务不能进入批量发布、定义编辑、代码同步或新的构建。
- 环境覆盖服务选择器同步过滤停用服务，避免出现无法加载的历史选项；工作流定义仍可查看历史引用。
- 后端统一校验服务状态，阻断停用服务创建 Revision、手动构建和 Webhook 构建；状态值白名单只接受 `active/inactive`。
- 发布抽屉统一使用启用服务和工作流运行权限，删除未引用的旧 `QuickReleaseDialog.vue`。
- 删除未引用的旧 `ServiceDetailDrawer.vue`，详情入口统一使用三栏 `ServiceWorkbenchPanel.vue`；旧 `/api/v1/releases` 仅作为兼容接口保留。
- 修复发布环境多选、待分类项目类型和状态标签的 CI/CD 定向 TypeScript 问题。
- 验证通过：CI/CD 定向 ESLint、定向 `vue-tsc`、`go test`、`go vet` 和 Vite 生产构建（6260 modules）。
- 本轮未新增或修改 SQL，迁移版本仍为 `028_access_department_workflow_scope.sql`；CI/CD API 与 Workflow Worker 需要重启，Platform API 不需要重启。
- 浏览器运行环境没有可用会话，真实登录点击验收仍由用户接入测试集群后执行。

## 2026-09-06 前端优化 P0：服务环境矩阵

- 根据前端优化方案建立分阶段实施计划：`FRONTEND_OPTIMIZATION_PLAN.md`。
- 项目概览增加服务×环境矩阵，环境列按项目实际配置动态生成；仅测试和生产的项目不会出现开发/预发布占位列。
- 矩阵状态根据服务 Revision、环境绑定状态和生产标识显示可发布、待导入或待验证，并可进入环境配置。
- 本轮未新增 API、SQL 或后端运行逻辑。

验证：`axiops-web\\node_modules\\.bin\\prettier.cmd --check src/views/cicd/project-workspace/index.vue` 通过；
`axiops-web\\node_modules\\.bin\\eslint.cmd src/views/cicd/project-workspace/index.vue` 通过。

## 2026-09-06 前端优化 P0：服务详情三栏

- 服务列表增加“详情”入口，新增 `ServiceDetailDrawer.vue` 三栏视图。
- 左栏显示服务元信息和 Revision 历史；中栏显示资源定义、Commit/摘要和历史 Revision 对比；右栏显示项目实际环境引用及目标验证状态。
- 编辑、导入定义、构建配置和环境配置继续复用现有权限校验和接口，详情页不会绕过生产审批。
- 构建配置按钮增加 `build_config:view` 前端入口控制，和后端权限复核保持一致。
- 本轮未新增 API、SQL 或后端运行逻辑。

验证：详情组件与工作台文件 Prettier/ESLint 通过；全量 `vue-tsc` 定向筛选无本轮文件错误，剩余为既有工作流组件类型错误。

## 2026-09-07 前端优化 P0：环境覆盖差异与结构化编辑

- 环境服务覆盖增加目标摘要、基线 Revision、环境覆盖 Revision 和镜像/副本/版本差异展示。
- 增加 K8s YAML 多文档基线副本数提取；Helm 基线定义按 JSON 快照格式展开，均保持只读。
- 增加明确提示：环境覆盖只影响当前环境，不修改服务基线，不触发重新构建；资源结构修改需创建新服务 Revision。
- 保持现有 `environment_service_override` 乐观锁、权限和真实资源预览接口，不新增 SQL 或后端运行逻辑。

验证：`EnvironmentConfigDialog.vue` Prettier/ESLint 通过；定向 `vue-tsc` 无本轮文件错误。

## 2026-09-06 项目环境按需启用（测试/生产最小链）

- 确认环境是项目级运行上下文，不要求每个项目都拥有 `dev/test/stage/prod` 全部阶段；只有测试和生产的项目使用 `test -> prod` 晋级链。
- 明确工作流环境下拉框、权限范围和晋级门禁只读取项目已配置且用户可见的环境，不创建隐式占位环境。
- 明确测试与生产使用同一个构建产物 digest/checksum，生产发布必须经过测试验证和生产审批。
- 更新 `CI_DESIGN.md`、`DECISIONS.md` 和 `ACCEPTANCE_RUNBOOK.md`，本次无新增 SQL，无运行代码变更。

更新时间：2026-09-06

## 2026-09-06 Zadig 风格服务创建与仓库目录选择

- K8s YAML/Helm 服务创建弹窗增加三种模式：手工输入、从代码库同步、使用模板新建；主机服务继续复用代码库模式。
- 手工输入和模板模式会在创建或编辑服务时保存新的不可变 `service_revision`，编辑不会覆盖历史版本；模板模式当前提供内置 YAML 起始模板，模板库同步列入后续任务。
- 代码库模式创建服务后自动打开定义导入窗口；公共代码源和仓库下拉选择沿用平台统一代码源，仓库地址会自动带入导入表单。
- 新增只读仓库目录浏览接口 `GET /api/v1/code-sources/{id}/repository-files`，按代码源权限读取单层目录，导入窗口可进入目录并把 YAML 文件加入待读取列表。
- 本轮没有新增 SQL。后端路由变更需要重新编译并重启 CI/CD API；仅前端改动的开发服务可热更新。

验证：`go test ./internal/cicd/codesource ./cmd/cicd-api` 通过；目标 Vue 文件 ESLint/Prettier 通过。Vite 生产构建仍受 Windows 应用程序控制策略的 `spawn EPERM` 阻断。

## 2026-09-06 CI 一次构建多环境晋级设计

- 明确 CI 与 CD 边界：服务 Commit 和构建配置只构建一次，生成不可变镜像 digest/checksum；开发、测试、预发布、生产只晋级和部署，不按环境重复构建。
- 固化平台 -> 项目 -> 服务 -> 环境 -> 运行参数的变量优先级、Secret 仅凭据引用和模板表达式静态校验规则。
- 固化流水线模板、DAG 并行构建、matrix 展开、并发上限、失败策略和运行时 Job 展开模型。
- 固化滚动、金丝雀、蓝绿三种发布策略，以及逻辑环境下多集群目标的批次/目标任务关系。
- 补充镜像仓库、制品晋级、环境门禁、标准上线流程和回滚规范；当前仅为产品/接口设计，未新增 SQL 或运行代码。
- 详细方案见 `docs/02-cicd/CI_DESIGN.md`。

## 2026-09-06 CI 可追溯性、缓存与配置覆盖设计

- 固化 Source Snapshot：代码源、仓库、分支、完整 Commit、提交作者、Commit 链接、变更文件和增删行统计。
- 固化 Build Config Snapshot：构建模板、Runner、构建镜像、命令、变量、资源、缓存和脚本摘要。
- 固化分阶段结构化日志和统一脱敏规则，支持按服务、任务、阶段查看，并从失败日志跳转到 Commit 文件和配置。
- 固化 Runner 本地、项目共享、OCI BuildKit 和对象存储四层缓存，缓存键包含 lockfile、工具版本和平台，不以分支名单独作为缓存键。
- 固化服务基线 Revision 与环境覆盖 Revision 分离，副本、资源、镜像 digest 和参数可在环境级修改，修改先生成 Diff，不触发重新构建。
- 固化一次构建、多环境晋级、同一逻辑环境多集群批次和独立目标 Job 的验收场景。
- 本轮仅更新产品设计文档，未新增 SQL、未修改运行代码。

## 2026-09-06 企业级 CI/CD 优化设计

- 补充服务依赖图、依赖版本约束、契约测试、影响分析和循环依赖校验。
- 补充可配置环境晋升链，明确同一 artifact digest 经过门禁晋级而不是按环境重新构建。
- 补充流量染色、Prometheus 指标门禁、自动暂停/回滚和多集群目标批次。
- 补充菜单、资源范围、动作和字段级权限，以及风险豁免、变更窗口和生产镜像限制。
- 补充成本中心、合规检查、SBOM/签名、日志指标 Trace 关联、2000+ 服务搜索分组、变更日历和灾备演练。
- 本轮仍仅更新设计文档，未新增 SQL、未修改运行代码；后续实现前先评估现有表兼容性并按迁移编号新增脚本。

## 2026-08-20 本地启动策略修复

- Windows 应用程序控制策略会拦截 PowerShell 直接启动本地 Go `.exe`；`scripts/start-dev.ps1` 现通过 `cmd.exe` 启动后端，并记录命令包装进程以便停止脚本处理。
- Vite 默认配置加载会启动 esbuild，受同一策略影响；前端启动增加 `--configLoader runner`，并以 `axiops-web` 为工作目录读取 `VITE_PORT=3006` 和代理配置。
- `vite.config.ts` 改用基于 `import.meta.url` 的配置目录解析，兼容 runner 模式，不依赖未定义的 `__dirname`。
- `start.cmd -SkipBuild -NoWeb` 已通过数据库迁移、Platform API 和 CI/CD API 健康检查；前端 runner 命令已在允许环境验证监听 `3006`。
- 健康检查由固定 3 秒改为最多 60 秒逐秒重试，覆盖 MySQL/Redis 冷启动和应用策略导致的延迟；实测完整 `start.cmd -SkipBuild` 最终输出 `AxiOps is ready`。
- 本轮未新增 SQL。若当前终端仍被应用程序控制策略拦截，请使用仓库根目录的 `start.cmd`，不要直接执行 `Start-Process bin\\*.exe`。

## 已完成

- 根据 Zadig 三类项目思路完成 AxiOps 的三类型项目产品基线：K8s YAML、Helm Chart、主机应用；形成项目、服务、环境、工作流四层模型、接口和验收计划。

- 统一服务、构建、发布数据模型和 API 骨架。
- 构建查看、执行、取消权限复用 `internal/shared/authz`，按项目/服务范围过滤。
- 新增 `migrations/006_cicd_build_execution.sql`，补齐 `build_config`、`build_run`、
  `build_log`、`build_template` 和 `build_command`，保留旧 `build_record` 兼容历史代码。
- 使用 `build_number_sequence` 分配构建编号，避免并发请求通过 `MAX()+1` 产生重复编号。
- 触发构建时记录和 submit 命令同事务写入，命令使用幂等键 `build:{id}:submit`。
- 取消构建时写入幂等 cancel 命令，状态使用 `cancelling -> cancelled` 的确认边界。
- 提供 provider 注册契约、worker 命令领取/分发/完成/失败重试编排器；死信命令与构建失败状态同事务更新。
- 完成 `kube_job` provider 的 client-go Job 创建/删除/状态观察适配器和隔离 manifest builder：包含源码 initContainer、共享工作区、资源限制、超时、禁用 ServiceAccount Token 和仓库 URL 凭据校验；已注册到可选 build command 子循环。
- 凭据写入改为 AES-GCM `v1:` 密文，未配置 `CredentialEncryptionKey` 时平台 API 拒绝启动，构建配置不再允许凭据明文扩散。
- 已通过 `go test ./internal/cicd/build` 及 CI/CD 包编译检查。
- 新增 `migrations/007_cicd_project_delivery.sql`：旧项目统一迁移为
  `unclassified/pending_confirmation`，不从旧部署字段猜测类型；新增项目必须在创建时显式选择
  `k8s_yaml`、`helm` 或 `host`。
- 落地项目交付类型 API：`GET/PUT /api/v1/projects/{id}/delivery-profile`。仅有
  `project:update` 范围权限的用户可对旧项目确认一次类型；确认后类型不可修改，普通项目更新接口
  不接受该字段。
- 新增 `migrations/008_cicd_service_revision.sql`，在保留旧 `app_service` 的前提下增加服务交付类型、
  非敏感来源引用、当前 revision 和组件数量字段，并提供并发安全的 revision 编号分配表和不可变
  `service_revision`。
- 新增项目内服务创建与 revision API。新服务类型由项目类型派生，客户端不能把 YAML 项目服务改成
  Helm/主机；旧 `unclassified` 服务不能创建可执行 revision。revision 创建和查询分别校验
  `service:update`、`service:view` 的项目/服务范围权限。
- 新增版本化数据库迁移器和独立 `axiops-migrate` 命令。开发/测试可在服务启动时自动迁移；生产可
  关闭自动 DDL 并在发布前独立迁移。待执行迁移、迁移校验和漂移或迁移失败时服务拒绝启动。
- 将缺失的权限基线 004 纳入受管迁移，并通过 `information_schema` 幂等补齐字段和索引；已有
  001-003 数据库无需重建即可继续升级。
- 新增 `009_cicd_project_environment.sql` 和项目环境 API。YAML/Helm 环境校验项目类型、集群状态、
  集群权限及 Namespace；生产环境需要专用权限并强制审批。主机环境在统一主机组授权完成前保持
  `pending_validation`，不能用于部署。
- 创建环境时事务内创建默认变量组；变量更新使用 revision 乐观锁，Secret 只允许凭据引用，查询不返回明文。
- `migrations/README.md` 固化全局编号、模块命名、自动/手工执行和禁止破坏性启动迁移的规则。
- 新增 `010_cicd_environment_service_override.sql` 和环境服务覆盖 API，固定目标服务 revision、镜像和
  副本数；校验环境/服务同项目、交付类型兼容、revision 归属、双重授权和乐观锁。失效环境允许只读
  历史覆盖，但不能写入新的可部署状态。
- 新增 Helm Chart Git 导入预览与 revision API：校验 Chart 目录边界、Chart v2 application 元数据、
  values 敏感默认值、commit 一致性和 4 MiB 快照上限，保存包含完整文本文件和逐文件 SHA-256 的确定性定义。

## 进行中

- 阶段 0：项目类型、服务 revision、项目环境、变量组、服务覆盖、环境克隆、工作流 DAG、审批和任务租约已完成。
- 阶段 1：代码能力已完成，等待真实 Kubernetes、镜像仓库和大规模并发端到端验收。
- 阶段 2：Chart/values 导入、官方 Helm v3 渲染/发布/回滚、执行恢复、revision 审计和资源 Diff 已完成；待真实集群端到端验收。
- 阶段 3：主机 Agent、制品校验、探活、回滚和滚动批次后端已完成；待真实主机组端到端验收和前端。
- 阶段 4：通知 outbox 与主流渠道后端已完成；待 CI/CD 项目内前端和效能指标。
- 补充大规模构建并发限制、日志外部存储和真实镜像仓库 digest 验证。
- 增加认证的 provider 回调接口和超时扫描任务。

## 待完成

- Helm/YAML 真实集群多环境发布、预览和回滚演练。
- 浏览器交互、权限矩阵和三类项目完整基础设施 E2E。

## 兼容与下线策略

本阶段属于渐进式重构，不是一次性重写。旧 `build_record`、旧 API 和旧部署清单仍可能
被引用，不能删除。完成 provider 流量切换、历史数据校验、监控观察期和回滚演练后，
再提交独立的下线变更。

## 验证记录

### 2026-08-19 权限验收收口与流水线前置盘点

- 新增 `scripts/test-access-acceptance.ps1`，通过真实前端代理登录四类账号，只读验证项目范围、菜单范围和开发账号跨项目 HTTP 403。
- 新增 `ACCEPTANCE_RUNBOOK.md`，固化本地账号、三类验收项目、逐步验收流程、完成标准和外部资源前置条件。
- 当前验收库存在 YAML、Helm、主机三个项目及对应服务，但没有代码源、环境、工作流和已连接 Kubernetes 集群；因此真实部署 E2E 仍为待完成，不能标记通过。
- 记录历史孤儿服务风险：`app_service.id=1` 指向不存在的项目 1，完成数据核对和约束迁移前不删除、不复用该项目 ID。

验证命令：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-access-acceptance.ps1
```

预期：四类账号范围检查全部通过，开发账号读取 Helm 项目和服务均返回 HTTP 403。

### 2026-08-19 YAML 项目服务配置入口修复

- 项目服务列表新增“编辑”入口，管理员或具备 `service:update` 范围权限的用户可补充 Git 仓库、分支、代码源、镜像和技术栈信息。
- 编辑态锁定服务编码，避免已经被工作流、授权和审计引用的稳定标识发生变化；创建态继续执行原有编码校验。
- 新增前端 `updateServiceApi`，复用现有 `PUT /api/v1/services/{id}`，现有验收服务无需删除重建即可绑定后续创建的代码源。
- K8s 环境创建弹窗在没有可见且已连接集群时显示明确的资源状态，避免空下拉框被误认为页面故障。

验证结果：

| 检查项 | 结果 |
|---|---|
| 服务更新 API 写入、读取并恢复原值 | 通过 |
| 本次文件 Prettier / ESLint | 通过 |
| 本次文件 `vue-tsc` 定向检查 | 通过 |
| 前端生产构建 | 通过，6239 个模块 |
| 全仓 `vue-tsc` | 未通过，旧前端模块仍存在历史类型错误；本次文件无错误 |

下一步：接入一个真实 Git 代码源和 Kubernetes 集群，完成 YAML revision 导入、环境渲染预览和首条 `build -> yaml_deploy` 工作流。

### 2026-08-12 本轮

- 完成 Webhook 安全入口：独立 `git_webhook_secret` 加密凭据、GitLab Token 常量时间比较、投递幂等记录、分支包含/排除规则和代码源绑定校验。
- 完成 `GET /api/v1/builds/{id}/logs/stream` SSE，复用 `build_log:view` 权限并支持 `Last-Event-ID` 重连游标。
- 新增迁移 `020_cicd_webhook_delivery.sql`，本轮未执行数据库变更。
- `gofmt` 通过；完整 `go test` 被当前环境阻断：Go 缓存缺少 `github.com/golang-jwt/jwt/v4@v4.5.2` 且网络代理不可达；未连接业务数据库或 Kubernetes 集群。

```text
go test -count=1 ./internal/cicd/build                PASS
go test -count=1 ./internal/shared/secretbox          PASS
go test ./internal/platform/credential ./cmd/platform-api ./cmd/cicd-api  PASS (compile)
go vet ./internal/shared/secretbox ./internal/platform/credential ./internal/cicd/... ./cmd/platform-api ./cmd/cicd-api  PASS
go test -count=1 ./internal/platform/project              PASS
go test -count=1 ./internal/cicd/appsvc                   PASS
go test -count=1 ./migrations                             PASS
go test -count=1 ./internal/cicd/projectenv               PASS
go test -count=1 ./internal/shared/authz                  PASS
go test -count=1 ./cmd/platform-api                       PASS (compile; no test files)
go test -count=1 ./cmd/cicd-api                           PASS (compile; no test files)
go test -count=1 ./cmd/axiops-migrate                     PASS (compile; no test files)
go vet ./migrations ./internal/shared/config ./internal/platform/project ./internal/cicd/appsvc ./internal/cicd/build ./cmd/axiops-migrate ./cmd/platform-api ./cmd/cicd-api  PASS
git diff --check                                          PASS

Note: ./internal/shared/middleware test execution is intermittently blocked by
Windows Application Control in this environment; compile-only verification passes and the package was not changed in this stage.
```
# 阶段 0 增量记录（2026-08-11）

- 环境克隆 API、事务复制和权限边界已完成；生产环境不能作为源或目标，Secret 变量不复制。
- 工作流 DAG 声明式模型和校验器已完成，支持 build、YAML/Helm/主机 deploy、test、approval、notify 节点，并拒绝缺失依赖、环和类型不兼容。
- 新增迁移 `011_cicd_workflow.sql`，当前只完成定义表，运行快照和执行器将在后续迁移中增加。
- 新增迁移 `012_cicd_permission_defaults.sql`，撤销内置 developer 的 `build:run` 和 `workflow:run` 默认权限，符合开发人员只读策略。
- 新增工作流运行快照 API：运行时固化定义版本、环境上下文和节点快照；生产运行必须包含审批节点，支持受限取消。
- 新增生产运行审批 API 和迁移 `014_cicd_workflow_governance.sql`；审批后运行从 `pending_approval` 回到 `pending`，审批节点标记成功。
- 新增无副作用 DAG 调度计划器，支持并行就绪、依赖等待、失败跳过和完成判断，并已增加单元测试。
- 新增 worker 执行框架、executor 注册表和任务租约，支持节点领取、成功/失败回写及租约过期恢复；具体 provider executor 下一步接入。
- 新增异步 executor 生命周期和 build executor；构建节点复用现有 `build_run/build_command/kube_job` 链路，提交成功只标记运行中，直到构建终态才推进 DAG。
- 新增 YAML 部署 executor、清单解析安全校验和 SQL 数据解析层；部署只读取不可变服务 revision 与已验证项目环境，不把 kubeconfig 或 Secret 写入快照。
- 新增 Kubernetes YAML provider 抽象，支持按资源 apply、资源级预览和同步成功状态；真实多集群 client factory 可按集群接入。
- 新增迁移 `016_cicd_deployment_snapshot.sql` 和 YAML 部署快照存储：provider apply 前保存已校验清单及 SHA-256，成功后标记 `applied`；Secret 不进入快照，只有已应用且摘要校验通过的快照可作为后续回滚输入。
- 部署快照与 YAML executor 定向测试、迁移器测试和权限测试通过；本轮未连接数据库、未执行 SQL。
- 快照补充工作流任务、节点、集群和 Namespace 精确身份；同一运行内同一服务的多个部署节点不会相互覆盖。
- 新增 YAML 回滚 executor 和 `POST /api/v1/deployment-snapshots/{id}/rollback`：只接受已应用且摘要一致的 YAML 快照，按项目/服务/环境/集群范围授权，生产环境自动进入审批，系统运行支持取消、租约和异步回查。
- 新增迁移 `017_cicd_deployment_rollback.sql`：回滚运行保存操作类型、来源快照和唯一请求键，并只为 project_admin、platform_admin、operator 增加默认回滚权限；developer 仍为只读。
- 回滚逻辑、调度、executor、迁移器和权限测试通过，`cmd/cicd-api` 完整包编译通过；本轮未连接数据库、未执行 SQL。
- 新增 `axiops-workflow-worker` 独立进程和配置，注册 build、YAML deploy、YAML rollback executor；支持立即轮询、任务租约、错误后继续运行、SIGTERM/中断优雅停止，且启动时只做迁移校验、不自动执行 SQL。
- 新增真实多集群 Kubernetes client factory：按集群读取加密 kubeconfig，设置 QPS/Burst/请求超时和客户端缓存，通过 discovery RESTMapper 执行动态资源 Server-Side Apply 与 dry-run 预览。
- 修复集群凭据写入边界：`platform-api` 创建/更新集群时先验证再 AES-GCM 加密 kubeconfig，手动连通性测试在内存解密；历史明文记录必须由管理员重新保存，worker 不接受明文降级。
- 本地构建脚本已加入 `axiops-workflow-worker.exe`。工作流、worker、集群加密、Platform/CICD API、迁移器和权限测试及 `go vet` 均通过；本轮未连接业务数据库、未执行 SQL。
- build command 子循环已接入 `kube_job` provider：触发时冻结仓库和构建配置，submit 领取原子进入 dispatching，Job 创建后进入 running，Job condition 回传 success/failed/timeout，Pod 名称和节点用于后续日志定位。
- 修复构建入队后立即取消的命令阻塞问题；无外部 Job 时直接确认取消并清理失效 submit。构建 processing 命令和未提交工作流任务均已接入过期租约回收，worker 崩溃后可按稳定幂等键重试。
- 新增迁移 `018_cicd_build_observability.sql`：为新 worker 日志增加可空唯一 `log_key` 和截断标记；历史日志保持不动，避免上线迁移删除或强制去重数据。
- 构建日志由 worker 读取 builder 容器并保存最多 1 MiB 的幂等阶段快照；镜像 digest、大小及二进制制品 URI/大小通过 termination message JSON 校验后回写，拒绝非法 digest 和不受支持的 URI。
- 仓库凭据改为运行时读取 `code_source.credential_id` 指向的统一 `git_token` 密文，临时创建 Kubernetes Secret，Job Spec 只包含 `secretKeyRef`，取消或终态后清理；安全测试确认 Token 不进入 Job JSON。
- 发现旧 `internal/cicd/codesource` 使用与 `002_asset_management.sql` 不一致的 `token/base_url` 字段和旧 AES 格式；新 worker 不依赖该实现，后续必须统一重构代码源管理 API，禁止继续扩散旧密文格式。
- 已重构代码源管理 API，直接匹配 `002_asset_management.sql` 的 `url/credential_id` 模型；删除旧 Token 加密和不存在的仓库缓存表访问。CRUD、分支、标签、提交查询全部按项目范围授权，绑定或使用凭据还需 `credential:view`。
- 新增迁移 `019_cicd_code_source_permissions.sql`：developer/tester/viewer 默认只读，project_admin/platform_admin/operator 可管理；CI/CD API 未配置 `CredentialEncryptionKey` 时拒绝启动。
- GitLab provider 可用于仓库查询；GitHub、Gitee、Gitea 仍为明确的未支持状态。代码源 check 当前只验证配置和凭据，不伪装成远端探测成功；Webhook 旧实现尚未注册，下一步按统一凭据和签名模型重构。

## 2026-08-17 本地启动修复与验证

- Platform API、CI/CD API 和 Workflow Worker 统一按“环境变量、YAML、本地密钥文件”顺序读取凭据加密密钥；本地使用被 Git 忽略的 `local/credential-encryption.key`，生产环境仍必须通过 Secret 注入。
- 新增 `scripts/init-local-secrets.ps1`，密钥只初始化一次；已有加密凭据后禁止直接轮换，否则历史凭据将无法解密。
- Workflow Worker 通过 `DatabaseConfigFile: configs/platform-api.yaml` 复用 Platform API 的 `MySQL` 配置，避免本地配置漂移和数据库密码重复保存；Worker 仍只校验迁移，不自动执行 DDL。
- 数据库实测：`axiops-migrate -check-only` 返回 `current=023_cicd_host_artifact_integrity.sql applied=[]`，无需补执行 SQL。
- 运行实测：Platform API 和 CI/CD API 健康接口均返回 `status=ok`；Workflow Worker 通过迁移检查并输出 `AxiOps workflow worker started`。
- 验证通过：`go test -count=1 ./internal/shared/config ./cmd/platform-api ./cmd/cicd-api ./cmd/workflow-worker`、`go vet`，以及三个服务的本地二进制构建。
- 新增简洁的统一开发启动入口 `scripts/start-dev.ps1`：先构建到固定 `bin/` 目录并检查迁移，再分别打开 Platform API、CI/CD API 和 Workflow Worker 三个可见终端，避免 Windows Application Control 拦截 `go run` 临时程序。

## 2026-08-18 Helm Chart 导入

- 新增 `POST /api/v1/services/{id}/helm-import/preview` 和 `POST /api/v1/services/{id}/helm-import`，仅允许 Helm 服务使用其绑定代码源。
- 导入 revision 必须携带预览 commit；服务端重新读取固定 commit 后创建不可变 `helm_chart` definition，不信任客户端上传 Chart 内容。
- 定向测试覆盖 Chart 元数据、values 和镜像摘要、文件排序、目录逃逸、跨目录文件、明文密码、library Chart、commit 漂移及非 Helm 服务拒绝。
- 本功能未新增数据库迁移；当前数据库仍以 `023_cicd_host_artifact_integrity.sql` 为最新版本。
- 验证通过：`go test -count=1 ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./migrations`、对应 `go vet`、CI/CD API/Worker/迁移器构建及数据库 `-check-only`。
- 前端依赖恢复仍受网络审批服务阻塞，Vite/Rollup 未安装，因此本轮没有提交无法构建验证的 CI/CD 页面。

## 2026-08-18 主机服务定义导入

- 新增 `POST /api/v1/services/{id}/host-import/preview` 和 `POST /api/v1/services/{id}/host-import`，仅允许主机服务使用其绑定代码源。
- 新增 `axiops.io/v1alpha1 HostService` 定义：支持 build/static 制品来源、Linux/Windows、deploy/health/rollback 脚本引用和执行超时。
- 三个脚本必须来自同一固定 Git commit；服务端校验相对路径、OS 扩展名、UTF-8、大小和 SHA-256 后冻结脚本正文，预览接口不返回正文。
- 主机工作流按 revision 的 `os` 精确筛选同项目、同主机组的在线 Agent，避免把 Linux 脚本下发到 Windows 主机或反向下发。
- 本功能未新增数据库迁移；旧的无 `os` 主机 revision 不再部署，需通过导入接口生成新 revision。
- 验证通过：`go test -count=1 ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./cmd/axiops-host-agent ./migrations`、对应 `go vet`、四个程序构建和数据库 `-check-only`。

## 2026-08-18 主机滚动批次

- 新增迁移 `024_cicd_host_rolling_batch.sql`：增加滚动控制记录、命令批次、可领取时间和停止原因；迁移会为旧命令回填单批兼容记录，不删除历史数据。
- `HostService` 增加 `rollout.batchSize`、`rollout.batchPercent` 和 `rollout.intervalSeconds`，固定数量与百分比互斥，默认每批 1 台。
- 目标 Agent 按 ID 稳定排序并计算摘要；相同工作流任务重试时若目标集合或策略漂移，拒绝改变已创建的发布范围。
- 首批命令可领取，后续批次保持 `blocked`；当前批全部成功后在事务中释放下一批。失败或 Agent 完成自身回滚时，将尚未领取的命令标记为 `cancelled`。
- Agent 终态上报支持幂等重试；滚动控制行使用 `FOR UPDATE` 串行化同一任务的并发上报，避免重复放行。
- 定向验证通过：`go test -count=1 ./internal/cicd/hostagent ./internal/cicd/workflow ./internal/cicd/hostimport ./migrations`。本轮未连接业务数据库，`024` 尚未执行。

## 2026-08-18 官方 Helm v3 Provider

- 接入与 Zadig 参考一致的官方 `helm.sh/helm/v3 v3.18.5`，未实现自研 Helm 模板引擎。
- 新增 SQL resolver：严格校验项目、服务 revision、环境、集群和 Namespace 归属；按项目、环境、服务顺序合并 values。Secret 仅接受同项目或全局 `secret` 凭据，运行时在内存解密。
- 渲染快照只保存凭据 ID 和资源身份清单，不保存 Secret 明文或完整 Kubernetes 对象；项目发布拒绝 ClusterRole、CRD 等集群级资源。
- Worker 已注册 Helm deploy 和系统 Helm rollback executor。发布使用 atomic/wait/wait-for-jobs，成功记录 release revision；生产回滚继续经过统一审批和 `deployment:rollback` 范围权限。
- 新增迁移 `025_cicd_helm_execution.sql`：扩展部署快照的 release/revision 字段，并增加稳定幂等键、请求摘要、超时恢复和终态记录。
- 单元测试覆盖官方模板渲染、values 覆盖、Chart 文件摘要、Secret 脱敏、跨项目凭据、集群级资源拒绝和回滚范围隔离。
- 验证通过：`go test -count=1 ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./cmd/axiops-host-agent ./migrations`、对应 `go vet` 和 `go mod verify`。本轮未连接 Kubernetes 或业务数据库，`024-025` 尚未执行。

## 2026-08-18 构建治理、日志归档与产物校验

- 新增迁移 `026_cicd_build_governance.sql`：支持平台和项目两级构建并发、排队超时、默认 50 并发，并增加构建效能查询索引。
- 新增构建效能和治理 API：成功率、成功/失败/取消数量、平均/最大耗时、每日趋势，以及并发策略查询和更新。
- 构建日志支持可选 MinIO 归档；对象存储凭据不进入 YAML 或数据库，数据库仅保留 64 KiB 尾部，读取完整日志前继续执行 `build_log:view` 授权。
- 新增迁移 `027_cicd_build_output_verification.sql`：持久化镜像、主机制品的独立校验状态、失败原因和校验时间。
- 主机制品通过 HTTPS 流式计算实际 SHA-256 和大小，校验失败会把构建置为失败；主机发布只允许使用 `verified` 构建制品。
- 镜像仓库 manifest digest 校验支持 Bearer/Basic 认证并复用加密仓库凭据，通过 `AXIOPS_BUILD_VERIFY_IMAGE` 显式开启。
- 权限前端已确认属于本次交付范围，包含用户、部门、角色、菜单以及项目、服务、环境、集群、构建和工作流资源范围配置，不作为后续可选项。
- 验证通过：`go test -count=1 -buildvcs=false ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./cmd/axiops-host-agent ./cmd/axiops-migrate ./migrations` 和对应 `go vet`。本轮未执行 SQL，数据库仍由用户在后端完成后统一迁移。

## 2026-08-18 项目工作区与权限前端

- 新增 CI/CD 项目列表，创建时明确选择 K8s YAML、Helm Chart 或主机应用。
- 新增项目工作区，按“概览、服务、环境、工作流、设置”组织；支持服务、K8s/主机环境和 build/approval/deploy 工作流基础创建。
- 生产环境自动强制发布审批；Helm 工作流生成 `helm_deploy` 节点，主机工作流生成 `host_deploy` 节点。
- 新增项目工作流列表接口并接入前端；旧 DevOps 菜单从新路由入口移除，旧源码保留到完整 E2E 后再决定删除。
- 权限中心同步完成，工作流可以按单个工作流 ID 授权，不会退化为整个项目权限。
- 前端生产构建通过；后端 CI/CD、平台和共享包完整测试与 `go vet` 通过。当前环境没有浏览器实例，截图与真实交互 E2E 未执行。

## 2026-08-18 工作流运行闭环

- 新增 `GET /api/v1/workflows/{id}/runs`，按工作流范围权限分页查询历史运行。
- 运行详情补充任务 `started_at`、`finished_at`；worker 领取任务时写入开始时间。
- 前端工作区增加执行、运行记录、运行详情抽屉、取消和生产审批操作；按钮按工作流资源范围权限控制。
- 构建节点可从运行详情打开构建日志，运行中每 3 秒轮询状态。
- 本轮未新增数据库迁移，复用 `013_cicd_workflow_run.sql` 已有字段；数据库仍需按现有迁移器状态执行后续 `024-028`。
- 验证：workflow 单元测试、CICD API 编译、前端 ESLint 和生产构建通过。

## 2026-08-18 代码源与三类服务导入前端

- 项目工作区增加代码源页，支持创建加密 `git_token` 凭据、创建 GitLab 代码源和配置校验。
- 新建服务可绑定项目代码源；只有具备服务更新权限的用户可以打开定义导入。
- YAML、Helm Chart、主机服务统一采用“仓库路径 + ref + 文件清单 -> 预览 -> 固定 Commit -> 创建不可变 Revision”流程。
- 预览展示资源、组件、镜像、Chart 元数据或主机脚本摘要，不在前端持久化 Token 和完整服务定义。
- 本轮未新增 SQL；前端 ESLint 和生产构建通过，生产构建完成 6241 个模块。

## 2026-08-18 构建与环境配置前端

- 服务列表新增构建配置入口，支持自定义、Go、Maven、Gradle、Node.js、Python 模板，以及构建镜像、命令、Dockerfile、上下文、镜像仓库、缓存、资源限制、超时、分支规则和环境变量配置。
- 构建配置按服务资源范围执行 `build_config:view/update` 权限判断；只读用户可以查看，但看不到保存操作。当前仅开放已接入真实 worker 链路的 `kube_job` provider。
- 环境列表新增环境配置入口，完成默认环境变量组读取和更新；普通变量保存文本，Secret 变量只能选择统一凭据 ID，前端和后端均不读取或回显明文。
- 环境变量使用服务端返回的 `revision` 乐观锁；环境服务覆盖支持不可变 Service Revision、镜像和副本数，并按首次 `expected_revision=0`、后续返回 revision 保存。
- 环境配置写操作按环境资源范围执行 `environment:update` 权限判断，服务覆盖继续由后端复核环境和服务双重权限、项目归属和类型兼容性。
- 本轮未新增 SQL；数据库后续仍按既有顺序执行 `024-028`。
- 验证通过：新增页面 Prettier、ESLint、前端生产构建（6247 modules）、`go test -count=1 ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./migrations` 和对应 `go vet`。
- 当前环境未提供可用浏览器会话，也未连接真实 Git、MySQL、Kubernetes、镜像仓库或主机 Agent，因此未把真实交互和基础设施 E2E 标记为通过。

## 2026-08-18 工作流 DAG 可视化编排

- 项目工作区新增基于现有 `@vue-flow/core` 的工作流编排弹窗；按工作流资源范围支持只读查看或编辑，不具备 `workflow:update` 的用户看不到新增、删除和保存操作。
- 支持拖动节点、添加构建/审批/当前项目类型部署节点、连线和删除依赖、删除节点，以及编辑服务、分支、Service Revision、Helm Release、超时和重试参数。
- 新连线在前端拒绝自依赖、重复依赖和成环；保存前先调用 `POST /api/v1/workflows/{id}/validate`，通过后使用 `expected_version` 乐观锁调用更新 API。
- 当前只开放 worker 已有真实执行闭环的 build、approval、yaml/helm/host deploy；后端虽然允许 test/notify 定义，但 worker 尚未注册对应 executor，因此前端暂不开放这两类节点。
- 本轮未新增 SQL；验证通过：相关文件 Prettier、ESLint 和前端生产构建（6250 modules）。后端工作流与迁移全量测试、`go vet` 已在同轮前一阶段通过。
- 当前没有可用浏览器会话，未执行拖拽、连线和移动端真实交互 E2E，不能将该项标记为浏览器验收通过。

## 2026-08-18 通知订阅与投递结果

- 新增 `GET /api/v1/notification-deliveries`，按 `notification:view` 项目范围授权，支持状态过滤和最大 100 条分页；返回订阅、渠道、事件、状态、尝试次数和失败原因，不返回 payload 或凭据。
- 项目工作区新增通知页，支持项目全部或仅当前触发用户订阅，以及飞书、钉钉、企业微信、邮件和通用 Webhook 渠道。
- 支持选择已有 `notification_webhook`/`smtp` 加密凭据；具备 `credential:create` 时可在订阅表单内创建，Webhook 强制 HTTPS，凭据明文只提交统一凭据 API，不进入订阅或投递记录。
- 支持成功/失败事件开关、删除订阅、投递状态过滤、分页、刷新、尝试次数和失败原因查看。构建与工作流终态已分别在现有事务中写入 outbox。
- 本轮未新增 SQL，复用已迁移的 `021_cicd_notification_outbox.sql`。
- 验证通过：通知权限与状态参数单元测试、`go test -count=1 ./internal/cicd/... ./cmd/cicd-api ./cmd/workflow-worker ./migrations`、对应 `go vet`、前端 Prettier/ESLint 和生产构建（6253 modules）。
- 当前未配置真实飞书/钉钉/企业微信/SMTP/Webhook 凭据，未执行外部渠道真实投递 E2E。

## 2026-08-18 公共错误响应修复

- 修复统一 HTTP 错误映射顺序：`CodeParamInvalid (50004)` 返回 400，`CodeNotFound (50005)` 返回 404，数据库和内部错误继续返回 500。
- 新增响应层单元测试覆盖三类状态，避免后续模块把客户端参数错误误报成服务端故障。

## 2026-08-19 YAML/Helm 发布前资源预览

- 新增 `GET /api/v1/environments/{id}/render-preview`，支持显式 Service Revision；未指定时按环境覆盖、服务当前 Revision 的顺序解析。
- 接口同时复核 `environment:view`、`service:view` 和 `cluster:view`，并校验项目、环境、服务、Revision、集群与 Namespace 归属。
- YAML 和 Helm 均在目标集群执行 Server-Side dry-run，资源动作区分 `create/update/unchanged`；Helm 根据同一 Release 上一成功快照补充 `delete`。
- YAML 当前发布语义为 SSA 且不执行 prune，因此不会把已移除资源错误标记为本次会删除。Helm 删除清单读取前校验快照 SHA-256。
- Helm Secret 只在内存参与真实模板渲染和集群校验；API 仅返回资源身份与脱敏 `credential_id` values，不返回 manifest、明文或密文。
- 环境配置前端新增“资源预览”页签，支持服务、Revision、Release 选择、四类变化汇总、资源明细和脱敏 values 展示；只读用户也可在具备三类查看权限时使用。
- 本轮未新增 SQL。后端全量 CI/CD 测试及 `go vet` 通过；新增前端文件 Prettier、ESLint、Stylelint 和生产构建通过，构建完成 6253 个模块。
- 项目级 `vue-tsc --noEmit` 仍被仓库既有跨模块类型错误阻断，本次 `EnvironmentConfigDialog.vue` 未出现在错误列表；当前无可用浏览器会话和真实 Kubernetes，未把真实交互或集群 E2E 标记为通过。

## 2026-08-19 API 路由参数与前端服务分流

- 前端 Platform 资源请求统一使用 `/platform/v1`，CI/CD 运行资源统一使用 `/cicd/v1`；开发代理分别
  转发到 8090 和 8091，修复原 `/api/v1` 无法选择目标服务的问题。
- Platform 项目、集群、凭据继续由 Platform API 管理；服务、代码源、构建、环境、工作流和通知由
  CI/CD API 管理。
- Platform API 与 CI/CD API 注册全局 `SyncPathValues` 中间件，修复 go-zero `:id` 参数无法被
  Go 标准 `Request.PathValue` 读取的问题。该问题此前会阻断服务详情、工作流详情、取消、审批、
  环境配置以及所有带资源 ID 的操作。
- 中间件单元测试通过；经前端代理请求不存在的服务、工作流和项目 ID，均正确进入资源查询并返回
  “资源不存在”，不再错误返回“无效 ID”。

## 2026-08-20 Kubernetes 集群管理闭环

- 新增 AxiOps 原生“集群管理”页面：集群接入、编辑、连接测试、状态筛选、版本/节点/CPU/内存展示和删除保护。
- kubeconfig 在浏览器端校验 `apiVersion`、clusters、users、contexts、current-context 及当前 API Server；API Server 可从 kubeconfig 自动填充。
- kubeconfig 仅提交至后端加密存储，列表、详情和错误响应均不回显明文或密文。
- 权限中心已支持集群范围角色绑定；后端每次集群查看、编辑、测试、删除和环境创建都会复核 cluster scope。
- 项目工作区请求 `/platform/v1/clusters?project_id=...` 时，项目范围授权只能看到已被该项目环境引用的集群；集群管理页不带项目参数，不会因项目授权泄露全局集群。
- 已被 `project_environment` 或 `release_record` 引用的集群禁止删除，避免破坏交付链路。
- 集群连接探测增加 15 秒超时；CPU 按 millicore 汇总、内存按字节换算，避免容量整数截断；空权限结果稳定返回 JSON `[]`。
- 本轮未新增 SQL，复用当前迁移至 `028_access_department_workflow_scope.sql` 的表结构。
- 已验证：`go test ./internal/platform/cluster ./internal/shared/authz`、集群页面 Prettier/ESLint；真实 Kubernetes 连接和 YAML 部署 E2E 待接入用户提供的测试集群 kubeconfig 后执行。
# 2026-09-06 Code source consolidation

- Added platform-level code source creation and routing; `project_id=0` is the
  reusable company connection scope.
- Added GitLab, Gitea, GitHub and Gitee repository adapters, including internal
  HTTP endpoints and GitHub Enterprise API path handling.
- Added the `CI/CD -> Code Sources` page and service form repository selection.
- Added permission-safe global source reuse for project-scoped developers;
  platform mutations remain restricted.
- Added REST mock tests and recorded the AI RCA design in
  `CODE_SOURCE_AND_AI_RCA.md`.
- Connection testing now performs a real repository-list request instead of
  only constructing a provider.
- Project credential lookup now includes reusable platform Git tokens while
  platform credential listings remain restricted to platform rows.
- Service create/update now rechecks code-source visibility and source scope,
  allowing platform sources but blocking references to another project.
- Rebuilt and restarted the local API binaries after adding the platform source
  route; temporary admin smoke test created and removed a credential and source
  successfully. Runtime logs are under `local/dev/`.
- No SQL changes were introduced in this stage.

## 2026-09-06 Internal Git provider connection fix

- Root cause confirmed: the local PowerShell environment sets `HTTP_PROXY`,
  `HTTPS_PROXY` and `ALL_PROXY` to `127.0.0.1:9`; the Gitea endpoint is an
  RFC1918 address, so the default Go client attempted to use the dead proxy.
- REST Git providers now bypass the environment proxy for private, loopback
  and link-local IP addresses while retaining proxy support for public GitHub,
  GitLab and Gitee endpoints.
- Added a regression test covering private-IP bypass and public-provider proxy
  selection. `go test ./pkg/gitprovider ./internal/cicd/codesource ./cmd/cicd-api`
  and `go vet ./pkg/gitprovider ./internal/cicd/codesource` pass.
- Project workspace no longer exposes a code-source management tab or create
  action. Sources are managed once at platform scope; project services only
  select an already-authorized shared source.
- Local services were rebuilt and restarted. Platform API, CI/CD API, worker
  and web health checks passed. No SQL changes are required.
- A live check against the existing source returned the precise provider error
  `代码源认证失败，请检查 Token 权限`; replace the Gitea token if this source
  is expected to connect. The API no longer reports this as a generic internal
  error.
- Frontend Prettier and ESLint pass. Vite production build remains blocked by
  the existing Windows Application Control `spawn EPERM` policy; this is an
  environment limitation, not a TypeScript or lint failure.

## 2026-09-06 Code source editing and log encoding

- Added platform code-source edit controls for name, provider type, URL,
  encrypted credential reference and active/inactive status.
- Reused the existing authorized `PUT /api/v1/code-sources/{id}` endpoint and
  added the missing frontend API binding. Tokens are never loaded into the edit
  form; rotating a token requires selecting an existing credential or creating
  a new encrypted credential.
- Fixed idempotent MySQL updates: `RowsAffected=0` now succeeds when the row
  exists and values were unchanged. Update failures are logged with source and
  user IDs instead of being silent.
- Updated local startup output and core service logs to UTF-8-safe ASCII
  messages. `scripts/start-dev.ps1` sets UTF-8 console/output encoding.
- Verification: backend tests, `go vet`, frontend Prettier/ESLint, service
  health checks and a live idempotent platform code-source update all pass.
- No SQL changes are required.

## 2026-09-06 Service repository selector

- New service dialogs automatically select the only available platform code
  source and load repositories immediately.
- Empty and failed repository loads now show an actionable message instead of
  presenting an unexplained empty dropdown.
- Gitea repository discovery now uses its native `limit/page/affiliation`
  query and falls back to organization repository enumeration for older Gitea
  versions that omit organization repositories from `/user/repos`.
- Live verification against the configured `sinozo` source changed from zero
  repositories to one visible repository. Backend tests, `go vet`, frontend
  Prettier/ESLint and all local health checks pass.
- CI/CD API and local services were rebuilt and restarted. No SQL changes are
  required.

## 2026-09-07 工作流目标与运行详情 P0 完成

- 执行工作流时只展示当前用户可见且已验证、已启用、交付类型匹配的环境，不生成固定的开发/测试/预发/生产环境。
- 目标选择展示环境名称、集群与 Namespace 或主机组；生产环境明确提示需要审批，无可用环境时引导进入环境配置。
- 运行记录不再只显示环境 ID，增加环境名称和资源目标，便于在多集群项目中确认实际发布位置。
- 运行详情增加发布目标、触发方式、服务、构建编号、分支、Commit/变更链接、镜像与 digest、构建阶段、Job/Pod/节点、部署执行标识和构建日志入口。
- 构建详情请求复用现有 `GET /cicd/v1/builds/{id}`；工作流任务先于构建记录可见时，由轮询静默重试，避免误报系统错误。
- 修复工作流编排器已有的 Vue Flow 可选节点数据类型检查问题；CI/CD 项目工作区定向 `vue-tsc` 无错误。
- 验证通过：目标文件 Prettier、ESLint；全量 `vue-tsc` 仍被仓库其他旧模块错误阻断，本次 CI/CD 项目工作区未出现在错误列表。
- 本轮未新增 SQL、API 或后端运行逻辑。Vite 开发服务可热更新前端；后端不需要重启。

## 2026-09-08 CI/CD 页面主题可读性修复

- 修复 CI/CD 页面使用但未定义的主题变量：`art-bg-color`、`art-main-bg-color`、
  `art-border-color` 以及灰阶文本变量。此前变量解析失败会导致工作区出现白底白字、
  边框和背景丢失等问题。
- 为亮色和暗色主题分别提供兼容别名，确保项目列表、项目工作区、服务编辑器及弹窗
  使用同一套可读的颜色语义。
- 项目工作区首次加载遮罩改用主题 overlay，工作区和服务目录背景改用主题变量，避免
  暗色主题被硬编码的浅色背景覆盖。
- 验证：`npm.cmd run build`（Vite 6262 modules）通过；本轮无 SQL 变更。
- 前端代码修改后需要重启/刷新前端开发服务；后端无需重启。

## 2026-09-08 环境详情页 P0

- 在项目工作区新增“环境”页签，支持在项目已有环境之间切换，并保留新建环境入口。
- 新增环境信息卡：环境类型、Namespace、镜像仓库占位、更新时间、环境状态、生产标识及
  更新资源、管理服务、环境配置、更多、立即部署操作区。
- 新增服务列表工具栏：名称搜索、服务总数、新增服务、列表/资源视图切换。
- 新增服务表格：服务名称、运行状态、版本镜像、外网访问、重启、重新部署、历史版本入口。
- 服务行数据复用现有环境覆盖接口；未绑定覆盖的服务显示为待配置，不隐藏服务。
- 本轮未新增 SQL；前端需刷新或重启 Web 开发服务，后端无需重启。
- 验证通过：Prettier、ESLint、Vite 生产构建（6262 modules）。

## 2026-09-08 服务配置页语义调整

- 将项目页签从“服务与环境”调整为“服务配置”，服务配置页不再展示环境发布矩阵。
- 服务配置页仅保留服务目录、服务定义、Revision、构建产物和操作菜单；环境目标统一进入“环境”页签。
- 顶部新增“模板导入”和“Git 导入”快捷入口，左侧服务目录 `+` 继续用于添加服务。
- 服务创建向导增加来源预选：模板导入直接进入模板选择，Git 导入直接进入仓库选择，手工添加仍可通过目录 `+` 选择。
- 本轮无 SQL；前端需刷新或重启 Web 开发服务，后端无需重启。
- 验证通过：Prettier、ESLint、定向 vue-tsc、Vite 生产构建（6262 modules）。
- 新建项目进入项目工作区后默认显示“概览”，不再直接落到服务配置页。

## 2026-09-09 服务目录式创建入口修正

- 按 Zadig 服务树交互修正：服务目录左侧 `+` 直接创建手工服务目录，不再打开来源导航弹窗。
- 新建目录下展示 `deployment.yaml`、`service.yaml` 文件节点，并支持在目录内联添加 YAML 文件。
- 模板导入、Git 导入改为服务配置页顶部独立入口，分别进入模板选择和仓库选择。
- 服务配置页继续只负责服务定义、文件、Revision 和构建，不混入环境配置。
- 本轮无 SQL；后端无需重启，前端需刷新/重启开发服务。
- 验证通过：Prettier、ESLint、定向 vue-tsc。

## 2026-09-09 服务配置单页编辑器修正

- 移除服务创建器中的“项目配置 / 创建环境 / 执行工作流”向导步骤；项目首次向导与服务编辑页面彻底分离。
- 服务创建器现在只负责服务目录、YAML/Helm 文件、变量、Revision 和保存服务。
- 保存按钮改为“保存服务”，不会在服务编辑页面隐式创建环境或工作流。
- 服务配置页支持删除服务，删除前二次确认，并调用现有服务删除 API。
- 本轮无 SQL；后端无需重启，前端需要刷新/重启 Web 服务。
- 验证通过：Prettier、ESLint、定向 vue-tsc、Vite 生产构建（6262 modules）。

## 2026-09-09 服务文件树与三栏工作台

- 服务配置页改为左侧服务目录 + 右侧服务工作台，隐藏原来的服务环境矩阵，环境发布统一在“环境”页处理。
- 服务工作台增加文件树，支持在同一个服务内切换 `deployment.yaml`、`service.yaml`、`values.yaml` 等文件。
- 服务定义保存使用 `# AxiOps-File: <path>` 标记和 YAML 文档分隔符串行化，重新打开 Revision 时可以恢复多文件内容；不新增数据库字段，兼容现有 `definition` 存储和 Kubernetes 多文档解析。
- 工作台右侧增加检测到的服务组件、系统内置变量、自定义变量和变量校验入口。
- 本轮无 SQL；后端无需重启。前端开发服务需要刷新/重启，生产环境需要重新构建并重启 Web 服务。
- 验证通过：目标 CI/CD 文件 ESLint、定向 `vue-tsc` 无新增错误；主布局改动后的 Vite 生产构建通过（6262 modules），最后的模板初始化微调通过 ESLint 和定向 `vue-tsc`。

## 2026-09-09 服务目录加号交互修正

- 左侧“服务目录”右侧 `+` 改为内联创建目录，不再打开服务编辑抽屉或项目初始化页面。
- 创建目录只需填写服务名称，系统自动生成符合规范的服务编码并调用现有服务创建接口；创建后自动选中目录。
- 模板导入、Git 导入和服务编辑入口保持独立，不受目录 `+` 行为影响。
- 本轮无 SQL；后端无需重启，前端刷新或重启开发服务即可看到变更。
- 验证通过：目标 CI/CD 文件 ESLint、定向 `vue-tsc`。

## 2026-09-09 服务配置页面按截图收敛

- 服务目录 `+` 与编辑器手动写入 `+` 明确分离：前者只创建目录，后者才进入当前 YAML 编辑。
- 编辑器顶部增加手动写入、代码库导入、模板导入三个图标入口，符合服务配置页面的导入职责。
- 工作台变量区补充默认变量 `cmd/cpuLimit/memoryLimit/port/replicas`，并自动从 YAML 的 `{{.xxx}}` 占位符发现新变量。
- 变量校验检查 YAML 占位符是否已定义；保存 Revision 时同时写入变量 JSON 元数据。
- 默认 K8s YAML 使用真实换行，修复编辑器整行显示和解析错误。
- 本轮无 SQL；后端无需重启，前端刷新或重启开发服务即可。
- 验证通过：CI/CD 相关 ESLint、定向 `vue-tsc`。

## 2026-09-09 YAML 与变量联动主路径修正

- 服务工作台默认主视图固定为 YAML/Helm 编辑器 + 变量面板，Revision Diff 和服务概览降为次级 Tab，不再遮挡编辑区。
- 左侧服务列表移除“全部/已配置/待定义”筛选块，仅保留搜索、服务列表和目录 `+`。
- 编辑器顶部聚合三个配置入口：手动写入、代码库导入、模板导入；目录 `+` 仍只创建服务目录。
- 待定义服务自动加载可编辑的初始 YAML，不再显示白底空态；空态改为深色主题。
- 右侧组件区增加“添加构建”，系统变量补充 `$EnvName$`、`$ClusterName$`，自定义变量支持增删、YAML 占位符自动发现与校验。
- 本轮无 SQL；后端无需重启，前端刷新或重启 Web 开发服务即可。
- 验证通过：CI/CD 相关 ESLint、定向 `vue-tsc`。

## 2026-09-09 变量面板与导入入口再优化

- 服务工作台默认只展示“资源定义 + 变量面板”两栏，Revision/Diff/服务概览保留为次级入口，不再占据默认主区域。
- 服务列表移除冗余状态筛选块，保留搜索、列表和目录创建。
- 编辑器顶部聚合手动写入、代码库导入、模板导入三个入口；目录 `+` 仍只创建服务目录。
- 待定义服务自动提供可编辑初始 YAML，避免白底空态遮挡编辑区；空态改为深色主题。
- 变量面板支持系统变量、默认自定义变量、自动发现 YAML 占位符、变量校验和“添加构建”。
- 增加“预览变量”模式：修改右侧变量后，左侧 YAML 可即时查看渲染结果，同时保留模板原文用于保存。
- 本轮无 SQL；后端无需重启，前端刷新或重启 Web 开发服务即可。
- 验证通过：CI/CD 相关 ESLint、定向 `vue-tsc`。

## 2026-09-09 服务变量持久化与运行时变量契约

- 创建服务时将 `cmd/cpuLimit/memoryLimit/port/replicas/domain` 变量 JSON 写入 Revision 定义元数据，重新打开服务时自动恢复。
- 工作台系统变量改为真实项目名、服务编码和动态镜像变量 `$<service>-image$`；`Namespace/EnvName/ClusterName` 保持运行环境注入语义。
- 增加 `domain` 自定义变量，为后续 Ingress/二级域名模板预留稳定替换键。
- 变量区使用白色配置背景，组件与校验使用绿色强调，删除操作保留红色，编辑器继续使用深色代码区提升可读性。
- 本轮无 SQL；后端无需重启，前端刷新或重启 Web 开发服务即可。
- 验证通过：CI/CD 相关 ESLint。

## 2026-09-09 服务文件树位置与编辑态修正

- 服务 YAML 文件节点从编辑器内部移到左侧服务目录下，选中服务后展开 `deployment.yaml`、`service.yaml` 或 Helm 文件。
- 中央编辑器只负责当前选中文件，不再重复显示一套文件树。
- YAML 编辑器改为浅色工作区，工具栏、标题栏和配置区保持白底/浅灰分隔，避免整个页面黑底影响阅读。
- 有服务编辑权限时默认进入可编辑状态；无权限时仍保持只读并遵守后端授权，不绕过 RBAC。
- 待定义服务也会直接生成可编辑初始 YAML，避免出现无法编辑的锁定空态。
- 本轮无 SQL；后端无需重启，前端刷新或重启 Web 开发服务即可。
- 验证通过：CI/CD 相关 ESLint、定向 `vue-tsc`。
# 2026-09-09 YAML 区域重建

- 项目工作区已恢复为单一 YAML 编辑区域：左侧服务目录文件选择，中央编辑，右侧变量面板。
- 移除重复文件树和乱码导致的无效模板结构，浅色背景与稳定宽度已落实。
- 验证：定向 ESLint、Vue 类型检查通过；本次无 SQL，开发服务不需重启。

## 2026-09-09 Project workspace resource loading fix

- Fixed the workspace run-history request to use the project-level build list;
  it no longer sends a project ID to the workflow-ID endpoint.
- Optional services, environments, clusters, workflows and build lists now
  fail silently and render as normal empty lists instead of showing 404 toasts.
- Project detail loading is silent and falls back to the authorized project list
  for deployments where the detail route is not available yet.
- Removed the large no-service placeholder from the YAML workbench and kept the
  project failure state compact. No SQL changes; backend restart is not needed.
- Validation: targeted ESLint and vue-tsc checks pass. Vite dev server starts at
  http://127.0.0.1:3006 when started outside the Windows application policy sandbox.

## 2026-09-09 Clean YAML text editor

- Added a `showTitleBar` option to the shared YAML editor.
- The project service workbench now uses a plain white text area without the
  editor title bar, toolbar, status bar, line-number gutter or highlight layer.
- Business actions remain outside the text area; selecting a YAML file in the
  left service tree is the only navigation needed for editing.
- No SQL or backend changes. Frontend refresh/hot reload is sufficient.
- Targeted ESLint and vue-tsc checks pass.

## 2026-09-10 Service editor header cleanup

- Removed the duplicate service header from the editing pane, including the
  repeated `pic03_server` name and service metadata.
- Added clear `手动`, `模板` and `仓库` actions beside the 服务目录 heading;
  the directory `+` behavior remains a manual service-directory creation path.
- The text editor remains a clean white surface with no embedded import/export
  controls. No SQL changes and no backend restart are required.
- Targeted Prettier, ESLint and vue-tsc checks pass.
- `npm.cmd run build:dev` 构建成功。

## 2026-09-10 Collapsible service tree

- Service directories in the left catalog now behave as a tree: clicking a
  directory expands or collapses its YAML files, and clicking again restores it.
- The selected service remains highlighted while collapsing; the active file is
  preserved when the same directory is toggled.
- The first available service is selected and expanded automatically on load.
- No SQL or backend changes. Refresh/restart the frontend only.
- Targeted Prettier, ESLint and vue-tsc checks pass.

## 2026-09-10 Service entry flow

- The service catalog actions now open the real manual, template and repository
  flows instead of showing informational placeholder messages.
- Manual and template actions use the service editor wizard; repository action
  uses the bulk directory import flow so one repository can create multiple
  service directories.
- Removed the old simplified service-directory dialog from the project
  workspace. Project onboarding remains a first-time project flow; later edits
  happen inside the project tabs.
- No SQL changes or backend restart required. Refresh/restart the frontend only.
- Targeted Prettier, ESLint and vue-tsc checks pass.

## 2026-09-10 Global white canvas and editor resizing

- Light-theme global canvas and page containers now use white backgrounds,
  including cluster list and cluster management pages.
- Added a bounded mouse-drag divider between the service YAML editor and the
  variables panel. The editor can be resized from 42% to 78% of the workbench.
- Mobile layout falls back to a stacked editor/variables view.
- No SQL or backend restart required. Refresh or rebuild the frontend.
- Targeted ESLint and vue-tsc checks pass.
## 2026-09-10 Global white theme and brand refresh

- Fixed light-theme initialization and prevented OS dark-mode preference from
  reintroducing dark surfaces.
- Unified shell, navigation, project selector and CI/CD variable panel colors:
  white backgrounds with dark readable text.
- Refreshed the AxiOps logo and project icons using a teal/blue professional
  visual system.
- No SQL or backend restart required. Targeted ESLint and Prettier passed;
  full type/build checks are limited by existing repository errors and Windows
  `esbuild` spawn policy.
## 2026-09-10 Project workspace entry alignment

- New projects now continue in the same service-configuration workspace used by
  configured projects such as `fghfgh`, instead of landing on an empty overview.
- Workspace tab state is persisted in the URL and project-ID changes fully
  reload project-scoped data.
- No SQL or backend restart required. Targeted formatting, lint and Vue type
  checks passed; the local frontend is running on port 3006.
# 2026-09-10 项目初始化向导主链路

- 新增正式迁移 `migrations/029_cicd_project_initialization.sql`，为项目增加
  `initialization_status`、`initialization_step`、`initialized_at`，并将已有项目标记为已完成。
- Go 项目模型、查询、创建和更新接口已支持初始化状态；新项目默认 `pending/0`，
  完成向导时写入 `completed/4`。
- 新增正式页面 `axiops-web/src/views/cicd/project-wizard/index.vue`，实现项目配置、
  服务、环境、工作流四步初始化流程。
- 新项目创建后进入 `/cicd/projects/:projectId/wizard`；未完成初始化的项目直接访问工作区
  会自动回到向导；已完成项目继续进入日常工作区。
- 服务创建向导支持在初始化阶段复用，但关闭自动生成工作流，工作流统一在第 4 步按环境生成。

验证：

- `go test ./internal/platform/project ./migrations`
- `go test ./internal/cicd/...`
- 前端定向 `prettier`、`eslint` 通过。
- 全量 `vue-tsc` 仍受历史示例 `src/router/modules/route-config-example.ts` 的语法错误阻断。
