# AxiOps 数据库迁移

所有模块的数据库变更统一放在本目录，命名格式：

```text
NNN_模块_功能.sql
```

例如：

```text
007_cicd_project_delivery.sql
008_cicd_service_revision.sql
009_cicd_project_environment.sql
010_cicd_environment_service_override.sql
```

规则：

1. 编号全局递增且唯一，不按模块重新编号。
2. 已发布 SQL 不修改；修复使用新编号。
3. 数字编号 SQL 会嵌入 `axiops-migrate`，支持自动执行和手工执行。
4. 无数字前缀的维护脚本不会自动执行。
5. 每份受管迁移最后写入 `schema_migration`，手工执行后也能被启动校验识别。
6. 禁止在启动自动迁移中加入 DROP TABLE、DROP COLUMN、TRUNCATE 或无条件数据删除。

手工执行时必须严格按编号顺序。推荐使用统一命令，它会自动加锁、排序和校验：

```powershell
go run ./cmd/axiops-migrate -f configs/platform-api.yaml
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

生产发布步骤见 [数据库迁移与发布步骤](../docs/10-engineering/DATABASE_MIGRATIONS.md)。
# 当前 CI/CD 迁移序列

当前受管序列为 `001` 至 `052`，完整清单见 [迁移顺序](../docs/02-cicd/MIGRATION_ORDER.md)。工具根据 schema_migration 执行缺失版本，不要只挑最后一份 SQL。

`052_delivery_template_versions.sql` 为共享交付模板增加不可变历史。执行后配套更新 CI/CD API、Workflow Worker 与前端；启动时仅回填模板的确切当前版本，无法恢复迁移前被覆盖的历史。迁移只建表，不修改现有环境绑定或触发部署。
