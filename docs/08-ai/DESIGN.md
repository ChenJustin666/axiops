# AI 运维设计

## 1. 产品定位

> 当前范围以 `docs/AXIOPS_PRODUCT_REQUIREMENTS.md` 为准。本文描述长期 AI 安全模型；首期 AI 位于
> `observability/ai`，只覆盖监控和日志，不单独启动 `ai-ops`。

AI 是运维平台的理解、检索、分析和任务编排入口，不是绕过权限的超级管理员。

第一阶段目标：

- 根据用户问题查询授权范围内的告警、指标、日志和观测事件。
- 给出带证据的故障假设和排查步骤。
- 生成日志解析规则和只读巡检报告草稿。
- 关联最近发布只读取平台提供的事件摘要，不调用发布操作。

资产操作、发布操作、主机任务、DMS 和跨领域 Action Router 属于后续阶段。

## 2. 强制安全边界

```text
AI Assistant
  -> Intent Parser
  -> Skill/SOP
  -> Action Router
  -> RBAC + Data Scope + Environment Policy
  -> Preflight
  -> Pending Action / Approval
  -> Platform API
  -> Provider / Worker / Agent
  -> Audit
```

禁止：

- AI 直接执行 `kubectl`、SSH 或 SQL。
- AI 自己拼接任意 URL 调用内部服务。
- AI 绕过项目、环境和资源范围。
- AI 在生产环境自动执行重启、发布、删除或数据库写入。
- AI 把凭证、Token、SQL 原文和敏感日志直接放入模型上下文。

## 3. Skill/SOP

每个 Skill 是经过审核的能力包：

```text
skill_id
name
version
purpose
input_schema
allowed_actions
required_permissions
preflight_rules
approval_policy
evidence_requirements
```

例子：

```text
inspect-pod-crash
analyze-alert
compare-release
check-host-disk
generate-k8s-readonly-report
```

Skill 只能调用注册的 Tool，不能携带任意代码执行。

## 4. Tool Registry

工具按资源域注册：

```text
cluster.get_resource
cluster.list_events
workload.get_status
log.query
metric.query
alert.get_detail
release.get_history
host.get_health
dms.explain_sql
```

写操作工具必须声明风险级别：

```text
read
low_risk_write
high_risk_write
production_change
```

## 5. Pending Action

所有写操作先创建：

```text
pending_action
  id
  requester
  intent
  skill
  tool
  resource
  environment
  parameters_hash
  preflight_result
  approval_status
  expires_at
```

用户确认后，平台重新做一次权限和预检，再交给 Worker 执行。确认内容必须展示资源、环境、参数摘要、风险和回滚方式。

## 6. 故障排查流程

```text
告警
 -> 定位项目/环境/服务/资源
 -> 查询最近发布和变更
 -> 查询指标和日志
 -> 执行只读诊断 Skill
 -> 输出证据、假设、置信度和建议
 -> 需要修改时生成 Pending Action
```

AI 回复必须区分：

- 已确认事实
- 查询到的证据
- 推断出的可能原因
- 尚未验证的建议

## 7. 审计

审计对象包括：

- 用户问题和会话 ID
- 模型和提示版本
- Skill 版本
- Tool 调用参数摘要
- 权限决策
- 预检结果
- 审批记录
- 实际执行结果

敏感参数只保存哈希、脱敏摘要或密文引用，不保存明文凭证。
