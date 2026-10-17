# CI/CD 前端工作记录：项目工作台重构

日期：2026-09-07

## 完成内容

- 重写 `axiops-web/src/views/cicd/project-workspace/index.vue`，将项目内默认入口调整为服务与环境工作台。
- 新增 `ServiceWorkbenchPanel.vue`，提供服务元数据、Revision 列表、定义查看/编辑、Diff、环境引用和发布入口。
- 保留既有服务导入、批量同步、构建配置、环境配置、工作流编排和运行详情组件，并从新工作台统一调用。
- 增加运行记录和构建制品视图，展示环境、触发方式、Commit、镜像地址和 Digest。
- 增加服务搜索、状态筛选、环境筛选和矩阵批量选择。
- 统一 UTF-8 文案；本轮检查 CI/CD 目录未发现历史乱码序列。

## 验证结果

- Prettier：通过。
- 定向 ESLint：通过。
- Vite production build：通过，6263 个模块完成转换。
- 定向 `vue-tsc`：新增工作台文件无错误。
- 全量 `vue-tsc`：仍被仓库其他历史模块错误阻断，详见命令输出；没有发现本轮文件相关错误。

## 数据库和重启

- 未新增或修改 SQL，数据库无需操作。
- 只修改前端，开发服务器刷新即可；不需要重启 Platform API、CI/CD API 或 Workflow Worker。

## 2026-09-08 收尾补充

- 修复项目工作台源码中文乱码，所有用户可见文案恢复为 UTF-8 中文；同时检查 CI/CD、集群和权限前端目录，没有发现替换字符乱码。
- 对项目入口和服务向导执行 Prettier，CI/CD 相关目录定向 ESLint 通过。
- Vite 生产构建成功，6263 个模块转换完成。
- 定向 Go 工作流测试及 `go vet` 通过。
- 临时构建产物解析工具已删除，未新增数据库迁移。

## 体验验收要点

1. 进入项目后默认看到“服务与环境”三栏工作台，而不是多个弹窗。
2. 选择服务可以在右侧查看 Revision 和 YAML/Helm/主机定义。
3. 选择历史 Revision 后切换到 `Revision Diff` 可并排查看差异。
4. 编辑定义保存后生成新的 Revision，原版本仍可查看。
5. 矩阵环境单元格显示目标集群/Namespace 和生产审批状态。
6. 多选服务后可以进入发布批次；无权限的操作按钮不显示。

## 本轮最终验收

1. 使用 `cmd /c node_modules\\.bin\\prettier.cmd --check` 检查 CI/CD、集群和权限前端目录，应输出 `All matched files use Prettier code style!`。
2. 使用 `cmd /c node_modules\\.bin\\eslint.cmd src/views/cicd src/views/clusters src/views/system/access src/api/axiops src/composables`，应无输出并返回 0。
3. 使用 `cmd /c node_modules\\.bin\\vite.cmd build`，应看到 `✓ built`。
4. 使用 `go test ./internal/cicd/workflow ./cmd/workflow-worker` 和 `go vet ./internal/cicd/workflow ./cmd/workflow-worker`，均应返回 0。
