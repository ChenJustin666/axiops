# CI/CD 工作记录：UI 与后端收口（2026-09-08）

## 本次完成

- 服务工作台继续保持 Zadig 风格三栏布局：服务目录、服务×环境矩阵、Revision/资源工作区。
- 项目概览增加交付就绪度检查项，直接提示服务、环境、工作流和构建产物是否满足发布条件。
- 工作流运行详情增加阶段进度、当前节点、可读节点名称和构建详情入口，便于定位失败原因。
- Revision 工作区支持复制 commit SHA，并可直接打开代码仓库提交页面。
- 服务停用采用可恢复的 `inactive` 状态，保留 Revision、环境引用、运行记录和审计数据；停用服务不再允许新 Revision 或构建。
- 后端补充项目、服务、集群、Revision 等输入校验，并统一 JSON/SSE 响应的 UTF-8 charset。

## 验证

- `go test -count=1 ./internal/cicd/... ./internal/platform/cluster ./internal/platform/project ./internal/shared/response`
- `go vet ./internal/cicd/... ./internal/platform/cluster ./internal/platform/project ./internal/shared/response ./cmd/cicd-api ./cmd/platform-api ./cmd/workflow-worker`
- CI/CD 定向 ESLint、Prettier 通过。
- Vite 生产构建通过（6260 modules）。仓库全量 `vue-tsc` 仍有既有非 CI/CD 模块类型问题，本次修改文件未新增筛选错误。

> Windows 本机默认 Go 构建缓存目录受应用控制策略限制时，可将 `GOCACHE` 指向仓库内临时目录后重试，例如 `$env:GOCACHE='E:\\axiops\\.gocache-test2'`；本轮测试已使用该方式通过。

## 数据库、重启与发布

- 本轮没有新增或修改 SQL，不需要执行迁移；当前迁移版本仍为 `028_access_department_workflow_scope.sql`。
- Platform API 的项目/集群校验和 CI/CD API 均有代码变化，建议统一重新构建并启动：

  ```powershell
  cd E:\axiops
  .\start.cmd
  ```

- Workflow Worker 本轮自身没有代码变化，不要求单独重启；统一启动脚本会同时拉起完整本地链路。
- 使用生产静态前端时，需要将 `axiops-web/dist` 发布到静态服务器并刷新浏览器；开发模式可直接热更新。

## 手工验收入口

1. 登录 AxiOps，进入 CI/CD 项目工作台。
2. 检查项目概览的交付就绪度、服务×环境矩阵和环境晋升链。
3. 打开服务工作台，查看 YAML/Helm、Revision Diff、commit 链接和环境引用。
4. 运行工作流，确认阶段进度、节点名称、构建日志和失败原因可见。
5. 使用测试集群完成 YAML 环境预览、发布和回滚；生产环境只验证审批门禁，不直接发布。
