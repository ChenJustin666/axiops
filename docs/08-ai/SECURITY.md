# AI 与 Agent 安全边界

> 目标：让 AI 能帮助巡检和排障，但不能把平台变成一个没有审计的远程 Shell。
>
> 当前产品范围以 `docs/AXIOPS_PRODUCT_REQUIREMENTS.md` 为准。本文是长期安全边界设计；
> 首期只启用 `observability/ai` 的监控和日志只读工具，其他工具仅作为未来注册契约。

## 1. 总体原则

```text
AI 负责理解、分析、规划
平台负责鉴权、校验、审批、执行、审计
Agent 负责在受限范围内执行任务
```

任何 AI 输出都不是权限凭证。AI 不能自行扩大资源范围、修改审批人或绕过环境策略。

## 2. Action Router

所有 AI 工具必须注册到 Action Router：

```text
AI 请求
 -> 意图识别
 -> 工具和参数校验
 -> 当前用户权限校验
 -> 资源范围校验
 -> 风险评估
 -> 依赖和前置检查
 -> 生成待执行动作
 -> 人工确认或自动执行
 -> worker 执行
 -> 结果和审计
```

### 2.1 工具定义

每个工具至少声明：

```text
tool_id
name
domain
input_schema
target_kind
required_permission
risk_level
read_only
approval_policy
timeout
retry_policy
audit_fields
```

示例工具：

- `k8s.get_workload`
- `k8s.get_events`
- `k8s.get_pod_logs`
- `prometheus.query`
- `loki.query`
- `host.collect_diagnostics`
- `delivery.preview_release`
- `delivery.rollback_release`
- `dms.explain_sql`
- `dms.execute_approved_change`

## 3. Direct、ReAct 和 Plan 模式

### Direct

只允许低风险、只读、参数明确的动作，例如查看 Pod 状态、查询指标和查看日志。

### ReAct

适合故障排查。每一步工具调用都要重新检查权限、目标和上下文，不能因为上一步允许就默认后续步骤全部允许。

### Plan + ReAct

适合复杂发布、巡检和变更。AI 先输出计划，平台把计划拆成待执行动作，用户确认后才可执行。

## 4. Preflight 检查

执行前必须检查：

- 用户是否拥有功能权限和操作权限。
- 目标资源是否在项目、环境、集群、Namespace、主机组或数据库范围内。
- 目标环境是否为生产。
- 是否存在变更窗口和审批单。
- 目标资源当前状态是否允许操作。
- 是否存在并发发布、锁或冲突。
- 参数是否完整，是否存在危险通配符。
- 预计影响范围、超时和回滚方式。

Preflight 失败时只能返回原因，不能自动降级为更高权限。

## 5. 高风险动作

以下动作默认需要人工确认，生产环境需要审批：

- 删除 Kubernetes 资源或 Namespace
- 修改生产 Deployment、Service、Ingress、Istio 路由
- 扩容、缩容或重启生产工作负载
- 在主机执行 Shell、安装包、修改防火墙或重启服务
- 生产发布、回滚和流量切换
- DDL、批量写 SQL、无条件 UPDATE/DELETE
- 修改告警静默和值班配置
- 创建或轮换平台凭据

写操作的待确认内容必须包含：

```text
动作
目标
参数摘要
风险等级
影响范围
变更前快照
回滚方案
审批单
预计耗时
```

## 6. Agent 安全模型

### 6.1 注册和身份

- Agent 使用一次性注册 Token 或 mTLS 注册。
- 注册后签发短期证书或可撤销 Token。
- 每个 Agent 有唯一 ID、版本、能力和所属主机组。
- 禁止多个主机共享永久 Token。

### 6.2 任务模型

Agent 只接受平台签发的 `TaskEnvelope`：

```text
task_id
tenant_id
project_id
environment_id
target_id
allowed_capabilities
command_template_id
parameters
expires_at
approval_id
signature
```

Agent 必须验证签名、过期时间、目标和能力。平台撤销 Agent 或任务后，Agent 不得继续接收新任务。

### 6.3 能力白名单

推荐能力：

- `metrics.collect`
- `logs.collect`
- `process.snapshot`
- `disk.snapshot`
- `network.diagnose`
- `package.read`
- `service.read`
- `script.execute_template`

默认禁止：

- 任意 Shell
- 任意文件上传下载
- 修改防火墙
- 修改用户和 SSH
- 读取私钥
- 访问平台数据库
- 提升自身权限

## 7. Skill/SOP 约束

AI 不直接拼装高风险命令。每个 Skill 必须包含：

- 适用场景
- 输入参数
- 允许资源范围
- 只读步骤
- 写操作步骤
- 风险级别
- 需要的审批
- 回滚方式
- 超时和重试
- 失败处理

平台可以参考 SXDevOps 的知识图谱、Action Router、Agent Mode、Pending Action 和统一任务中心，但不能照搬不完整的工具鉴权。

## 8. 审计要求

以下事件必须审计：

- 用户登录、登出和权限变化
- AI 会话和模型版本
- AI 生成的计划
- 工具调用和参数摘要
- Preflight 结果
- 审批、拒绝、撤销和超时
- Agent 注册、心跳、升级和撤销
- K8s、主机、发布和数据库执行结果

审计事件最少包含：

```text
audit_id
request_id
operator_id
department_id
project_id
resource_kind
resource_id
action
risk_level
approval_id
before_snapshot
after_snapshot
result
error_code
created_at
```

审计日志不能依赖业务管理员的普通删除权限。归档、留存和查询应独立授权。

## 9. DMS 安全规则

- DMS 只能通过数据库代理或受控连接池访问目标数据库。
- 平台 API 不得直接把数据库密码交给前端。
- 连接按实例、库、表和环境授权。
- SQL 先解析，禁止多语句和危险注释绕过检查。
- SELECT 结果默认脱敏，限制最大行数和导出。
- 写 SQL 必须有变更申请、审批、执行窗口和回滚说明。
- 执行人和审批人分离。
- 生产连接使用只读账号作为默认账号。
- SQL 原文和结果按敏感级别加密存储，普通日志只保留摘要。

## 10. 提示词和数据安全

- 不把 Secret、Token、私钥、完整 SQL 结果和敏感日志直接发送给外部模型。
- 发送模型前做字段脱敏、长度限制和上下文裁剪。
- 模型输出必须视为不可信输入，所有参数重新做 Schema 校验。
- 知识库按项目、环境和用户权限过滤。
- 模型调用、工具调用和数据访问都要可追踪。

## 11. 安全验收用例

```text
无角色用户 -> 不能调用任何业务写 API
只读用户 -> 不能触发发布、重启、SQL 执行
项目 A 用户 -> 不能查询项目 B 的日志和指标
测试环境权限 -> 不能操作生产 Namespace
AI 无审批 -> 只能生成计划，不能执行写操作
被撤销 Agent -> 不能接收新任务
DMS 申请人 -> 不能审批自己的 SQL
过期 TaskEnvelope -> Agent 拒绝执行
```
