# Agent 与可观测性方案

## 1. 产品目标

按 5000 台主机、多 Kubernetes 集群和 100+ 开发人员设计。平台 API 不负责逐台轮询，也不把所有指标、日志和事件塞入业务数据库。

## 2. Agent 分层

### HostAgent

职责：

- 注册、心跳、版本和标签
- 主机 CPU、内存、磁盘、网络和进程采集
- 接收经过授权的任务
- 返回任务状态和有限结果
- 支持灰度升级、暂停和回滚

### ClusterAgent

职责：

- 连接 Kubernetes API Server
- 资源和事件增量同步
- 采集 Pod、Node、Workload、Namespace 状态
- 受控执行工作负载操作
- 网络隔离时通过反向通道连接平台

Agent 不接受任意 shell。任务必须是平台定义的类型，例如：

```text
CollectMetrics
SyncResources
RestartWorkload
DeployArtifact
RunInspection
```

## 3. Agent 状态模型

```text
pending
online
offline
draining
upgrading
disabled
```

心跳不能只记录在线/离线，还要记录：

```text
agent_id
agent_type
version
capabilities
labels
last_seen_at
last_sync_at
load
connection
```

## 4. 5000 台主机的采集策略

- HostAgent 负责本机采集，平台不逐台 SSH。
- 指标优先使用 Prometheus node exporter/vmagent 体系。
- 平台只保存资产、标签、采集状态、规则配置和聚合结果。
- 按部门、环境、地域、主机组分片，避免单个 Worker 负责全部主机。
- 采集任务带租约、批次号和幂等键。
- 大批量规则变更采用分批发布和失败重试。

推荐初始链路：

```text
HostAgent/node_exporter -> regional vmagent -> VictoriaMetrics/Thanos/Mimir
                                           -> Alertmanager
                                           -> platform-observability
```

长期指标存储可根据实际保留周期和查询量选择 VictoriaMetrics 等 Prometheus 兼容方案。第一阶段不因为参考项目使用某个存储就强行引入。

## 5. Kubernetes 观测

ClusterAgent 增量同步：

```text
ClusterAgent -> Worker -> resource cache
             -> event pipeline -> alert/event store
             -> Prometheus/Loki query adapter
```

页面查询优先读取资源缓存，指标和日志查询走专用 provider。K8s Event 必须保留 namespace、kind、name、reason、message、first_seen、last_seen 和关联资源。

## 6. 日志方案

第一阶段优先 Loki：

- 统一查询接口和标签规范。
- 日志告警规则与 Prometheus 告警使用相同的项目、环境、服务标签。
- 支持 Pod 实时跟踪和查询时间范围限制。
- 导出需要单独权限和审计。
- 新部署优先使用 Vector，保留 Grafana Alloy 和 Fluent Bit Provider，不再使用已于 2026-03-02 结束生命周期的 Promtail。
- HostAgent 管理主机 Vector 的安装、升级、配置版本和状态；Kubernetes 使用平台管理的 DaemonSet。
- 日志默认进入区域 Gateway，再路由到 Loki；VictoriaLogs 和 Elasticsearch 作为可选 Provider。

保留 ELK adapter 的原因：

- 某些审计和全文检索场景更适合 Elasticsearch。
- 不把 Loki/ELK 的查询 DSL 泄露给前端，平台使用统一查询模型。

## 7. 告警和值班

告警生命周期：

```text
firing -> acknowledged -> assigned -> resolved
                    -> silenced
```

告警必须关联：

```text
project_id
environment_id
service_id
resource_id
rule_id
change_id
on_call_schedule_id
```

通知策略包括值班表、升级级别、维护窗口、静默时间和重复通知间隔。

## 8. 巡检模型

先实现数据模型和任务框架：

```text
inspection_rule
inspection_plan
inspection_task
inspection_evidence
inspection_result
inspection_remediation
```

第一阶段支持人工触发、只读检查和结果审计；第二阶段增加定时计划；第三阶段才允许经过审批的自动修复。

## 9. 参考项目的取舍

- 借鉴 KubeDoor 的多集群 Agent、事件、日志和资源使用分析。
- 借鉴 Pixiu 的隔离网络反向通道。
- 借鉴 AutoOps 的主机资产、任务和 Agent 管理。
- Prometheus/Loki 作为第一阶段标准入口，避免同时建设多套观测后端。
