# Zadig Portal 源码核对与工作台修复记录（2026-09-08）

## 结果

- 确认 `reference/zadig-portal` 是 Zadig 官方 Portal 前端源码，包含服务树、YAML/Helm 编辑、构建变量侧栏、环境详情、工作流、测试和主机应用向导。
- 确认不能直接复制：参考项目为 Vue 2 + Vuex + Element UI + Webpack，AxiOps 为 Vue 3 + Pinia + Element Plus + Vite，且 API、权限和 Revision 模型不同。
- 已新增源码分析和 AxiOps 分阶段重构计划：`ZADIG_PORTAL_ANALYSIS_2026-09-08.md`。
- 修复项目工作台刷新时全页 `v-loading` 覆盖已有内容的问题；只有首次项目未加载时显示全页加载状态，刷新期间页面保持可读。
- 为工作台根容器补充 `width: 100%` 和 `min-width: 0`，降低父容器约束导致的横向溢出风险。

## 验证

- `prettier --check src/views/cicd/project-workspace/index.vue` 通过。
- `eslint src/views/cicd/project-workspace/index.vue` 通过。
- Vite 生产构建通过（6260 modules）。仓库该页面历史样式规则尚未全量整理，单独运行 Stylelint 仍会报告既有的大量属性顺序/空行问题；本次未进行自动重排，避免引入无关视觉差异。
- 参考源码静态核对完成：项目向导、服务编辑器、环境页面、工作流列表和 API 路由均已定位。

## 数据库与重启

- 无 SQL 变更，不需要执行迁移。
- 仅前端文件变化：开发服务器可热更新；若当前运行的是生产静态包，重新执行前端构建并刷新部署文件。
- 本次没有后端代码变化，不需要重启后端服务。

## 下一步

按分析文档的 P0 -> P1 顺序，先把工作台改为稳定的 Zadig 信息架构和响应式三栏，再完善服务三步向导、Helm 文件树、环境矩阵和工作流运行详情。每个阶段单独记录、测试和截图验收。
