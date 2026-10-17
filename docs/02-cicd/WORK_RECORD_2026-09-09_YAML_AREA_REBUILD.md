# CI/CD 工作记录：YAML 区域重建

日期：2026-09-09

## 完成内容

- 重建项目工作区页面，恢复为 AxiOps 项目导航、服务目录和项目页签结构。
- 左侧服务目录展开 `deployment.yaml`、`service.yaml` 或 Helm 文件；选择文件后只在中央显示对应 YAML 编辑器。
- 移除工作台内部重复文件树和 Revision 主视觉，避免与服务目录重复。
- YAML 编辑器固定为浅色、白色容器、稳定最小宽度和独立滚动区域。
- 右侧保留检测到的组件、系统内置变量、自定义变量及变量校验。
- 保留 Revision 加载、变量元数据恢复、手动保存 Revision、导入/构建/环境事件入口。
- 修复工作区中阻止 Vue 解析的乱码字符串。

## 验证

- `npx.cmd eslint src/views/cicd/project-workspace/index.vue src/views/cicd/project-workspace/components/ServiceWorkbenchPanel.vue --fix` 通过。
- `npx.cmd vue-tsc --noEmit --pretty false` 定向检查未发现上述两个文件错误。

## 数据库与重启

- 本次无 SQL 变更。
- 不需要后端重启；开发服务热更新即可。生产环境需要重新构建前端静态包并刷新页面。

## 2026-09-09 Workspace resource error follow-up

- Corrected the run-history API call to use project-level build runs instead of
  treating the project ID as a workflow ID.
- Optional workspace list requests are silent on 404/permission failures, so a
  valid project renders clean empty sections rather than a resource-not-found toast.
- Project detail loading now falls back to the authorized project list when an
  older platform API does not provide the detail route.
- Hidden the redundant no-service placeholder. No SQL changes and no backend
  restart are required; refresh or restart only the frontend dev server.
- Targeted ESLint and vue-tsc checks pass. Vite dev server verified at port 3006.

## 2026-09-10 Service editor header cleanup

- Removed the duplicate service name/header from the YAML editing region.
- Added manual, template-import and repository-import actions beside the
  service catalog title. The service directory add action remains separate.
- No SQL changes or backend restart. Frontend refresh/hot reload is sufficient.
- Targeted Prettier, ESLint and vue-tsc checks pass.

## 2026-09-10 Collapsible service tree

- Added expand/collapse behavior to service directories in the left tree, with
  chevron indicators and nested YAML file entries.
- Toggling a directory does not lose the selected file; switching services still
  selects that service's first file.
- No SQL changes or backend restart. Frontend refresh/hot reload is sufficient.
- Targeted Prettier, ESLint and vue-tsc checks pass.

## 2026-09-09 Plain text editing mode

- Removed editor chrome from the project YAML editing area: title bar, toolbar,
  status bar, line numbers and syntax-highlight overlay are disabled.
- The editor is now a clean white text surface with the existing save/revision
  behavior unchanged. Import/template actions are not rendered inside it.
- No SQL changes and no backend restart. Refresh the frontend or rely on Vite
  hot reload.
- Targeted ESLint and vue-tsc checks pass.
