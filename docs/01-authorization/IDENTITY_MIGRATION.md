# 身份收敛实施方案

> 状态：桥接基础与只读盘点工具已完成，身份数据导入待执行  
> 更新日期：2026-08-10

## 1. 问题

现有 Vue 前端和 `/portal/v1`、`/manager/v1`、`/workload/v1` 使用旧 `autoops` 身份库、旧 JWT 格式以及 Redis Token 吊销校验。目标 `platform-api` 使用 `axiops` 的 `user/role/user_role` 模型和新 JWT。

两套系统的用户 ID、角色 ID、JWT claims、签名密钥和吊销机制都不能假定相同。直接把前端请求改到 `/api/v1` 会造成登录失败，或者更危险的身份错配。

## 2. 决策

- 长期唯一身份源是 `axiops` 的身份模型。
- 过渡期不复制旧 JWT 密钥到各新服务，不让新服务自行猜测旧 Token claims。
- 旧 Token 由旧 `portal-rpc VerifyToken` 校验签名、有效期、Redis 吊销状态和账号状态。
- 校验成功后，通过显式身份映射表转换为目标用户 ID，再进入统一授权内核。
- 不信任浏览器或网关直接传入的用户 ID、角色或部门请求头。

## 3. 数据模型

新增 `identity_link`：

| 字段 | 说明 |
|---|---|
| `provider` | 身份源，旧平台固定为 `legacy_autoops` |
| `external_user_id` | 旧 `sys_user.id` |
| `user_id` | 新 `user.id` |
| `external_username` | 迁移核对使用，不作为授权主键 |
| `status` | active/disabled |
| `created_at/updated_at` | 审计时间 |

`provider + external_user_id` 和 `user_id + provider` 必须唯一。禁止仅按用户名在每次请求时临时匹配，避免重命名和重复账号造成串号。

## 4. 过渡期认证流程

```text
Browser legacy token
  -> platform-api LegacyTokenVerifier
  -> portal-rpc VerifyToken
  -> identity_link
  -> target user status check
  -> authz policies
  -> API handler
```

新平台原生 Token 继续走本地验证。认证中间件只负责得到可信的目标 `user_id`，角色和权限始终从目标数据库实时读取。

旧 Token 验证结果可以按 Token 哈希短时缓存，但缓存 TTL 必须小于吊销容忍时间；不保存明文 Token。验证服务不可用时默认拒绝，不降级为解析但不验签。

## 5. 实施步骤

1. 盘点旧用户、部门、角色和重复账号，生成只读迁移报告。
2. 建立 `identity_link` 迁移和唯一约束。（已完成）
3. 编写审核清单驱动的幂等映射导入任务；默认只校验，显式 `--apply` 才写入。（映射工具已完成，账号/部门迁移待盘点后决定）
4. 在 `platform-api` 增加可插拔 `LegacyTokenVerifier`，通过 portal-rpc 验证旧 Token。（已完成，默认关闭）
5. 增加身份映射失败、禁用用户、Token 吊销、认证服务超时和默认拒绝测试。（基础单测已完成，集成测试待真实环境）
6. Vite 增加 `/api/v1 -> platform-api` 代理，前端资源授权页开始使用新接口。
7. 逐模块把旧 API 迁到目标服务；全部迁移后切换到平台原生登录并删除旧 Token 适配器。

只读盘点命令：

```powershell
go run ./cmd/identity-inventory -f configs/identity-inventory.yaml
```

配置模板为 `configs/identity-inventory.example.yaml`。实际运行时复制为未纳入版本控制的 `configs/identity-inventory.yaml`。工具只执行 SELECT 并输出 JSON，不创建用户、角色或映射。建议使用两个数据库各自的只读账号。

目标 `department` 表当前没有启停字段，因此报告中的目标部门数是全部部门数；旧部门数只统计启用且未删除记录。部门迁移前需要先决定是否给目标部门补充状态字段。

审核映射清单后先校验：

```powershell
go run ./cmd/identity-import -f configs/identity-import.yaml -manifest configs/identity-mapping.json
```

校验通过并完成审批后才允许写入：

```powershell
go run ./cmd/identity-import -f configs/identity-import.yaml -manifest configs/identity-mapping.json --apply
```

映射工具会同时核对旧用户和目标用户均为启用状态、ID 有效且用户名完全一致；整批映射在单一事务内写入。已有相同映射按幂等成功处理，任何冲突都会终止并回滚整批。

## 6. 接口与配置

过渡适配器使用内部 RPC，不新增浏览器可调用的 Token exchange 接口。配置只保存 portal-rpc 内部地址、超时和熔断参数，不在 `platform-api` 保存旧 JWT 签名密钥。

建议接口：

```go
type LegacyTokenVerifier interface {
    Verify(ctx context.Context, bearerToken string) (LegacyPrincipal, error)
}

type IdentityMapper interface {
    TargetUserID(ctx context.Context, provider string, externalUserID int64) (int64, error)
}
```

## 7. 验收标准

- 同一个旧账号访问旧 API 和新 API 时映射到同一业务人员。
- 无映射、重复映射、禁用用户、过期或已吊销 Token 全部默认拒绝。
- portal-rpc 不可用时新 API 返回认证服务不可用，不绕过验证。
- 日志、审计和缓存中不出现明文 Token、密码或 JWT 密钥。
- 前端只有一个登录会话，不要求用户重复登录。
- 切换平台原生登录前，有可回滚的数据备份、映射报告和灰度开关。

## 8. 当前完成记录

- 新增迁移 `005_identity_link.sql`。
- 新增 portal-rpc 旧 Token 验证适配器和目标身份映射仓储。
- 新平台 JWT 失败后才尝试旧 Token；未启用适配器时行为保持不变。
- 旧 Token 验证超时、映射缺失、目标用户禁用和服务错误均默认拒绝。
- 新增只读身份盘点工具及无凭据示例配置。
- 新增审核清单驱动的幂等映射导入工具；默认 dry-run，`--apply` 才写入。
- 身份桥接、中间件、权限和两个 API 已完成编译；中间件测试通过，身份测试二进制在当前 Windows Application Control 策略下被阻止执行，待环境放行后复测。
