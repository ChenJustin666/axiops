# CI/CD 前端调研工作记录

日期：2026-09-07

## 已完成

- 核查 `reference/zadig` 和 Zadig 当前主仓库结构。
- 确认正式前端组件名为 `zadig-portal`，但其源码不在当前公开主仓库中。
- 核查本地 `reference/KubeDoor`、`reference/sxdevops/frontend` 的技术栈和可复用范围。
- 整理 Argo CD、VelaUX、Headlamp、KubeSphere Console、Devtron Dashboard 和 Jenkins Blue Ocean 的参考重点。
- 明确 Zadig 附加许可限制，AxiOps 只参考交互和领域模型，不复制受限源码。
- 输出 `FRONTEND_RESEARCH.md` 和 `UI_REBUILD_PLAN.md`。

## 验证

- 两份新文档均以 UTF-8 写入并重新读取验证。
- 本轮未修改前端运行代码、后端代码或数据库迁移。
- 本轮不需要重启服务，不需要执行 SQL。

## 下一步

先制作“服务与环境”页面级高保真骨架，确认布局和核心操作路径后，再接入现有 API。旧弹窗只保留为回归入口，不继续扩展。

