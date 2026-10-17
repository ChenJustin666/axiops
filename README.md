# AxiOps

AxiOps 是 Kubernetes 管理与 CI/CD 平台，采用 Go 后端和 Vue 3 前端。

- 仓库：<https://github.com/ChenJustin666/axiops>
- Go 模块：`github.com/ChenJustin666/axiops`
- `cmd/`、`internal/`：AxiOps API、工作流 Worker、迁移器与业务模块。
- `application/`、`common/`：已有集群管理、身份认证及公共能力。
- `axiops-web/`：前端完整源码，作为本仓库普通目录维护。
- `configs/`、`deploy/`、`migrations/`：配置示例、部署文件和有序迁移。

## 本地开发

需要 Go 1.25.5 或更新版本、Node.js 20.19+、pnpm 9、MySQL 8 与 Redis。

```powershell
git clone https://github.com/ChenJustin666/axiops.git
cd axiops
powershell -ExecutionPolicy Bypass -File scripts/setup-local-config.ps1
```

修改生成的 `configs/*.yaml` 中的数据库、Redis、JWT 及所需集群配置。
示例中的 `REPLACE_ME` 必须换成自己的值；JWT 密钥应随机生成，相关 API 使用一致的密钥。
实际密码、凭据加密密钥和私钥留在本地或外部密钥管理系统中，不提交 Git。

```powershell
cd axiops-web
corepack pnpm install --frozen-lockfile
cd ..
powershell -ExecutionPolicy Bypass -File scripts/start-dev.ps1
```

现有安装可继续使用原有本地配置、数据库和加密密钥；模块改名不要求迁移业务数据。
`scripts/start-local-backend.ps1` 启动的已有 RPC/API 使用 `config/local/`，应另行配置这些服务。
新安装的身份认证、初始管理员和数据库初始化请按部署文档配置；仓库不携带现有环境的用户或业务数据。
`sql/legacy-schema.sql` 仅含已有系统表结构，不含账号和运行数据。

## 验证

```powershell
go mod tidy -diff
go test ./...
go build ./...
cd axiops-web
node --test tests/*.test.mjs
corepack pnpm build
```

数据库集成用例需要配置 `AXIOPS_TEST_CONFIG`，并使用有权创建、删除临时测试数据库的测试账号。
不要把它指向不允许集成测试访问的数据库。

## 操作说明

- [Helm 多环境 values](docs/02-cicd/HELM_MULTI_ENV_VALUES_2026-10-17.md)
- [Git 原生 Chart](docs/02-cicd/HELM_GIT_CHART_2026-10-17.md)
- [Java / Nacos / SkyWalking 模板](docs/02-cicd/JAVA_SKYWALKING_TEMPLATE_2026-10-17.md)
- [仓库迁移与验收](docs/REPOSITORY_MIGRATION_2026-10-17.md)

## 来源和许可

本项目基于 [Kube-Nova](https://github.com/yanshicheng/kube-nova) 和其前端继续开发。
保留原作者版权、[Apache-2.0 许可证](LICENSE)、[NOTICE](NOTICE) 及前端对应声明。
原项目介绍保存在 [上游说明](docs/upstream/KUBE_NOVA_README.md)。
