# 旧模块权限接入核对（2026-10-16）

本文件记录实际入口、已修复项和接入缺口。新版权限中心的监控/日志选项还没有控制这些旧接口，不能将规则单元测试视为模块验收。

## 入口与边界

| 能力 | 当前入口/代码 | 接入需要 |
| --- | --- | --- |
| 账号鉴权 | `common/middleware/JWTAuthMiddleware.go` → portal-rpc `VerifyToken` → `ApiAuth` | 已补齐每次读取当前账号/角色。保留原 Casbin API 鉴权，不能用新版菜单权限替代 |
| 日志检索/导出 | manager-api `/manager/v1/logging`，QueryLogs、DownloadLogs | 查看与导出分开授权；同一服务端解析的集群/Namespace/项目范围；不能只检查请求声明的 ProjectUuid |
| 日志元数据 | ListLogLabelKeys、ListLogLabelValues、GetLogQueryModeOptions | 可见标签和值也必须限范围，不能借元数据查询发现其他项目 |
| 日志规则/事件 | List/Get/Create/Update/DeleteLogAlertRule、SearchLogAlertEvents 等 | 先按数据库记录定位归属；创建/移动同时校验新旧范围；列表计数和分页一致。集群全局规则与项目规则分开 |
| 应用指标 | console-api 的 pod/namespace/cluster/node monitor 路由，workload-api monitor 路由 | 项目环境授权解析为指定集群和 Namespace；集群/节点总览单独授权；监控查询本身也限制范围 |
| 监控管理 | manager-api `/manager/v1/monitoring` 的 Alertmanager/Prometheus 配置 | 配置可能含通知凭据；不因有指标查看权限就返回原始配置或允许修改 |
| 容器日志/流/下载 | console-api pod logs/get、logs/stream、logs/download 等 | 使用相同资源边界；流连接增加到期/撤销检查。初次握手通过不等于长期授权 |
| AI | 本次未在 application/internal/cmd/common 找到实际 AI 调用/检索/工具后端 | 当前只有架构契约，接入真实接口时再分别验证调用、来源读取、工具操作；不能声称已有 AI 权限接入完成 |

## 本次完成：账号撤销

旧普通会话此前直接返回 JWT 中登录时的角色。现在 JWT 验签和 Redis 会话校验后，再使用 `SQLPrincipalReader` 查询当前用户和角色绑定，一条 SQL 保证用户状态与角色来自同次数据库快照。禁用/删除/要求重置密码的账号无效；已删除绑定或角色不参与授权；没有角色的有效账号仍保持零权限，不补管理员角色。

刷新令牌也重新获取当前账号和角色，不能签发带旧管理员角色的新访问令牌。系统 Token 每次直接读取当前记录，校验停用/删除/到期和当前用户或角色所有者；不依赖旧缓存、不返回 Token 明文。角色所有者 Token 保留原有自动化身份语义，不伪造用户 ID。

这些修改只处理请求时的身份与角色。Casbin 策略的分布式刷新、已有长连接的撤销、新版资源范围授权，仍是独立事项。

## 显式资源映射

现有 `identity_link` 只关联旧账号和新版账号，没有项目/集群的对应关系。旧代码使用 OnecProject/ProjectCluster/Workspace/UUID；新版使用 project/cluster/project_environment。不能将同一个数字 ID 当成同一资源，也不能通过名称相似自动授权。

后续映射必须由服务端维护并验证关联有效：旧集群 UUID → 新集群；旧项目 UUID → 新项目；旧工作空间/Namespace → 新项目的具体环境。开发/测试共用集群时，集群映射不能代替 Namespace 和项目环境匹配。未绑定、失效、归属矛盾或匹配多个目标时应拒绝，并提示管理员处理映射。

保留旧权限作为第一层，新版动作与资源策略作为另一层；不通过复制旧角色 ID 或发放通配权限解决迁移。新旧账号停用状态均应生效，授权前重新确认实际目标，客户端 scope 不作为证据。

## 查询和验收要求

当前 Loki `buildQuery` 在 code 模式直接采用原表达式，不能假设请求中的 Namespace/ProjectUuid 已限制表达式。因此资源接入还必须在数据源查询层保证边界。未实现可靠限制前，不向局部授权账号开放任意原始查询；不能只在前端禁用输入。

使用普通账号逐一验证：本范围读取成功、跨项目/跨 Namespace 被拒绝、查看不能导出或改规则、标签查询不泄漏、原始表达式不能越界、分页总数一致、撤销后新请求/长连接行为明确。用隔离数据和受控后端测试，真实用户、业务发布、重启、消息发送不作为权限测试手段。

本次没有修改页面、没有迁移或业务数据改动，没有替换运行进程。验证记录见 HANDOFF 最新节。
