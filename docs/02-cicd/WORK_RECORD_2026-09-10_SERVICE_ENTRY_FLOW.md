# 2026-09-10 服务入口与项目工作区流程

## 本次完成

- 服务配置页左侧目录旁的三个入口已接入真实流程：
  - `手动`：打开服务编辑向导，直接维护服务目录和 YAML 文件。
  - `模板`：打开模板选择和预览流程，再进入服务编辑向导。
  - `仓库`：打开批量仓库导入，可选择代码源、仓库、分支和多个服务目录。
- 移除旧的“新建服务目录”简化弹窗，避免在项目已创建后再次出现项目配置式导航。
- 代码源在进入项目工作区时按项目加载，仓库导入使用统一代码源和仓库列表。
- 工作区继续保持白色背景、可折叠服务树和干净文本编辑区域；发布入口不放在服务配置页。

## 验证

- `npx.cmd prettier --write src/views/cicd/project-workspace/index.vue src/api/axiops/cicd.ts`
- `npx.cmd eslint src/views/cicd/project-workspace/index.vue src/views/cicd/project-workspace/components/ServiceWorkbenchPanel.vue src/components/yaml-editor-pro/index.vue src/api/axiops/cicd.ts`
- `npx.cmd vue-tsc --noEmit --pretty false`
- `npm.cmd run build:dev`（Vite 构建成功，6262 modules transformed）

以上检查通过。本次无 SQL，后端无需重启；前端刷新或重启 Vite 开发服务即可生效。

## 验收路径

登录 -> CI/CD -> 项目 -> 进入项目 -> 服务配置。

在左侧服务目录旁分别点击 `手动`、`模板`、`仓库`，确认进入对应流程；选择服务目录后展开 YAML 文件，点击文件在中央文本区编辑并保存。
