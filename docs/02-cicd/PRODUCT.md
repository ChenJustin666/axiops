# CI/CD 产品设计

> 状态：已确认为下一阶段 CI/CD 的产品基线
>
> 参考：用户提供的 Zadig 三类项目流程，以及仓库 `reference/zadig` 中的 Pod executor / VM agent 分离设计。

## 1. 产品定位

AxiOps CI/CD 是企业交付控制面，不是单一的构建页面。用户在项目内完成服务定义、环境管理、工作流交付、日志追踪和发布治理；实际构建、Kubernetes 操作和主机执行由受控 provider 完成。

核心路径固定为：

```text
项目 Project -> 服务 Service -> 环境 Environment -> 工作流 Workflow -> 运行记录 Run
```

项目是权限隔离与交付语义边界。服务是最小可交付单元。环境是服务的运行上下文。工作流是对服务和环境变更的可审计 DAG。

## 2. 三类项目

| 项目类型 | 标识 | 服务来源和解析 | 环境资源 | 交付 provider | 回滚基线 |
| --- | --- | --- | --- | --- | --- |
| K8s YAML | `k8s_yaml` | Git 中的 YAML/Kustomize；解析 Workload、Service、Ingress、镜像字段 | 集群 + Namespace | server-side apply 或受控 apply Job | 上一份成功渲染清单 |
| Helm Chart | `helm` | Git/Chart 仓库中的 Chart 与 values；解析 chart 元数据和组件镜像 | 集群 + Namespace + release name | Helm `upgrade --install --atomic --wait` | Helm revision |
| 主机应用 | `host` | Git 代码与构建产物；部署脚本、启停策略、健康检查 | 主机组 + Agent | 主机 Agent 拉取制品并执行受签名任务 | 上一成功制品和脚本版本 |

MVP 中项目类型创建后不可更改，服务类型必须与项目类型一致。这样可以避免 Helm values、YAML overlay 和主机脚本在同一项目中被错误地混用。混合项目如有真实业务需求，后续通过“编排项目”聚合多个标准项目实现，不在第一阶段引入第四种类型。

## 3. 用户操作步骤

### 3.1 K8s YAML 项目

1. 创建项目，选择 `K8s YAML`，指定项目成员和角色。
2. 新建服务，关联代码源和 YAML 路径；平台解析资源、工作负载、容器和镜像替换位。
3. 选择构建模板和镜像仓库；平台保存不可变构建配置版本。
4. 创建环境，绑定已授权的集群和 Namespace，导入变量组并预览最终渲染 YAML。
5. 从“工作流”选择构建、部署、测试、审批节点，保存并校验 DAG。
6. 运行工作流，查看节点日志、资源差异、镜像 digest 和环境状态；失败可定位节点，成功后保存渲染清单用于回滚。

### 3.2 Helm Chart 项目

1. 创建项目，选择 `Helm Chart`。
2. 新建服务，关联 Chart 路径或 Chart 仓库，解析 `Chart.yaml`、默认 values 和组件镜像字段。
3. 配置构建或选择已有镜像制品；镜像不要求与 Chart 在同一仓库。
4. 创建环境，绑定集群、Namespace 和稳定 release name；配置全局、环境、服务三级 values 覆盖。
5. 工作流先生成 `helm template` 预览和 diff，再经审批执行 `upgrade --install --atomic --wait`。
6. 发布结果展示 Helm revision、values 快照、资源状态；回滚必须指定可审计的历史 revision。

### 3.3 主机应用项目

1. 创建项目，选择 `主机应用`。
2. 新建服务，配置代码源、二进制/压缩包产物规则、部署目录、部署/启动/停止脚本和健康检查。
3. 创建环境，绑定主机组；主机必须已注册 Agent、在线且符合标签筛选。
4. 工作流构建制品并生成 checksum，Agent 主动领取短期任务，下载制品后执行受限脚本。
5. 按滚动批次发布，逐批执行探活；失败自动停止后续批次并可回滚到上一成功制品。
6. 页面展示每台主机的任务、日志、版本、探活和回滚结果。

## 4. 变量和模板

变量优先级固定为：运行参数 > 服务环境覆盖 > 环境变量组 > 项目变量组 > 模板默认值。Secret 只能保存 credential reference，不能进入 values 快照、命令队列、日志或前端响应。

模板中心包含构建模板、YAML 服务模板、Helm Chart 模板、主机部署模板和工作流模板。模板版本不可变，实例保存 `template_version`，支持审计和可重复执行。

## 5. 工作流和治理

工作流以声明式 DAG 保存。节点包含 `build`、`yaml_deploy`、`helm_deploy`、`host_deploy`、`test`、`approval`、`notify` 和后续插件节点；每个节点必须定义输入、超时、重试、依赖和输出。保存时校验无环、类型匹配、项目类型兼容性和权限。

生产环境默认需要独立的 `approve` 权限。`deploy`、`approve`、`rollback`、`export` 等动作均按项目、服务、环境、集群或主机组范围授权，并写入审计记录。

## 6. 前端信息架构

项目内固定导航：概览、服务、环境、工作流、运行记录、制品、变量与模板、设置。创建流程使用三步向导，服务页采用左侧服务目录、中间 YAML/Chart/主机定义编辑器、右侧组件和镜像信息的三栏布局；运行详情使用稳定的 DAG 画布、节点列表、日志面板和资源/values diff，避免将配置页做成单一长表单。完整 Zadig 对齐范围见 [ZADIG_ALIGNMENT.md](ZADIG_ALIGNMENT.md)。

开发人员默认只能看到自己有权限的项目、服务、环境及只读日志；测试人员可在测试环境运行和查看；运维按授权执行生产审批、发布和回滚。
