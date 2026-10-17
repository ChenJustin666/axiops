# 权限模块详细设计

> 状态：后端与权限中心前端已完成并通过代码门禁，待环境 E2E  
> 更新日期：2026-08-18

## 1. 架构结论

采用 `RBAC + scoped binding + environment policy`：

```text
用户 -> 平台角色 / 项目成员 / 资源绑定 -> resource/action/effect -> 范围匹配 -> allow/deny
```

规则：

- 默认拒绝，没有明确 allow 即返回 403。
- 任意匹配的显式 deny 优先于 allow。
- 后端同时控制列表、详情和写操作，前端隐藏菜单不是安全边界。
- 策略实时从数据库读取，不使用 JWT 中的权限快照。
- 禁用或软删除用户不再产生任何有效策略。
- 临时绑定超过 `expires_at` 自动失效。

## 2. 角色与绑定

角色字段 `binding_scope` 只能是：

| 值 | 绑定位置 | 典型角色 |
|---|---|---|
| `platform` | `user_role` | `super_admin`、`platform_admin`、`operator` |
| `scoped` | `project_member`、`role_binding` | `project_admin`、`developer`、`tester`、自定义只读/运维角色 |

防提权约束：

- `platform` 角色不能写入 `role_binding` 或 `project_member`。
- `scoped` 角色不能写入 `user_role`，也不能绑定 `platform/0`。
- 已有绑定不兼容时禁止修改角色的 `binding_scope`。
- 普通平台管理员不能分配、修改或移除超级管理员，也不能通过自定义角色写入或分配 `resource=*`、`system_role` 等价权限。
- 系统角色修改要求 `system_role:update`，超级管理员分配要求 `system_role:assign`。

内置运维角色是全平台角色。若只允许管理指定项目或集群，应创建 `scoped` 运维角色并绑定具体资源。

## 3. 资源范围

| `scope_type` | `scope_id` | 用途 |
|---|---:|---|
| `platform` | 0 | 平台级角色，仅通过 `user_role` |
| `project` | 项目 ID | 项目下服务、构建、日志和发布 |
| `service` | 服务 ID | 指定服务 |
| `build` | 构建 ID | 指定构建任务或日志 |
| `workflow` | 工作流 ID | 指定工作流的查看、执行、取消或审批 |
| `cluster` | 集群 ID | 指定 Kubernetes 集群 |
| `environment` | 环境 ID | 指定发布环境，同时校验环境名称 |

`environment` 范围必须同时提供有效环境 ID 和非 `*` 环境名称。CI/CD 构建本身没有环境语义时使用 `environment='*'`；发布接口必须传入真实环境 ID 和名称进行鉴权。

部门主体当前只匹配用户的直属 `department_id`。部门树递归继承必须在部门路径模型、变更传播和审计规则完成后显式开启。

## 4. 动作与角色模板

动作统一为 `view/create/update/delete/run/cancel/deploy/approve/rollback/export`。兼容旧策略 `read -> view` 和 `write -> create/update/delete`。

- 开发：授权项目/服务内查看服务、构建、日志并触发构建；默认不能改配置或生产发布。
- 测试：测试范围内查看与触发验证构建；默认无生产权限。
- 项目管理员：管理指定项目成员、服务、构建和凭据。
- 运维：生产资源、集群、发布、监控和日志；不自动拥有用户、角色和授权管理权限。
- 平台管理员：用户、角色、菜单和授权管理；不能管理超级管理员。
- 只读：仅 view，不允许触发、取消、修改、发布或回滚。

## 5. 接口鉴权

| 能力 | `resource/action` | 范围 |
|---|---|---|
| 服务列表、详情 | `service:view` | project/service |
| 服务写操作 | `service:create/update/delete` | project/service |
| 构建列表、详情 | `build:view` | project/service/build |
| 触发、重试 | `build:run` | project/service/build |
| 取消构建 | `build:cancel` | project/service/build |
| 构建日志 | `build_log:view` | project/service/build |
| 构建配置 | `build_config:view/update` | project/service |
| 集群列表、详情 | `cluster:view` | cluster |
| 集群写操作、连接测试 | `cluster:create/update/delete` | cluster/platform |
| 用户、角色、菜单、授权 | 对应资源动作 | platform |

列表查询将允许和拒绝范围转换成参数化 SQL；详情与写操作加载资源归属后再次鉴权，避免通过猜测 ID 越权。

## 6. 菜单与前端

- `role_menu` 决定角色菜单。
- 当前用户菜单合并平台角色、项目成员和资源绑定的菜单。
- 如果只绑定子菜单，API 自动补齐所有可见父菜单。
- 超级管理员通过通配权限看到全部可见菜单。
- 前端应使用 `GET /api/v1/auth/menus` 生成路由，使用 `GET /api/v1/auth/access` 控制按钮；所有按钮操作仍由后端复核。

## 7. 审计

认证后的非 GET/HEAD/OPTIONS 请求统一记录到 `audit_log`：操作人、动作、资源、资源 ID、HTTP 方法、路径、结果状态、对端 IP、User-Agent 和耗时。

请求体不写入审计表，避免密码、Token、云凭据、仓库凭据和 kubeconfig 泄漏。反向代理后的原始客户端 IP 只有在后续配置受信代理列表后才能采信，当前记录直接连接的对端地址。

## 8. 事务与一致性

- 角色、权限、菜单创建或更新使用同一数据库事务。
- 用户与初始平台角色使用同一事务。
- 项目与创建者的 `project_admin` 成员关系使用同一事务。
- `project_admin` 按角色编码查询，禁止依赖固定数据库 ID。
- 授权更新通过 `clear_expires_at` 区分“不修改”和“清空过期时间”。

## 9. Helm 与通知衔接

Helm 是发布 provider，不放入构建容器。发布请求解析 `project/service/environment/cluster/namespace` 后校验 `release:deploy`；生产环境默认只授予运维类角色。保存 chart、values 摘要、镜像 digest、revision 和回滚点。

构建和发布事件写入 outbox，由 worker 异步发送飞书、钉钉、企业微信、邮件和通用 Webhook。通知包含状态、失败阶段、提交、制品、耗时、失败原因、日志链接和操作人；通知失败不改变构建结果。

## 10. 剩余边界

- 新权限中心直接使用 `/api/v1`；旧 `/portal/v1` 用户/角色页面不再作为新权限入口。过渡期仍必须完成用户 ID/角色映射，不能混用两套身份。
- 凭据和 kubeconfig 已有访问控制，但仍需信封加密、密钥轮换和脱敏展示。
- 部门树递归授权尚未开启。
- 数据库迁移由版本记录保证一次执行；`004_authorization_scope.sql` 不是重复初始化脚本。
