# AxiOps CI 设计与参数规范

更新时间：2026-09-06

## 1. 设计结论

CI 负责把一次代码变更转换为可追踪的构建产物：源码 Commit -> 构建 -> 测试 -> 扫描 -> 镜像/制品。CD 负责把已经固定 digest/checksum 的产物部署到环境目标。二者共用流水线 DAG、变量、凭据和运行记录，但不共用执行语义。

```text
项目
  ├─ 服务基线（代码源、语言、构建配置、默认镜像仓库）
  ├─ 环境（按项目启用：测试/生产、开发/测试/预发布/生产或自定义组合）
  │   └─ 发布目标（cluster + namespace，可多个）
  └─ 流水线模板/工作流
      ├─ CI 节点：checkout -> build -> test -> scan -> publish
      └─ CD 节点：approval -> yaml/helm/host deploy -> verify -> notify
```

同一项目可有多个服务、多个环境和多个集群。服务只定义一次；环境保存服务的目标 Revision、镜像 digest、变量和发布策略。不要为每个环境复制一套服务配置。

环境阶段不是平台级固定字典。项目可以只配置 `test` 和 `prod`，形成最小生产链：

```text
build once -> test deploy/verify -> production approval -> prod deploy
```

不存在的 `dev`、`stage` 不会被隐式创建，也不会出现在该项目的环境列表、工作流目标下拉框或晋级门禁中。

## 2. 项目、服务、环境、Job 的关系

| 对象 | 作用 | 是否长期保存 |
| --- | --- | --- |
| 项目 | 团队、权限、默认仓库、服务和环境的隔离边界 | 是 |
| 服务 | 一个可构建/可部署单元；包含代码、构建、YAML/Chart 和默认镜像 | 是 |
| 环境 | dev/test/stage/prod 等逻辑发布面 | 是，可按项目选择数量 |
| 发布目标 | 环境下的具体 cluster + namespace 或主机组 | 是，可一对多 |
| 流水线模板 | 可复用的节点、参数和变量映射 | 是，版本化 |
| 工作流 | 某项目基于模板实例化后的可编辑 DAG | 是，版本化 |
| Job/Task | 一次运行中的具体构建、测试或部署执行单元 | 运行时创建，有历史记录 |

推荐关系：

```text
Project 1---N Service
Project 1---N Environment 1---N DeploymentTarget
Service 1---N EnvironmentServiceOverride
Project 1---N Workflow 1---N WorkflowRun 1---N TaskRun
WorkflowRun -> BuildTask(service) -> Artifact(digest)
WorkflowRun -> DeployTask(environment, target, service, artifact)
```

### 为什么不是每个环境永久配置多个构建 Job

构建结果应该由 `service + source_commit + build_config_revision + variables` 唯一确定。同一个 Commit 不需要在开发、测试、预发布、生产重复构建；一次构建产生镜像 digest，后续环境只提升同一个 digest。环境差异属于部署覆盖和验证策略，不属于 CI 构建任务。

运行时的 Job 数量按本次选择动态生成：

```text
选择 3 个服务
  -> 3 个并行 build job
  -> 3 个并行 test job
  -> 测试环境 1 个目标 = 3 个 deploy job
  -> 生产环境 2 个集群 = 6 个 deploy target job
```

页面显示的是一次运行的 Job/Task 树，不把这些临时任务误显示为项目永久配置。

## 3. CI 流水线标准步骤

### 3.1 通用步骤

所有项目类型都可以使用以下通用 CI 步骤，步骤是否启用由工作流模板决定：

1. `checkout`：按代码源、仓库、Branch/Tag/Commit/PR 固定源码。
2. `prepare`：选择构建节点池、构建镜像、缓存、变量和凭据引用。
3. `build`：按语言或自定义脚本编译、打包、构建容器镜像。
4. `test`：单元测试、接口测试、E2E、覆盖率和测试报告。
5. `scan`：依赖、镜像、SAST、Secret 和许可证扫描。
6. `publish`：推送镜像或二进制制品，记录 digest/checksum。
7. `report/notify`：写入运行结果并经 outbox 发送通知。

### 3.2 项目类型差异

| 项目类型 | CI 重点 | CD 节点 |
| --- | --- | --- |
| K8s YAML | Dockerfile/脚本构建镜像，解析 YAML 中的镜像替换点 | `yaml_deploy`，按资源 apply 和健康检查 |
| Helm Chart | Chart 本身不是必须构建对象；可构建镜像，也可复用已有镜像；执行 lint、template 和 values 校验 | `helm_deploy`，按 release upgrade/install、wait、rollback |
| 主机应用 | 编译二进制/压缩包，生成制品 checksum | `host_deploy`，Agent 滚动分批、探活、回滚 |

YAML 和 Helm 可以共享 checkout、prepare、build、test、scan、publish、notify 节点；部署节点不能混用。Helm 需要额外的 `chart_lint`、`helm_template` 和 values 校验，YAML 需要 manifest schema、资源白名单和镜像替换点校验。页面应复用相同的步骤编辑器，但根据项目类型只展示可执行的节点。

## 4. 通用变量与映射

### 4.1 变量作用域

变量采用分层模型，所有值在运行快照生成时解析，不把 Secret 明文写入工作流、Job、日志或 Diff：

```text
平台变量
  < 项目变量
    < 服务变量
      < 环境变量
        < 环境服务覆盖
          < 流水线运行参数
```

同名变量由更窄作用域覆盖更宽作用域。建议每个变量带有 `type`、`description`、`default`、`required`、`secret`、`allowed_values` 和 `source`。

支持的变量类型：

| 类型 | 示例 | 说明 |
| --- | --- | --- |
| `string` | `GO_VERSION=1.24` | 普通文本 |
| `choice` | `DEPLOY_ENV=test` | 从允许值选择 |
| `multi_select` | `SERVICES=api,web` | 批量服务选择 |
| `image` | `IMAGE_TAG=20260906.2` | 镜像仓库/标签/digest 结构化值 |
| `file` | `KUBECONFIG_REF` | 只保存凭据引用或临时文件引用 |
| `secret` | `REGISTRY_TOKEN` | 只引用 credential_id，运行时注入 |
| `json` | `HELM_VALUES` | 受 schema 限制的结构化覆盖 |

### 4.2 映射语法

模板节点使用稳定的变量引用，不直接写死项目或服务名称：

```text
{{ project.code }}
{{ service.code }}
{{ service.repo.branch }}
{{ run.commit }}
{{ run.build_number }}
{{ image.repository }}
{{ image.tag }}
{{ image.digest }}
{{ environment.code }}
{{ target.cluster }}
{{ target.namespace }}
{{ variables.GO_VERSION }}
{{ secrets.REGISTRY_TOKEN }}
```

映射必须在保存工作流时进行静态检查：变量是否存在、类型是否兼容、Secret 是否只能进入允许的输入、模板表达式是否越权读取其他项目资源。运行时再按当前用户和目标范围复核。

### 4.3 常用自动变量

系统自动生成且只读：`AXIOPS_PROJECT_CODE`、`AXIOPS_SERVICE_CODE`、`AXIOPS_WORKFLOW_ID`、`AXIOPS_RUN_ID`、`AXIOPS_COMMIT_SHA`、`AXIOPS_BRANCH`、`AXIOPS_BUILD_NUMBER`、`AXIOPS_IMAGE_DIGEST`、`AXIOPS_ENVIRONMENT_CODE`、`AXIOPS_CLUSTER_NAME`、`AXIOPS_NAMESPACE`。

用户可以覆盖显示标签和非敏感默认值，但不能覆盖 Run ID、Commit SHA、digest、权限范围和审计字段。

## 5. 多节点和并行构建

流水线定义采用 DAG，不使用固定的线性脚本：

```text
checkout
  ├─ build-backend-a ─┐
  ├─ build-backend-b ─┼─ test-backend ─┐
  ├─ build-frontend-a ┘                ├─ publish
  └─ security-scan --------------------┘
                                      -> deploy
```

### 并行规则

- 同一服务的 checkout、build、test 默认串行；不同服务可并行。
- 同一服务的多个组件可以通过 matrix 展开，例如 `linux/amd64`、`linux/arm64`。
- `max_parallel` 由平台、项目和工作流三级并发策略共同限制。
- 失败策略可选 `fail_fast`、`continue_on_error` 和 `allow_failure`；生产发布前的构建、测试和安全扫描默认不允许绕过。
- 一个 Job 只能绑定一个执行节点/Agent；多个节点由 Worker 调度到对应节点池，不在一个容器中伪装多节点。
- 任务名称在同一个工作流定义中必须唯一；矩阵展开后的运行任务使用 `task_name[index]`，原始模板名称仍唯一。

### 构建节点配置

```yaml
key: build-payment-api
name: 构建 payment-api
type: build
service_id: 42
runner_pool: k8s-build-amd64
build_image: registry.example.com/axiops/go-builder:1.24
command: make ci
working_dir: .
dockerfile: Dockerfile
context: .
cache: go-mod
resources: { cpu: "2", memory: 4Gi }
timeout_seconds: 1800
retry: { max_attempts: 1, backoff_seconds: 30 }
outputs: [image, junit, coverage]
```

服务的语言、框架和 Dockerfile 可以由导入向导识别并给默认值；用户只需要在构建配置中确认或修改。构建模板应支持 Go、Maven/Gradle、Node、Python、Docker 和自定义 Shell，但 Shell 仍受执行器白名单、凭据和资源限制约束。

## 6. 流水线模板

模板是版本化的声明式蓝图，不是点击后创建一堆永久 Job。模板实例化时生成项目工作流、节点和参数绑定：

```text
模板：标准 K8s 服务 CI/CD
  -> 项目：支付平台
  -> 选择服务：payment-api/order-api/portal-web
  -> 选择环境：项目已配置且当前用户有权限的环境（例如 test 或 prod）
  -> 生成工作流 v1
  -> 运行时展开 build/test/deploy Job
```

模板应包含：

- 支持的项目类型和服务类型。
- 节点 DAG、默认依赖和并发策略。
- 节点输入 schema、默认值、必填项和权限要求。
- 变量定义与映射表达式。
- 构建/测试/扫描 provider 和 runner pool。
- 产物规则、镜像仓库选择和通知策略。
- 是否允许环境发布、是否要求审批、超时和重试。

模板选中后的用户体验：

1. 选择模板版本。
2. 选择一个或多个服务。
3. 选择需要启用的环境。
4. 系统自动填充语言、构建镜像、镜像仓库、命名空间和默认节点。
5. 用户只修改项目特有的变量、服务目录和策略。
6. 预览将生成的工作流和 Job 展开数量，确认后保存。

模板更新不自动修改已经运行的工作流；用户显式“升级到模板版本”后生成新的工作流版本，历史运行继续使用旧快照。

## 7. 镜像仓库设计

镜像仓库是平台级资源，可授权给项目和服务选择。服务只保存 `registry_id`、镜像仓库路径和命名规则，不保存 Token 明文：

```text
registry: registry.axiops.local
project path: payment-platform
service: order-api
default image: registry.axiops.local/payment-platform/order-api
tag: {{ run.build_number }}
deploy image: registry.axiops.local/payment-platform/order-api@sha256:...
```

建议支持 Harbor、Docker Registry、阿里云 ACR、腾讯云 TCR、AWS ECR 和通用 OCI Registry。推送使用项目授权凭据；部署读取固定 digest，不接受工作流运行时任意改写。

## 8. 部署任务参数规范

下面这些字段属于 CD 的部署节点，不属于 CI 构建节点。任务名称在同一个工作流中必须唯一：

| 参数 | 必填 | 说明 |
| --- | --- | --- |
| 任务名称 | 是 | 工作流内唯一，例如 `deploy-order-test` |
| 服务 | 是 | 项目内服务；运行时可从服务选择器批量展开 |
| 镜像仓库 | YAML/镜像部署必填 | 从哪个已授权 OCI 仓库获取镜像；建议保存 registry_id，不保存 Token |
| 镜像版本 | 通常自动 | 默认使用上游 build 输出的 digest；手工指定必须有 `artifact:use` 权限 |
| 集群 | 是 | 已连接且用户有范围权限的集群；可选择一个或多个目标 |
| 命名空间 | 是 | 集群中的 Namespace；必须属于环境或通过 Namespace 策略校验 |
| 容器/工作负载 | 是 | 当前支持 Deployment 和 StatefulSet；YAML 可由资源解析自动列出 |
| 容器状态检测 | 否 | 开启后轮询 Deployment/StatefulSet rollout 和 Pod Ready 状态 |
| 超时时间 | 是 | 建议 60-3600 秒；超时即失败，不把未确认状态标记成功 |
| 发布策略 | 是/默认 | 滚动、金丝雀或蓝绿；默认按环境继承 |
| 副本与策略参数 | 否 | maxSurge、maxUnavailable、阶段比例、观测窗口等 |

多个集群目标不应合并为一个不可区分的 Job。系统显示一个部署批次节点，下挂每个 `cluster + namespace + workload` 的目标任务，每个目标有独立日志、状态、探活和回滚点。

## 9. YAML 与 Helm 的流水线差异

### 共用

- 项目、服务、环境、工作流、变量、凭据、构建、测试、扫描、产物和通知模型。
- 服务选择、镜像仓库选择、Commit/Revision、权限、审批、超时和重试。
- 运行详情的 DAG、日志、测试报告、Diff、审计和 AI 根因分析入口。

### 不共用

| 领域 | YAML | Helm |
| --- | --- | --- |
| 服务定义 | 多文档 manifest、Kustomize/目录文件 | Chart.yaml、values、templates |
| 预览 | manifest schema、资源级 Diff | `helm template`、values 合并和 release Diff |
| 部署 | Server-Side Apply | `upgrade --install --atomic --wait` |
| 回滚 | 上一成功 rendered manifest | Helm release revision |
| 镜像替换 | 解析容器 image 字段 | values 键或模板渲染后的 image 字段 |
| 容器状态 | Deployment/StatefulSet rollout | Release 资源和 wait 状态 |

所以答案是：流水线骨架相同，部署步骤和配置面板不同。前端可以用同一 DAG 编辑器和节点抽屉，但 `yaml_deploy` 与 `helm_deploy` 必须是两个强类型节点，不能让 YAML 节点填写 Helm release，也不能让 Helm 节点直接提交任意 manifest。

## 10. 一次构建，多环境晋级

### 10.1 基本原则

同一个 `service + source_commit + build_config_revision + build_variables` 只允许产生一个构建结果。构建成功后登记不可变镜像 digest 或制品 checksum，后续环境只引用这个产物：

```text
代码 Commit 3e1c6bb
       -> CI 构建一次
       -> registry/payment/order-api@sha256:abc...
       -> 开发验证
       -> 测试验证
       -> 预发布验证
       -> 生产审批后晋级
```

禁止以下行为：

- 为了进入测试、预发布或生产重新执行 Docker build。
- 让环境使用可变 `latest`、时间标签但不记录 digest，或运行时任意修改镜像地址。
- 生产环境直接选择未在上游环境验证的镜像。
- 用重新构建替代失败发布后的回滚。

允许的例外必须显式标记为新的制品版本，例如生产使用不同架构构建、不同合规编译参数或紧急修复；例外会生成新的 source/build provenance，并不能伪装成同一版本晋级。

### 10.2 制品晋级模型

制品状态建议采用：

```text
published -> dev_verified -> test_verified -> stage_verified -> prod_approved -> deployed
                                      \-> rejected / quarantined
```

每次晋级保存一条不可变记录：`artifact_id`、镜像 digest/checksum、来源 Commit、构建配置版本、测试结果摘要、扫描结果摘要、来源环境、目标环境、操作者、审批人、时间和请求键。晋级只改变制品的环境标签或引用关系，不复制镜像层。

镜像仓库建议使用不可变 digest 作为真实身份，环境标签仅用于检索：

```text
registry/payment/order-api@sha256:abc...     # 部署真实引用
registry/payment/order-api:test               # 可选检索标签
registry/payment/order-api:stage              # 可选检索标签
```

标签被覆盖或删除不会改变已经保存的 digest 和运行快照。

### 10.3 环境晋级门禁

环境不是简单的下拉选项，每个环境定义自己的发布门禁：

| 环境 | 默认门禁 | 默认执行方式 |
| --- | --- | --- |
| 开发 | 构建成功、基础测试通过 | 自动部署，可按服务单独运行 |
| 测试 | 构建成功、单元/接口测试、镜像和依赖扫描 | 自动部署，失败停止当前服务批次 |
| 预发布 | 测试环境验证记录、回归测试、变更摘要 | 自动或人工确认后部署 |
| 生产 | 预发布验证、风险检查、发布窗口、审批 | 审批后按目标集群批次部署 |

门禁结果必须关联同一个 artifact digest；只要 digest 变化，必须重新经过门禁。只修改环境副本数、资源限制或 Namespace 的配置，也要重新生成 Diff 和环境变更记录，但不重新构建镜像。

### 10.4 发布策略标准化

发布策略属于“环境服务覆盖”，由项目模板提供默认值，服务可以在授权范围内覆盖。三种标准策略统一包含准备、执行、验证、推进和失败处理阶段：

#### 滚动发布

```text
准备 Diff -> 更新工作负载 -> 等待 Ready/rollout -> 健康检查
         -> 成功完成 / 超时停止并保留旧版本
```

参数：`max_surge`、`max_unavailable`、探活超时、整体超时、失败后是否暂停。适用于大多数开发、测试和基础生产服务。

#### 金丝雀发布

```text
0% -> 5% -> 25% -> 50% -> 100%
       每阶段观测错误率、延迟、可用性和业务指标
```

参数：阶段比例、观测窗口、Prometheus 查询、自动推进阈值、人工确认点和失败回滚。没有流量路由或服务网格能力时，前端不展示该策略。

#### 蓝绿发布

```text
创建 Preview -> 部署并验证 -> 切换 Active -> 保留旧版本 -> 完成/回切
```

参数：Preview/Active 标识、切换方式、验证窗口、旧版本保留时间和回切条件。切换和回切都必须生成独立审计事件。

### 10.5 多集群发布批次

一个逻辑环境可以有多个目标集群。例如生产环境包含 `prod-sh` 和 `prod-bj`：

```text
生产发布批次（artifact sha256:abc）
  ├─ prod-sh / payment-prod / order-api  -> deploy target job
  └─ prod-bj / payment-prod / order-api  -> deploy target job
```

批次策略可选：

- `parallel`：多个集群并行发布，适合非核心服务。
- `ordered`：按集群顺序发布，前一集群健康后才进入下一集群。
- `canary_targets`：先发布一个集群作为集群级金丝雀，再扩散到其他集群。

每个目标任务单独保存状态、日志、Diff、健康检查和回滚点；批次汇总状态不能覆盖目标明细。任一目标失败时，根据策略停止后续目标、暂停等待人工处理或自动回滚已经成功的目标。

### 10.6 标准发布流程

推荐所有项目从同一发布模板开始：

```text
选择服务/版本
  -> 校验 artifact digest、Commit、扫描和上游环境门禁
  -> 解析目标环境和集群
  -> 生成 YAML/Helm/主机 Diff
  -> 创建发布批次
  -> 生产审批（如需要）
  -> 按目标和策略部署
  -> 等待容器状态/探活/指标门禁
  -> 成功登记晋级；失败暂停或回滚
  -> 通知、审计和发布报告
```

发布页面必须显示“构建一次、晋级多次”的关系：当前 artifact、来源 Commit、已通过环境、待审批环境、目标集群、策略阶段、每个目标的 Job 和回滚版本。

### 10.7 回滚标准

回滚不是重新构建。系统从最近一次成功部署的不可变快照恢复：

- YAML：使用上一成功 rendered manifest 的 digest。
- Helm：使用历史 release revision 和 values 快照。
- 主机：使用上一成功制品 checksum 和脚本 Revision。

生产回滚复用审批和 `deployment:rollback` 权限；回滚目标仍按集群、Namespace、服务范围校验。回滚结果要生成新的运行记录，不能修改原发布记录。

## 11. 前端发布页建议

发布页建议分成四个区域：

1. **版本选择**：服务、Commit、构建编号、镜像 digest、测试/扫描状态和可晋级环境。
2. **目标矩阵**：开发、测试、预发布、生产及其集群、Namespace、当前版本和目标版本。
3. **策略与门禁**：滚动/金丝雀/蓝绿参数、容器状态检测、超时时间、审批人和发布窗口。
4. **执行与结果**：批次进度、每目标 Job、实时日志、健康检查、Diff、失败原因和回滚。

用户选择多个服务时，页面先显示预计的构建、测试、部署和目标任务数量；构建阶段按服务并行，部署阶段按环境目标扇出。用户不能在生产目标上直接编辑镜像字符串，只能从已经通过门禁的 artifact 列表选择。

## 12. 前端页面建议

### 创建流水线

```text
选择模板 -> 选择服务（可多选） -> 选择环境/目标集群
          -> 确认变量映射 -> 预览 DAG 和 Job 数量 -> 保存工作流
```

### 构建配置页

左侧是服务列表，中间是构建/测试/扫描步骤，右侧是变量映射、缓存、资源和产物。支持“从模板重置”和“仅保存当前服务”。

### 运行页

顶部显示项目、服务、Commit、模板版本和目标环境；中部显示 DAG；底部显示 Job 列表。Job 列表按服务、阶段和目标集群分组，支持查看日志、重试、取消和产物。

### 2000+ 服务体验

- 所有服务列表服务端分页，不一次加载完整树。
- 预设视图：我的服务、待发布、构建失败、测试失败、环境偏离、生产审批中。
- 批量选择显示“选中 N 个服务，将生成 M 个构建/测试/部署 Job”。
- 权限过滤在服务端先执行，前端不依赖隐藏按钮保护数据。

## 13. 实施顺序

1. 固化变量模型、表达式和构建节点 schema，不改现有 YAML/Helm executor 语义。
2. 增加流水线模板和工作流实例化 API，先支持一个标准 K8s YAML 模板。
3. 增加多服务选择、DAG matrix 并行展开和 Job 数量预览。
4. 增加统一镜像仓库资源和项目/服务授权绑定。
5. 增加 test 节点、扫描节点和产物版本选择。
6. 增加环境多目标部署批次和每目标状态。
7. 最后实现 Helm/YAML 专用节点面板、金丝雀/蓝绿策略和真实集群 E2E。

本设计阶段不新增 SQL。实现阶段如需模板、变量版本、镜像仓库或多目标部署持久化字段，必须先新增编号迁移，并在模块目录记录手动执行方式。

## 14. 构建可追溯性与详细日志

### 14.1 构建必须回答的问题

任何一次构建或发布，页面都必须能回答：

```text
谁触发的？
来自哪个代码源、仓库、分支和 Commit？
Commit 改了哪些文件？
使用了什么构建模板、构建镜像、命令和变量？
在哪个 Runner/节点执行？
命中了哪些缓存？
生成了哪个镜像 digest/制品 checksum？
经过了哪些测试和扫描？
晋级到了哪些环境和集群？
```

### 14.2 Source Snapshot

构建开始时冻结 `source_snapshot`，至少记录：

| 字段 | 说明 |
| --- | --- |
| `code_source_id` | 平台代码源标识，不保存 Token |
| `provider` | Gitea/GitLab/GitHub/Gitee 等 |
| `repo_path` | owner/namespace/repository |
| `clone_url` | 脱敏后的仓库地址 |
| `branch` / `tag` | 用户选择的来源 |
| `commit_sha` | 实际构建 Commit，必须是完整 SHA |
| `commit_message` | 提交标题/摘要 |
| `author` | 提交作者 |
| `committed_at` | 提交时间 |
| `commit_url` | 代码源对应 Commit 页面链接 |
| `changed_files` | 文件路径、操作、旧路径、增删行统计 |
| `source_snapshot_hash` | 上述元数据摘要 |

`commit_url` 应由 provider 根据代码源地址和 Commit 生成，点击后直接打开 Gitea/GitLab 等仓库的具体提交页面。若代码源不支持链接，页面展示完整 SHA 和仓库地址，不拼接未经验证的 URL。

### 14.3 构建配置快照

运行时不能只显示“使用默认构建配置”。必须保存不可变 `build_config_snapshot`：

- 构建模板名称和版本。
- Runner pool、节点标签、构建镜像和镜像 digest。
- 工作目录、Dockerfile、构建上下文和实际命令。
- 解析后的非敏感环境变量名和值摘要；Secret 只显示凭据名称/ID。
- CPU、内存、磁盘、超时、重试和并发限制。
- 缓存类型、缓存键、命中/未命中、恢复大小和写入大小。
- 构建脚本、Dockerfile 和 lockfile 的 SHA-256。

### 14.4 日志结构

日志按 `run -> service -> task -> phase -> stream` 分层，阶段至少包括：

```text
queued / source / prepare / restore-cache / build / test / scan / publish / finalize
```

每条日志事件包含：`timestamp`、`sequence`、`level`、`phase`、`service_code`、`task_key`、`runner_id`、`message`。页面支持按阶段折叠、搜索、复制、下载、从失败行跳转到构建配置和 Commit 文件。

日志中禁止输出 Token、密码、Secret 值、完整 kubeconfig、Authorization Header 和临时凭据。Worker 在写入数据库、对象存储和 SSE 前都执行统一脱敏；数据库只保留有界尾部，对象存储保存完整压缩日志和校验摘要。

### 14.5 Commit 代码变更展示

运行详情增加“代码变更”标签页：

- Commit 标题、作者、时间和可点击 Commit 链接。
- 变更文件列表：新增、修改、删除、重命名。
- 增删行统计和文件类型统计。
- 可选受权限控制的 patch/diff 预览；默认只展示元数据，避免大 diff 影响页面和日志。
- 服务目录过滤：多服务构建时显示每个服务实际影响的文件和是否触发构建。

代码变更来自 provider 的 Commit API，不让前端自行拼装仓库 URL，也不把整个仓库内容复制到 AxiOps 数据库。

## 15. 构建缓存最佳方案

### 15.1 缓存分层

建议按成本和复用范围分四层：

1. **Runner 本地短缓存**：节点本地 Docker layer、Go build cache、Maven/Gradle、npm/pnpm、pip；速度快但可丢失。
2. **项目共享缓存**：以项目和语言隔离，存放依赖缓存；适合多个服务复用，不能跨租户读取。
3. **OCI Registry BuildKit cache**：构建镜像层和多架构缓存，适合弹性 Kubernetes Runner。
4. **对象存储归档缓存**：大体积依赖或冷缓存，使用 MinIO/S3/OSS/COS，设置 TTL 和配额。

### 15.2 缓存键

缓存键必须由输入内容决定，不能只使用分支名：

```text
{project_id}/{service_id}/{language}/{runtime}/{lockfile_sha256}/{build_tool_version}/{platform}
```

Docker layer cache 还要包含 Dockerfile、构建上下文相关文件摘要和基础镜像 digest。缓存命中只影响构建速度，不影响最终镜像 digest 的来源证明；缓存损坏或校验失败时自动丢弃并重新生成。

### 15.3 缓存安全和治理

- 默认按平台/项目隔离，禁止开发项目读取生产项目缓存。
- Secret、源码 Token、构建临时文件和部署凭据不能进入缓存。
- 缓存写入使用不可变 key 或版本化 key，避免并发构建相互覆盖。
- 项目、服务和平台分别配置容量、TTL、最大单文件大小和清理策略。
- 页面显示命中率、恢复/写入大小、耗时和清理结果；缓存故障不能改变构建正确性。

## 16. YAML/Helm 配置修改最佳方案

### 16.1 基线与覆盖分离

YAML/Helm 服务配置采用“基线 + 环境覆盖”，不能直接在生产环境编辑完整 YAML：

```text
服务基线 Revision r28
  ├─ Kubernetes YAML/Chart 文件
  ├─ 默认镜像、端口、资源和健康检查
  └─ 默认副本数 2

测试环境覆盖 v7
  ├─ image digest
  ├─ replicas: 3
  ├─ resources.requests
  └─ 非敏感环境变量
```

服务基线修改生成新的 Revision；环境修改只生成环境覆盖 Revision。这样同一服务可以在测试使用 3 副本、预发布使用 4 副本、生产使用 6 副本，而不复制三份 YAML。

### 16.2 三种编辑方式

服务详情提供三种同步的编辑视图：

1. **结构化表单**：资源树 -> Deployment/StatefulSet -> 容器 -> 镜像、端口、副本、资源、探活；适合修改常用参数。
2. **YAML 编辑器**：支持语法高亮、schema 校验、路径定位、搜索和多文档切换；适合高级字段。
3. **Diff 视图**：左侧基线/上一版本，右侧当前草稿/目标环境渲染结果；显示新增、删除和修改字段。

用户在结构化表单修改副本数或镜像时，系统生成受控 JSON Patch/Strategic Merge Patch，并同步更新 YAML 预览；用户直接改 YAML 时，系统重新解析并检查 patch 是否越界。两种编辑不会各自保存一份互相冲突的配置。

### 16.3 环境同步

环境更新操作分为：

- `提升版本`：把同一个服务 Revision 和 artifact digest 从开发提升到测试/预发布/生产。
- `同步基线`：把服务新的 Revision 应用为环境草稿，同时保留该环境已有的合法覆盖。
- `复制环境配置`：从测试复制非 Secret 覆盖和策略到预发布，集群、Namespace、审批和凭据必须重新确认。
- `重置覆盖`：删除某个环境字段的覆盖，使其回到服务基线默认值。

每次同步先执行 YAML/Helm 渲染、schema、安全规则和资源 Diff；确认后生成新的环境 Revision 和发布运行。同步不会自动构建镜像，除非用户明确选择新的源码 Commit。

### 16.4 常见参数的推荐归属

| 参数 | 服务基线 | 环境覆盖 | 说明 |
| --- | --- | --- | --- |
| 容器名称、端口、探活路径 | 默认 | 可覆盖 | 结构兼容时允许覆盖 |
| 镜像仓库路径 | 默认 | 通常不改 | 生产建议固定平台授权仓库 |
| 镜像 tag/digest | 不保存运行 tag | 是 | 环境只选择已晋级 digest |
| replicas | 默认 | 是 | 每环境可不同 |
| CPU/Memory | 默认 | 是 | 生产可提高资源 |
| HPA | 默认 | 是 | 依赖环境指标能力 |
| Namespace/集群 | 不保存具体目标 | 是 | 由环境目标绑定 |
| Secret | 只保存引用 | 是 | 禁止明文和跨项目复制 |
| 发布策略 | 默认 | 是 | 受环境能力和权限限制 |

## 17. 标准化验收场景

必须完成以下场景后，才能把 CI/CD 体验标记为完成：

1. 选择 3 个服务构建，页面显示 3 个并行构建任务、源码 Commit、变更文件、构建模板和缓存命中情况。
2. 构建只生成一个镜像 digest；该 digest 按项目配置的晋级链推进（例如 test -> prod，或 dev -> test -> stage -> prod），不重复构建。
3. 测试环境修改副本数和环境变量，系统只生成环境覆盖 Diff，不修改服务基线。
4. 从测试提升到预发布，页面显示相同 digest、不同集群/Namespace 和独立部署目标任务。
5. 生产有两个集群时显示一个发布批次和两个目标 Job；一个目标失败时可以暂停、查看日志并按策略回滚。
6. 点击 Commit 链接能进入代码源对应的具体提交页面；无链接时显示完整 SHA 和变更文件。
7. 缓存失效时构建仍能成功，缓存不会包含 Secret；开发账号不能读取其他项目缓存和日志。
