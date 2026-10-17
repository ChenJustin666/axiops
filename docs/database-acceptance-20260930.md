# 数据库恢复后的补测（2026-09-30）

## 已完成

| 检查 | 实测结果 |
| --- | --- |
| MySQL 与迁移 | 数据库恢复连接；已应用 041_workflow_task_retry.sql，再次 check-only 无待执行迁移。 |
| MySQL 集成测试 | workflow、build、projectenv、appsvc 的 6 组 MySQL 测试全部通过，含取消/恢复/重试、构建去重、取消后禁止入队、日志持久化、环境变量分层与隔离、环境配置同步、服务生命周期。使用独立临时库，结束后清理。 |
| 登录及业务读取 | 3 个项目、8 个服务、7 个环境、4 条工作流，共 90 项符合预期；模板分类包含 YAML、Helm、Dockerfile、构建、工作流，检查列表非空且类别一致。服务 20 尚无构建配置，返回 HTTP400/code30001，前端已有空表单分支，单独作为预期结果检查。 |
| 集群资源只读查询 | 7 个环境 × 11 类资源查询，共 77 项通过；核对每条资源的 namespace 与所选环境一致。覆盖 Deployment、StatefulSet、DaemonSet、CronJob、Job、Pod、Service、Ingress、PVC、ConfigMap、Secret 列表。未读取 Secret 正文。 |
| 前端回归 | 本轮重新执行 100 项 Node/Vue 测试，100 通过、0 失败、0 跳过。 |
| 前端修改校验 | 修改文件 ESLint 通过；两个修改的 Vue 入口经 Vite 实际转换返回 HTTP200。该结果不是浏览器点击验收。 |
| 平台 API | 新源码编译通过；平台 API 包检查通过；端口越界参数正确拒绝启动。 |

首轮 MySQL 测试有两处 `sql: no rows in result set`：测试夹具没有创建业务代码现在要求锁定的有效服务记录。已在临时库内补齐服务，保留业务检查不变，重跑通过。另给旧的两组集成测试增加 `AXIOPS_TEST_CONFIG` 读取入口，防止配置了测试文件却仍然跳过。

## 页面加载故障与当前启动方式

数据库恢复后，平台 API 仍未运行。8090 被 Code.exe 占用，HTTP 健康请求超时；没有关闭 VS Code。已给平台 API 增加 `-host`、`-port` 参数，默认仍沿用原配置；本地实例使用 `127.0.0.1:18090`。

前端 Vite 代理支持 `VITE_PLATFORM_API_TARGET` / `VITE_CICD_API_TARGET`，未设置时保持原地址。本机忽略提交的 `.env.development.local` 指向 `http://127.0.0.1:18090`。通过浏览器相同的 3006 代理路径验证登录与接口，用户访问地址保持 **http://127.0.0.1:3006/**。

- 平台 API：`bin/axiops-platform-api-delivery-fixes.exe -f configs/platform-api.yaml -host 127.0.0.1 -port 18090`，本次 PID 52780。
- CI/CD API：`bin/axiops-cicd-api-delivery-fixes.exe`，8091，本次 PID 39400。
- 前端 Vite：3006，本次 PID 3492。
- 以上 PID 仅为本次记录，重启后会变化；已同步 `local/axiops-dev-processes.json`。
- Worker 未启动；没有触发实际业务构建、部署、通知或资源清理。API/数据库验证不能替代实际任务执行验收。

## 仍未通过或未验收

1. 前端全量 `vue-tsc` 仍失败（exit=2）：1296 → **1290** 条诊断。此次修复交付页面 6 条（环境标签参数、预览空值保护），最终复查 `src/views/cicd` 的 Vue 文件无诊断，但其下仍有 41 条示例 TypeScript 片段诊断；其余分布于旧流水线、资源配置等模块。详见 `local/dev/delivery-db-types-final.log`。不能把生产打包通过或局部 ESLint 通过当作全量类型检查通过。
2. 浏览器工具返回 `apps: [] / browsers: []`，本轮没有真实点击、截图或视觉验收；此前已请求连接浏览器。
3. 真实 Job 构建/部署、Worker 进程中断恢复、多 Worker 抢占、网络故障、取消后清理及清理后日志的端到端演练仍未执行。数据库用例验证的是持久化与事务语义。
4. 固定已有产物晋级、完整配置历史/已部署对比、环境运行操作、主机批次中断/ECS 构建仍有实现缺口，见 `delivery-fixes-20260930.md`。旧的独立环境详情页仍有操作占位提示，不能宣称所有环境页的日志/操作入口均已完成。审批继续延期。

## 证据与复测

- `local/dev/delivery-db-final.log`：数据库集成最终全绿。
- `local/dev/delivery-api-acceptance.json`：90 项接口检查。
- `local/dev/delivery-runtime-acceptance.json`：上述接口 + 77 项资源查询，共 167 项符合预期。
- `local/dev/delivery-db-frontend-tests.log`：100 项前端回归。
- `local/dev/delivery-db-eslint.log`：修改文件 ESLint。
- `local/dev/delivery-db-types.log`、`local/dev/delivery-db-types-final.log`：全量类型检查前后记录。
- `local/dev/platform-api-acceptance.{out,err}.log`：本次平台 API 日志。

从仓库根目录复测数据库（需有创建/删除临时测试库的权限）：

```powershell
$env:GOCACHE = Join-Path (Get-Location) '.gocache-test'
$env:AXIOPS_TEST_CONFIG = Join-Path (Get-Location) 'configs/cicd-api.yaml'
go test -count=1 -v ./internal/cicd/workflow ./internal/cicd/build ./internal/cicd/projectenv ./internal/cicd/appsvc -run MySQL
```

本地接口验收脚本 `local/dev/check-delivery-api.cjs --runtime` 从已有本地验收配置读取登录信息，只向本地 AxiOps 登录；输出仅保留状态、计数和耗时，不保存响应中的凭据或配置正文。该脚本含当前数据的预期条件，不是可直接套用到任意环境的通用测试。
