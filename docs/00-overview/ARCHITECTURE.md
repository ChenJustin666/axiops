# AxiOps 总体架构

> 状态：方案基线 v1.0  
> 日期：2026-08-04  
> 适用规模：100+ 开发人员、约 5000 台监控主机、多 Kubernetes 集群、ECS/物理机和数据库

## 1. 结论

本项目不建议现在切换 Gin，也不建议推倒重写。当前代码已经具备 Kubernetes、DevOps、Prometheus、Loki/ES、Istio、权限和前端页面基础，最合适的路线是：

1. 继续使用 Go-zero 和现有 API/RPC 代码。
2. 先按领域重新划分 package 和数据边界，再减少运行进程。
3. 计费模块冻结并逐步移除，不再作为项目、集群和工作空间的前置依赖。
4. 巡检保留模型、规则和报告，自动化执行后置。
5. DMS 独立边界，禁止直接混入普通 Kubernetes 操作权限。
6. AI 只做理解、分析、规划和调用受控工具，写操作必须经过审批和审计。

目标不是把所有能力做成一个巨型服务，而是让用户看到一个统一平台，同时让后端按领域保持清晰边界。

## 2. 产品边界

### 2.1 首期必须有

- 统一登录、用户、部门、角色、菜单和 API 权限
- 项目、环境、集群、命名空间、主机和数据库资产
- Kubernetes 多集群接入、工作负载和 Istio 基础管理
- Prometheus 兼容监控、告警规则、通知和值班
- Loki 默认日志接入，保留 Elasticsearch 查询适配器
- CI/CD 项目、凭据、流水线、发布、审批和执行记录
- 异步任务、集群同步、告警处理和操作审计
- 监控和日志内的 AI 对话、证据检索、故障分析和解析规则草稿

### 2.2 后置能力

- 自动巡检执行和巡检报告
- ECS/主机通用批量命令和高风险任务；监控、日志 Collector 的 bootstrap 与受控 Agent 属于首期
- DMS 数据库执行、变更审批和 SQL 审计
- 资源建议、容量预测和成本分析

### 2.3 明确不做

- 不做 SaaS 计费、账单、价格套餐和资源收费结算
- 不自研 Jenkins、Tekton、Prometheus、Loki、ES、Kubernetes
- 不让 AI 直接执行任意 Shell、kubectl、SQL 或删除操作
- 不在平台数据库保存明文 kubeconfig、SSH 私钥、数据库密码

## 3. 目标运行拓扑

```text
Browser / CLI / Webhook
          |
     API Gateway
          |
   axiops-platform-api
   (platform-core package)
          |
  +-------+--------------------------+
  |                                  |
axiops-cicd-api       axiops-observability-api
  |                       Prom/Loki/ES + AI
CI Provider                          |
  +------------- axiops-task-worker --+
                            |
             K8s API / Cloud API / axiops-host-agent / Notification

后置独立边界：`axiops-dms-api`；AI 达到独立扩缩容、安全隔离或跨领域编排条件后拆分 `axiops-ai-api`。
```

### 3.1 推荐生产进程

| 进程 | 职责 | 建议初始副本 |
| --- | --- | ---: |
| `axiops-platform-api` | 统一 HTTP API、认证、鉴权、项目、集群、资产、Kubernetes；内含 `platform-core` 领域包 | 2 |
| `axiops-cicd-api` | CI/CD 配置、发布、审批、流水线适配 | 2 |
| `axiops-observability-api` | 监控、日志、告警、值班、规则、巡检 API，以及首期 `observability/ai` | 2 |
| `axiops-task-worker` | 同步、异步任务、告警计算、定时巡检、Agent 任务 | 2-4 |
| `axiops-dms-api` | 数据库连接、SQL 执行、变更审批、审计 | 2，生产建议独立节点 |

首期运行 `axiops-platform-api`、`axiops-cicd-api`、`axiops-observability-api`、`axiops-task-worker` 4 类平台进程。DMS 上线后增加 `axiops-dms-api`；AI 达到独立扩缩容、模型网关、安全隔离或跨领域编排条件后，再从 `observability/ai` 拆出 `axiops-ai-api`。`platform-core` 保留代码和领域边界，首期编译进 `axiops-platform-api`，避免为低流量内部调用额外维护一套 RPC 进程；达到独立团队、独立扩缩容或安全隔离条件后再拆。

边缘部署单元统一命名为 `axiops-host-agent`、`axiops-cluster-agent`、
`axiops-metrics-gateway` 和 `axiops-logs-gateway`。Gateway 可以封装成熟采集或代理组件，
不要求自研为独立业务微服务，也不计入上述中心服务数量。

### 3.2 当前到目标的映射

| 当前目录/服务 | 目标边界 | 处理 |
| --- | --- | --- |
| `portal-api`、`portal-rpc` | `platform-api/core/identity` | 保留逻辑，统一入口 |
| `manager-api`、`manager-rpc` | `platform-api/core/resource` | 拆出资产、项目、集群、策略 |
| `workload-api` | `platform-api/resource/k8s` | 合并为资源 API |
| `console-api`、`console-rpc` | `observability`、`platform-api/core/registry` | Pod 控制台和监控按领域归位 |
| `devops-api`、`devops-manager-rpc` | `delivery` | 配置、凭据、模板、发布 |
| `devops-pipeline-rpc` | `delivery` + `worker` | 执行状态和异步任务分开 |
| `common/k8smanager` | `platform-api/core/provider/kubernetes` | 去除全局单例，按集群注入 |
| `common/prometheusmanager` | `observability/provider/prometheus` | 统一 Provider 接口 |
| `common/logmanager` | `observability/provider/log` | Loki 默认，VictoriaLogs/ES 可选 |
| `manager billing` | 无目标边界 | 冻结、下线、最后删除 |
| `manager inspection` | `observability/inspection` | 保留规则和报告，执行后置 |
| DevOps host | `platform-api/core/asset/host` | 从发布配置中抽为资产 |
| 新增 | `observability/ai`、`dms` | AI 首期内聚于观测域；DMS 后续独立安全边界 |

## 4. 领域模型

### 4.1 统一资源层级

```text
组织
  -> 部门
    -> 用户 / 用户组
      -> 项目
        -> 环境
          -> 集群 / 主机组 / 数据库实例
            -> Namespace / 应用 / 服务 / 主机 / 数据库
```

所有业务资源必须至少具备：

- `tenant_id` 或组织标识
- `project_id`
- `environment_id`
- `owner_dept_id`
- `labels`
- `status`
- `created_by`、`updated_by`
- 资源版本或 `resource_version`

### 4.2 资源抽象

平台统一暴露 `ResourceRef`：

```text
kind: cluster | namespace | workload | host | database | alert_rule | pipeline
provider: kubernetes | ecs | ssh | mysql | postgres | prometheus | loki
scope: project/environment/namespace/host-group/database-instance
id: provider 内部唯一标识
```

这样可以让权限、审计、AI 工具和任务中心使用同一套资源范围，而不是每个模块自己解释资源。

## 5. 权限设计

当前的角色、菜单和 API/HTTP Method 权限可以继续使用，但必须增加数据范围层。

### 5.1 权限四层

```text
功能权限：能否看到监控、集群、发布、DMS
操作权限：能否查看、创建、更新、删除、执行、审批
数据范围：只能访问哪些部门、项目、集群、Namespace、主机、数据库
环境策略：生产环境是否需要二次审批、双人复核或禁止高危操作
```

### 5.2 推荐数据范围

| 范围 | 示例 |
| --- | --- |
| `ALL` | 平台管理员 |
| `DEPT` | 本部门及下级部门 |
| `PROJECT` | 指定项目 |
| `ENVIRONMENT` | 指定项目的测试/预发环境 |
| `CLUSTER` | 指定集群 |
| `NAMESPACE` | 指定命名空间 |
| `HOST_GROUP` | 指定主机组 |
| `DATABASE` | 指定数据库实例或库 |
| `SELF` | 仅本人创建或负责的资源 |

最终授权判断：

```text
allow =
  function_permission
  && operation_permission
  && resource_scope_match
  && environment_policy_pass
```

没有角色、没有数据范围或资源不匹配时默认拒绝。不能用“默认 user/super admin”兜底。

### 5.3 典型角色

| 角色 | 可见范围 | 典型操作 |
| --- | --- | --- |
| 平台管理员 | 全部 | 用户、角色、系统配置 |
| 项目管理员 | 指定项目 | 项目成员、集群和环境配置 |
| 开发者 | 指定项目/测试环境 | 查看资源、触发测试发布 |
| 发布管理员 | 指定项目/生产环境 | 发布、回滚、审批流转 |
| 监控管理员 | 指定项目/集群 | 规则、告警、值班 |
| 只读观察者 | 指定范围 | 只读查看监控、日志、资源 |
| DMS 审批人 | 指定数据库 | 审批 SQL 和变更 |
| DMS 执行人 | 指定数据库 | 执行获批任务，不得自批自执行 |

## 6. Kubernetes、ECS 和 Istio

### 6.1 Kubernetes

- 平台只保存集群引用和加密凭据，不把 kubeconfig 明文返回前端。
- 每个集群有 Provider 实例和同步任务，禁止全局单例污染多集群状态。
- 所有写操作先生成 `OperationTask`，记录请求、差异、审批和结果。
- 集群同步采用增量 Watch，定期全量校准，避免只依赖单次全量扫描。
- 工作负载页面统一从 `project/environment/namespace` 上下文进入。

### 6.2 ECS/主机

主机资产与 CI Agent 解耦：

- `HostAsset`：IP、主机名、云厂商、区域、操作系统、负责人、主机组、环境
- `HostAgent`（运行名 `axiops-host-agent`）：Agent ID、版本、心跳、能力、标签、证书、最后任务
- `HostCredentialRef`：只引用 Secret/Vault，不保存私钥内容
- `HostTask`：脚本、Ansible、巡检和发布任务

Agent 采用主动连接模式，适用于私网、跨地域和无入站网络。高风险命令只能由任务模板产生，禁止前端直接传入任意命令。

### 6.3 Istio

保留 Istio VirtualService、DestinationRule、Gateway、流量分配和 Flagger 能力，但放入 `platform-api/core/provider/mesh` 和 `delivery/release`：

- 平台负责资源模型、差异、审批和审计。
- Istio 控制面负责真正的流量治理。
- 金丝雀发布必须绑定 Prometheus 指标和自动回滚条件。
- 不把 Istio 专有字段散落到普通工作负载模型。

## 7. 可观测性设计

### 7.1 指标

默认链路：

```text
node-exporter / kube-state-metrics / app metrics
        -> vmagent 或 Prometheus Agent
        -> remote_write
        -> VictoriaMetrics / Thanos / Mimir
        -> platform query API / Grafana
```

5000 台主机不建议把全部历史数据只留在单实例 Prometheus。平台自身只保存数据源、规则、标签映射和查询审计。

统一外部标签至少包括：

```text
tenant, project, environment, cluster, region, host_group
```

### 7.2 日志

默认使用 Loki，保留 VictoriaLogs 和 Elasticsearch 适配器：

```text
Host Agent / Vector / Grafana Alloy / Fluent Bit
        -> regional log gateway
        -> Loki / VictoriaLogs / Elasticsearch
        -> platform log query API
```

新建设计不再使用 Promtail；其生命周期已于 2026-03-02 结束。默认采用 Loki 与对象存储；VictoriaLogs 用于偏好 Victoria 体系、字段查询和低资源运行的场景；ES 仅在复杂全文检索、合规或已有集群场景启用。平台不复制海量日志，只保存采集任务、配置版本、查询配置、索引策略、规则和审计信息。

### 7.3 告警和值班

```text
PrometheusRule / Loki Rule
        -> Alertmanager / Loki ruler
        -> alert-normalizer
        -> dedup / grouping / silence
        -> on-call schedule
        -> notification channel
        -> incident / task
```

告警必须关联资源和责任人，不能只发送一条无上下文文本。

## 8. CI/CD 和流程

`delivery` 只做统一编排和适配，不取代 Jenkins、Tekton、GitLab CI 或云厂商流水线。

统一对象：

- `DeliveryProject`
- `PipelineTemplate`
- `PipelineRun`
- `ReleasePlan`
- `Approval`
- `Artifact`
- `CredentialRef`
- `ProviderChannel`

推荐执行链：

```text
代码变更
 -> 构建
 -> 质量检查
 -> 镜像签名/入库
 -> 测试环境部署
 -> 指标验证
 -> 生产审批
 -> 灰度/金丝雀
 -> 自动或人工放量
 -> 结果和审计
```

首期只选一个主执行器。已有 Jenkins 的组织优先接 Jenkins；以 Kubernetes 为主且希望云原生执行时再接 Tekton。不要同时把两套执行模型都做成首期一等公民。

## 9. DMS 边界

DMS 必须单独部署或至少单独进程、数据库连接池和权限域：

- 数据库凭据只存在 Secret/Vault。
- 生产 SQL 默认只读。
- 写 SQL、DDL、批量更新、无条件删除必须审批。
- 执行前做 SQL 解析、风险评级、影响行数和超时校验。
- 执行人不能审批自己的申请。
- 结果脱敏后保存，禁止把完整敏感结果写入普通日志。
- 支持数据库实例、库、表和字段级授权。

## 10. 容量基线

初始目标不是保证所有极限值，而是建立可压测、可扩容的边界：

| 指标 | 初始目标 |
| --- | ---: |
| 开发用户 | 100-300 |
| 监控主机 | 5000 |
| Kubernetes 集群 | 20-50 |
| 平台 API 峰值 | 100-300 RPS |
| 并发登录/查询 | 50-100 |
| 同时运行平台任务 | 100-300 |
| Agent 心跳 | 5000 台每 30-60 秒 |
| 告警突发 | 1 万条/分钟可削峰 |
| 审计保存 | 热数据 90 天，归档按合规策略 |

容量设计原则：

1. API 无状态，至少 2 副本。
2. 同步、告警、巡检和发布全部异步化。
3. Redis 只承担缓存、锁、短任务状态和轻量队列；需要持久事件流时再引入 NATS/Kafka。
4. 指标和日志存储独立扩展，不让平台 MySQL 承担时序数据和日志正文。
5. 所有 Provider 调用设置超时、并发上限、重试和熔断。

## 11. 生产非功能要求

- 所有写操作有审计 ID。
- 所有异步任务可查询、可取消、可重试、可补偿。
- 所有外部凭据可轮换、可撤销、可验证。
- 关键 API 有幂等键。
- 生产发布和 DMS 写操作有审批记录。
- 平台自身暴露 Prometheus metrics、健康检查和结构化日志。
- 每个模块至少有领域单测、Provider contract test 和关键 API 集成测试。

## 12. 参考资料

- Kubernetes RBAC：<https://kubernetes.io/docs/reference/access-authn-authz/rbac/>
- Kubernetes 日志架构：<https://kubernetes.io/docs/concepts/cluster-administration/logging/>
- Prometheus remote write：<https://prometheus.io/docs/practices/remote_write/>
- Loki 文档：<https://grafana.com/docs/loki/latest/>
- 跨云监控、日志和前端设计：见 [监控模块](../03-monitoring/README.md)、[日志模块](../04-logging/README.md) 和 [前端模块](../09-frontend/README.md)。
- Istio 流量管理：<https://istio.io/latest/docs/concepts/traffic-management/>
- 项目本地参考：`reference/AutoOps`、`reference/sxdevops`、`reference/KubeDoor`、`reference/pixiu`

### 12.1 Zadig 借鉴边界

Zadig 纳入交付域参考，重点借鉴：

- 项目、服务、环境和工作流的组织方式
- 构建配置、制品、环境变量和发布参数的分层
- 工作流阶段、并行任务、人工审核、发布结果和回滚
- 将构建执行器作为可替换 provider，而不是把执行器写死在业务模型中

本项目不直接复制 Zadig 的完整控制面。交付只负责构建、发布和变更编排，集群、主机、监控、日志、DMS 和 AI 继续由各自领域负责。最终交付模型统一为：

```text
Project
  -> Service
    -> Environment
      -> Build
      -> Artifact
      -> Workflow
      -> Release
      -> Approval
      -> Audit
```
