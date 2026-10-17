# Zadig Portal 前端源码分析与 AxiOps 重构计划

## 结论

`reference/zadig-portal` 就是 Zadig 官方 portal 前端仓库（`koderover/zadig-portal` 的 Vue 2 版本），不是截图或静态模板。它包含项目、服务、环境、构建、测试、工作流、Helm Chart 和主机应用页面，可以作为 AxiOps 的交互与信息架构参考。

它不能直接复制到 AxiOps：该仓库使用 Vue 2.6、Vuex 3、Element UI 2、Webpack 3，接口全部指向 Zadig 的 `/api/aslan/*`；AxiOps 当前使用 Vue 3、Pinia、Element Plus、Vite 和自有 `/platform/v1`、`/cicd/v1` API。因此采用“按页面和交互重建、按接口和权限重新实现”，不做整仓替换。

## 已确认的 Zadig 设计资产

| 能力 | 参考源码 | AxiOps 落地方式 |
| --- | --- | --- |
| 项目创建向导 | `src/components/projects/guide/` | 保留三类项目向导：K8s YAML、Helm、主机；改为 Vue 3 分步组件 |
| 服务树与多栏编辑 | `guide/service.vue`、`service_mgr/common/service_tree.vue` | 服务目录 + 定义编辑器 + 构建/变量侧栏；适配 AxiOps Revision 模型 |
| YAML 编辑与校验 | `guide/k8s/container/service_editor.vue` | 使用现有 `YamlEditorPro`，增加资源导航、格式化、只读权限和 Diff |
| 构建/变量侧栏 | `service_mgr/k8s/service_aside.vue` | 统一镜像仓库、构建配置、系统变量、项目变量和环境覆盖 |
| 环境详情 | `env/outer_env/`、`env/inner_env/` | 环境矩阵、集群/Namespace 目标、渲染预览、发布和回滚 |
| 工作流列表与运行 | `pipeline/workflow_list.vue`、`pipeline/workflow_multi_task_detail/` | DAG 编辑、阶段状态、节点日志、测试门禁和审批抽屉 |
| Helm 文件树 | `service_mgr/helm/components/common/`、`service_mgr/helm/code.vue` | Chart 文件树、values 分层覆盖、渲染 Diff 和 Release 历史 |
| 主机服务 | `guide/not_k8s/`、`guide/host/` | 主机应用构建产物、Agent 部署、探活和回滚 |
| 资源与视觉规范 | `src/assets/css/component/*.less`、`src/assets/icons/step/` | 只提取布局/状态语义；AxiOps 统一使用现有设计令牌和 UTF-8 文案 |

## 与当前截图问题的关系

截图不是 Zadig 的原始布局，而是 AxiOps 当前工作台在加载状态和窄屏下的异常表现：全页 `v-loading` 遮罩使内容低对比度，三栏固定最小宽度又造成右侧裁切。第一批修复应包括：把全页加载改为内容骨架/局部加载、给三栏设置可收缩断点和横向滚动、提高正文与次要文字对比度、保证编辑器和操作栏不被遮挡。

## AxiOps 分阶段实施顺序

### P0：先恢复可用性和视觉基线

1. 修复项目工作台加载遮罩、布局溢出、浅色文字和移动/窄窗口断点。
2. 建立统一页面壳：项目标题、项目级导航、面包屑、操作区和状态反馈。
3. 以真实数据和空状态分别验收，不允许 API 失败时显示“系统内部错误”或空白页面。

### P1：服务管理按 Zadig 方式重构

1. 新建服务三步向导：选择方式（手工、代码库、模板）、填写服务元数据、确认定义与构建配置。
2. 服务目录支持搜索、类型/状态筛选、批量同步和服务克隆。
3. 服务详情采用三栏：Revision 历史、YAML/Helm/主机定义、环境引用与构建配置。
4. 编辑始终生成新 Revision；环境参数只生成环境覆盖和 Diff，不修改服务基线。

### P1：环境与发布矩阵

1. 项目可以只有测试+生产，也可以配置开发、测试、预发布、生产，不强制补齐环境。
2. 每个服务在每个目标环境显示一个可执行发布单元；工作流是复用模板，运行时按服务×环境生成 Job，不要求为每个环境复制工作流定义。
3. 支持滚动、金丝雀、蓝绿策略，生产强制审批，同一构建产物按晋升链复用。

### P2：工作流、测试和可观测性

1. 工作流节点统一为源码、构建、测试、部署、审批、通知，可并行并限制并发。
2. 运行详情对齐 Zadig：阶段时间线、节点日志、Commit 链接、镜像 digest、测试报告、失败根因和重试/回滚。
3. 日志与监控独立模块通过项目、服务、环境和发布运行 ID 关联。

### P3：大规模与企业能力

1. 服务目录采用分页/虚拟列表，支持 2000+ 服务和权限过滤。
2. 所有按钮、菜单、项目/服务/环境/集群查询执行同一权限范围校验。
3. 增加变更日历、审计、通知渠道、成本和合规检查。

## 页面验收标准

- 1440px、1280px、1024px 宽度下不出现右侧内容被裁切；窄屏可横向滚动或自动降为双栏/单栏。
- 加载时只遮罩正在请求的区域，标题、导航和已有数据保持可读。
- 空数据、无权限、请求失败、停用服务均有明确中文提示和下一步操作。
- 开发者只能看到授权项目/服务/集群，并且只读权限不会出现编辑、构建或发布按钮。
- YAML/Helm 编辑、Revision、环境 Diff、工作流运行和日志均可从页面闭环操作。

## 本阶段决策

- 不直接引入 Zadig Portal 旧依赖，避免 Vue 2/3 和接口模型冲突。
- 优先重构 AxiOps 项目工作台与服务向导，再扩展 Helm 和工作流高级能力。
- 每完成一个页面模块，必须补充 `docs/02-cicd` 工作记录、定向 lint/test 和截图验收。
