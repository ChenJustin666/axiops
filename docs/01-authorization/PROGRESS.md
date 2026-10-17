# 权限模块完成进度

## 2026-08-10：后端权限闭环

完成内容：

- 统一授权内核支持默认拒绝、显式 deny 优先、动作兼容和资源范围匹配。
- `role_binding` 支持用户/直属部门主体、资源范围和临时授权。
- 用户、角色、项目、项目成员、凭据、集群、服务、构建、构建日志和构建配置接口接入授权。
- 列表查询使用参数化 SQL 范围过滤，分页总数不会泄漏无权资源数量。
- 新增角色绑定 CRUD、当前用户菜单、当前权限和菜单列表 API。
- 角色增加 `binding_scope`，严格区分 `platform` 和 `scoped`；角色范围变更存在不兼容绑定时拒绝修改。
- 平台管理员不能分配、修改或移除 `super_admin`，也不能用自定义角色绕过 `*`、`system_role` 保护；只有 `system_role:assign/update` 可以执行受保护操作。
- 项目创建按 `project_admin` 角色编码查询，不依赖不稳定的固定 ID，并在同一事务创建项目和所有者成员关系。
- 用户和初始平台角色在同一事务创建；角色、权限和菜单在同一事务创建或更新。
- 当前菜单自动补齐父菜单，禁用或删除用户不再获得策略或菜单。
- 授权绑定支持 `clear_expires_at` 显式清除过期时间。
- 认证后的写请求记录审计日志；不记录请求体，避免敏感字段泄漏。

接口清单：

| 接口 | 用途 |
|---|---|
| `GET /api/v1/auth/menus` | 当前用户可见菜单 |
| `GET /api/v1/auth/access` | 当前用户生效策略 |
| `GET /api/v1/access/menus` | 授权管理员查看全部菜单 |
| `GET /api/v1/access/bindings` | 查询角色绑定 |
| `POST /api/v1/access/bindings` | 创建资源范围绑定 |
| `PUT /api/v1/access/bindings/:id` | 启停、设置或清除过期时间 |
| `DELETE /api/v1/access/bindings/:id` | 删除绑定 |

测试记录：

| 检查项 | 结果 | 说明 |
|---|---|---|
| `gofmt` | 通过 | 本模块相关 Go 文件已格式化 |
| 授权内核单测 | 通过 | deny 优先、项目/服务/构建范围、集群 SQL、环境 ID |
| 授权管理与角色单测 | 通过 | 主体、范围、平台角色防下放、权限项校验 |
| 审计辅助单测 | 通过 | 资源识别和客户端地址 |
| platform-api 编译 | 通过 | 用户、角色、项目、集群、凭据和授权模块 |
| cicd-api 编译 | 通过 | 服务、构建、发布骨架和 webhook |

验证命令：

```powershell
$env:GOCACHE='E:\axiops\.gocache'
$env:GOTMPDIR='E:\axiops\.gotmp'
go test ./internal/shared/authz ./internal/shared/middleware ./internal/platform/access ./internal/platform/role ./internal/platform/user ./internal/platform/project ./internal/platform/cluster ./internal/platform/credential ./internal/cicd/... ./cmd/platform-api ./cmd/cicd-api
```

当前结果：以上命令通过。

未完成项：

- 部门树递归授权未开启，当前仅匹配直属部门。
- 凭据和 kubeconfig 仍需静态加密与密钥轮换方案。
- 迁移 `004_authorization_scope.sql` 是一次性版本迁移，由迁移记录保证只执行一次，不作为可重复初始化脚本使用。

2026-08-18 范围确认：权限前端纳入本次平台交付，需在 CI/CD 前端重构阶段同步完成用户、部门、角色、菜单、项目、服务、环境、集群、构建和工作流范围授权页面，并通过只读与越权 Playwright 用例验收。

## 2026-08-18：权限中心前端与工作流范围

完成内容：

- 新增权限中心，包含资源授权、角色策略、用户管理和部门管理四个工作区。
- 角色编辑支持常用权限矩阵、显式 `deny`、菜单树和范围角色创建；系统角色继续受 `system_role` 保护。
- 资源授权支持用户或直属部门，下钻选择项目、服务、环境、构建、工作流和集群，并支持有效期、启停和删除。
- 新增 `/api/v1/departments` 查询、创建、更新和删除；存在子部门或直属用户时拒绝删除，更新时拒绝循环父级。
- 新用户管理直接使用 `/api/v1/users`，支持直属部门、平台角色、启停和删除，不再依赖旧 `/portal/v1` 权限入口。
- 授权内核新增 `workflow` 范围；工作流列表、详情、更新、执行、运行查看、取消和审批均传入工作流 ID 二次鉴权。
- 菜单处理同时使用 `/api/v1/auth/menus` 和 `/api/v1/auth/access`；新 CI/CD 与权限中心不再使用 `R_SUPER/R_ADMIN` 作为权限边界。
- 新增 `028_access_department_workflow_scope.sql`，为平台管理员补齐部门管理策略；本轮未连接业务数据库、未执行 SQL。

验证结果：

| 检查 | 结果 |
|---|---|
| `go test ./internal/cicd/... ./internal/platform/... ./internal/shared/...` | 通过 |
| `go vet ./internal/cicd/... ./internal/platform/... ./internal/shared/...` | 通过 |
| 权限/CICD 新前端文件 ESLint | 通过 |
| 权限/CICD 新前端 TypeScript 过滤检查 | 通过 |
| `corepack pnpm run build` | 通过，6235 个模块完成生产构建 |
| 桌面/移动浏览器截图 | 阻塞，当前执行环境没有可连接的浏览器实例 |

下一步：执行 024-028 迁移后，以管理员、开发只读、测试和运维账号完成真实登录态越权与菜单 E2E。

## 2026-08-19：AxiOps 原生登录与菜单基线

完成内容：

- 前端登录从旧 `/portal/v1/auth/login` 切换到 `/platform/v1/auth/login`，使用 AxiOps JWT 和原生用户结构。
- 路由守卫改为读取 `/platform/v1/auth/current-user`、`auth/menus` 和 `auth/access`，不再依赖旧平台、项目上下文接口。
- 新前端只注册 CI/CD 与权限中心路由；旧页面源码暂留，但不进入 AxiOps 菜单。
- Platform 与 CI/CD 请求分别使用 `/platform/v1`、`/cicd/v1`，开发代理已验证可正确转发到 8090 和 8091。
- 品牌标题和登录页统一为 AxiOps，删除旧演示账号、社区弹窗、滑块验证和 Base64 密码提交。
- 数据库迁移已完成到 `028_access_department_workflow_scope.sql`，`check-only` 返回 `applied=[]`。

验证结果：

| 检查 | 结果 |
|---|---|
| 变更文件 Prettier、ESLint、Stylelint | 通过 |
| 变更文件 TypeScript 过滤检查 | 通过 |
| 前端生产构建 | 通过，6239 个模块 |
| 前端代理登录与当前用户 | 通过，`admin` 登录返回 AxiOps JWT |
| 菜单与策略接口 | 通过，管理员返回 10 个菜单码和通配策略 |
| Platform 项目与 CI/CD 构建列表代理 | 通过，当前验收库均为空数据 |
| 浏览器视觉与点击 E2E | 阻塞，浏览器控制组件运行路径异常，未标记通过 |

下一步：在权限中心创建开发只读、测试和运维验收账号，验证菜单隔离、资源范围和越权拒绝。

### 路由资源 ID 回归修复

- 真实 CRUD 冒烟发现 go-zero 将 `:id` 保存于 `pathvar`，但现有 Handler 与审计使用 Go 1.22
  `Request.PathValue`，导致所有详情、更新、删除请求把资源 ID 解析为 0。
- 新增全局 `SyncPathValues` 中间件，在 Platform API 和 CI/CD API 路由入口统一同步参数；没有逐个
  Handler 添加重复兼容代码。
- 新增中间件单元测试，覆盖多参数和无参数请求。
- 通过前端代理完成角色、部门、用户的创建、详情、更新、删除并清理测试数据；最终恢复为 7 个
  内置角色、1 个管理员、0 个部门。
- CI/CD 服务、工作流和 Platform 项目详情使用不存在的正整数回归后，均返回对应“资源不存在”，
  不再返回“无效 ID”。
- Platform、CI/CD、共享包及命令的 `go vet` 通过；全量 Go 测试除 `webhook`、`jwtx` 测试二进制
  被 Windows Application Control 拦截外均通过，改用仓库内 `GOTMPDIR` 后仍被系统策略阻止，
  因此这两包不标记为本轮通过。

## 2026-08-10：身份收敛基础

完成内容：

- 新增 `identity_link` 数据模型和 `005_identity_link.sql`。
- 新增 portal-rpc 旧 Token 验证器、目标身份映射和可选认证回退；配置默认关闭。
- 映射只接受 portal-rpc 已验签、已检查 Redis 吊销和账号状态的 Token。
- 映射缺失、目标用户禁用、认证服务不可用全部默认拒绝。
- 新增只读 `identity-inventory` 工具，输出用户/部门规模、已有映射、未映射用户和同名账号样本，不写数据库。
- 新增 `identity-import` 工具，默认仅验证审核清单；显式 `--apply` 后才在事务中写入，重复相同映射幂等成功，冲突整批回滚。

测试记录：

| 检查项 | 结果 | 说明 |
|---|---|---|
| 身份解析单测 | 阻塞 | 测试代码已编译；Windows Application Control 阻止 `identity.test.exe` 执行 |
| 认证中间件单测 | 通过 | 新 Token 失败后才尝试可信旧认证 |
| 盘点工具和 platform-api 编译 | 通过 | 默认关闭桥接，不改变当前运行行为 |
| 映射导入校验单测与编译 | 阻塞 | 测试代码已编译；同一执行策略阻止 identity 测试二进制 |

下一任务：使用只读账号在目标环境生成身份盘点报告，审核同名账号和部门映射后实现幂等导入；真实映射完成前保持 `LegacyIdentity.Enabled=false`，不能把旧前端直接指向新 `/api/v1` 接口。
