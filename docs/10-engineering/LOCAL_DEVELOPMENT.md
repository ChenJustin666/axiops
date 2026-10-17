# AxiOps 本地开发与启动

## 当前运行架构

默认开发链路只启动 AxiOps 新服务，不启动 `application/` 下的旧 Portal、Manager 和 DevOps 服务。

| 进程 | 端口 | 职责 |
|---|---:|---|
| `axiops-platform-api` | 8090 | 登录、用户、部门、角色、菜单、授权、项目、集群、凭据 |
| `axiops-cicd-api` | 8091 | 服务、构建、环境、工作流、发布、通知、主机 Agent |
| `axiops-workflow-worker` | - | 构建和部署任务消费、状态与日志回传 |
| `axiops-web` | 3006 | AxiOps Vue 前端 |

数据库迁移由独立的 `axiops-migrate` 在业务进程启动前统一执行。三个业务进程均配置
`Migrations.AutoApply: false`，避免并发启动时执行 DDL。

## 一键启动与停止

在仓库根目录执行：

```powershell
cd E:\axiops
.\start.cmd
```

访问：

```text
http://127.0.0.1:3006
```

本地初始化管理员：

```text
用户名：admin
密码：REPLACE_ME
```

停止全部由脚本启动的进程：

```powershell
.\stop.cmd
```

仅启动后端：

```powershell
.\start.cmd -NoWeb
```

已编译且只需重启时：

```powershell
.\start.cmd -SkipBuild
```

## 启动脚本执行顺序

1. 初始化本地凭据加密密钥 `local/credential-encryption.key`；已存在时保持不变。
2. 编译迁移器、Platform API、CI/CD API 和 Worker。
3. 执行全部待处理数据库迁移。
4. 启动三个后端进程和前端。
5. 检查 8090、8091 健康接口和 3006 首页。

密钥文件已被 Git 忽略。不要删除或随意轮换；已有加密凭据依赖同一密钥解密。生产环境必须由
Secret 管理系统注入，不能复用本地文件。

## 数据库检查

只检查版本、不执行 SQL：

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

当前代码期望：

```text
current=028_access_department_workflow_scope.sql applied=[]
```

如需手工执行 SQL，必须按 `migrations/README.md` 的编号顺序执行，最后仍运行 `-check-only`。
生产发布必须先备份，再由独立 migration Job 执行，成功后才启动业务服务。

## API 路由

开发环境前端统一使用 `/api`，Vite 按服务分流：

```text
/api/platform/v1/* -> http://127.0.0.1:8090/api/v1/*
/api/cicd/v1/*     -> http://127.0.0.1:8091/api/v1/*
```

生产网关应提供等价路由。前端业务代码使用 `/platform/v1/*` 和 `/cicd/v1/*`，不要重新混用旧
`/portal/v1` 或 `/devops/v1` 登录链路。

## 日志与排障

运行日志位于：

```text
local/dev/platform-api.out.log
local/dev/platform-api.err.log
local/dev/cicd-api.out.log
local/dev/cicd-api.err.log
local/dev/workflow-worker.out.log
local/dev/workflow-worker.err.log
local/dev/web.out.log
local/dev/web.err.log
```

健康检查：

```powershell
Invoke-RestMethod http://127.0.0.1:8090/api/v1/system/health
Invoke-RestMethod http://127.0.0.1:8091/api/v1/system/health
Invoke-WebRequest http://127.0.0.1:3006 -UseBasicParsing
```

常见启动失败：

- `CredentialEncryptionKey must be configured`：先运行 `scripts/init-local-secrets.ps1`，统一启动脚本会自动处理。
- Windows 应用程序控制策略拦截 `.exe` 或 Vite `esbuild` 子进程时，启动脚本会通过 `cmd.exe` 启动后端，并使用 Vite `--configLoader runner`；不要改回 PowerShell 直接启动二进制。
- `database baseline is incomplete`：新库先执行 `001-003`，再运行迁移器；不能跳号执行后续 SQL。
- `database migrations ready ... applied=[]`：迁移已完整，无需重复 `SOURCE`。
- 端口占用：先执行 `stop.cmd`，确认没有手工启动的同名进程后再启动。

## 当前迁移策略

`application/` 和旧前端页面暂时作为参考源码保留，但默认路由不再注册它们。删除顺序固定为：先完成
新链路端到端验收，再核对数据与调用方，最后删除旧入口、接口和表；不能在验收前直接清理历史代码。
