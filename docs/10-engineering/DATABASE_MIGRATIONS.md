# 数据库迁移与发布步骤

## 1. 结论

AxiOps 从 `004_authorization_scope.sql` 开始统一由版本化迁移器管理。迁移器按编号执行 SQL，使用
MySQL `GET_LOCK` 防止多个服务并发迁移，并在 `schema_migration` 表记录文件名、SHA-256 和执行时间。

- 开发、测试：建议 `Migrations.AutoApply: true`，服务启动时自动迁移。
- 生产：建议 `Migrations.AutoApply: false`，发布任务先单独执行 `axiops-migrate`，业务服务只校验。
- 有待执行迁移、迁移失败或已执行文件被修改时，业务服务拒绝启动。

`fix_admin_password.sql` 等无数字前缀的维护脚本永远不会被自动执行。
迁移编号必须唯一；发现两个相同编号的文件时迁移器拒绝启动。

## 2. 首次初始化

迁移器不自动创建 001-003 的历史基础库。新数据库先按顺序执行：

```text
migrations/001_platform_base.sql
migrations/002_asset_management.sql
migrations/003_cicd_core.sql
```

Docker MySQL 首次创建数据卷时会通过 `/docker-entrypoint-initdb.d` 执行全部编号 SQL。已有数据卷不会
再次执行该目录，因此升级不能依赖 Docker 初始化目录。

完成基础库后运行迁移器：

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml
```

## 3. 开发和测试环境

配置：

```yaml
Migrations:
  AutoApply: true
  LockTimeoutSeconds: 30
```

启动 `axiops-platform-api` 或 `axiops-cicd-api` 时会自动执行待处理迁移。两个服务同时启动也只有
获得数据库迁移锁的实例执行，另一实例等待并复查版本。

## 4. 生产发布步骤

生产配置必须关闭业务服务自动 DDL：

```yaml
Migrations:
  AutoApply: false
  LockTimeoutSeconds: 60
```

每次发布严格按以下顺序：

1. 备份数据库并确认恢复方式可用。
2. 使用与目标应用完全相同版本的镜像或代码运行迁移器。
3. 迁移成功后执行 `check-only`。
4. 启动或滚动更新 Platform API、CI/CD API。
5. 检查服务健康状态、迁移日志和核心接口。

执行命令：

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

使用编译产物时：

```powershell
axiops-migrate.exe -f configs/platform-api.yaml
axiops-migrate.exe -f configs/platform-api.yaml -check-only
```

本地统一构建脚本会生成该工具：

```powershell
powershell.exe -ExecutionPolicy Bypass -File .\scripts\build-local-backend.ps1
.\bin\axiops-migrate.exe -f configs/platform-api.yaml -check-only
```

`check-only` 返回非零即停止发布，不得继续启动新版本服务。

当前 `deploy/docker-compose.yaml` 仍缺少其引用的业务服务 Dockerfile，不能作为已验收的生产发布方案；
其中 MySQL 初始化挂载仅用于空数据卷。生产迁移应使用发布系统中的独立 migration Job 或上述二进制。

## 5. 当前需要执行的迁移

当前 CI/CD 新增迁移为：

```text
007_cicd_project_delivery.sql
008_cicd_service_revision.sql
009_cicd_project_environment.sql
010_cicd_environment_service_override.sql
011_cicd_workflow.sql
012_cicd_permission_defaults.sql
013_cicd_workflow_run.sql
014_cicd_workflow_governance.sql
015_cicd_workflow_task_lease.sql
016_cicd_deployment_snapshot.sql
017_cicd_deployment_rollback.sql
018_cicd_build_observability.sql
019_cicd_code_source_permissions.sql
020_cicd_webhook_delivery.sql
021_cicd_notification_outbox.sql
022_cicd_host_agent.sql
023_cicd_host_artifact_integrity.sql
024_cicd_host_rolling_batch.sql
025_cicd_helm_execution.sql
026_cicd_build_governance.sql
027_cicd_build_output_verification.sql
028_access_department_workflow_scope.sql
```

推荐直接运行 `axiops-migrate`，它会先幂等补齐 004 权限范围，再按顺序执行全部待处理迁移。不要跳号执行；
016 依赖工作流运行、项目环境和不可变服务 revision，017 依赖 016 的部署快照，018 依赖 006 的构建日志表，019 依赖权限角色表，020 依赖 `code_source` 和 `credential`。

确需手工执行 SQL 时，现有库应严格按 004 至 028 的编号顺序执行。每份 SQL 会写入 `schema_migration`，服务首次校验时
补全校验和。执行完成后仍必须运行：

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

## 6. 变更规则

- 已发布的编号 SQL 不得修改；需要修复时新增下一个编号迁移。
- 每个迁移只做向前变更，禁止在应用启动过程中自动执行 DROP TABLE、DROP COLUMN 或数据清空。
- 大表 DDL、长时间数据回填必须拆为独立发布步骤，并在迁移文档说明锁表和回滚影响。
- 自动迁移失败后不得反复重启绕过；先查看失败的文件和 statement 编号，确认数据库实际状态。
- 迁移代码、SQL、配置、文档和测试必须随同一个应用版本发布。
# 本轮迁移更新

当前受管迁移已扩展到 `028_access_department_workflow_scope.sql`。用户当前数据库若停留在 023，应由同一版本的 `axiops-migrate` 一次执行 024-028，不要单独 `SOURCE 028` 或跳过中间迁移。
