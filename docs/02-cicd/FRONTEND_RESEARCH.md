# AxiOps CI/CD 前端调研结论

更新时间：2026-09-07

## 结论先行

Zadig 架构文档明确把正式前端称为 `zadig-portal`，但当前公开的 Zadig 主仓库只包含 Go 后端、领域模型、工作流和示例项目，没有 `zadig-portal` 的源码目录。因此目前找不到可以直接复制的官方 Zadig 控制台源码；可以复用它的产品信息架构和交互语义，再用 AxiOps 现有 Vue 3 + Element Plus 技术栈重做。

当前 `axiops-web/src/views/cicd/project-workspace/index.vue` 的主要问题不是缺少按钮，而是页面边界错误：项目、服务、环境、工作流都挤在一个带多个弹窗的页面里，服务 YAML 编辑也没有成为主工作区。这就是页面看起来只有约 20% Zadig 的原因。

## 可参考的开源前端

| 项目 | GitHub | 适合借鉴的部分 | 是否直接复用 |
| --- | --- | --- | --- |
| Argo CD | `https://github.com/argoproj/argo-cd/tree/master/ui` | 应用详情、资源树、同步 Diff、Revision、实时状态 | 不直接复用，提取交互和信息密度 |
| VelaUX | `https://github.com/kubevela/velaux` | 应用/环境/工作流模型、插件化页面、应用拓扑 | 不直接复用，模型和页面布局可参考 |
| Headlamp | `https://github.com/headlamp-k8s/headlamp` | Kubernetes 资源浏览、Namespace/集群切换、资源详情 | 不直接复用，适合参考资源详情和导航 |
| KubeSphere Console | `https://github.com/kubesphere/console` | 企业级集群、项目、工作负载和多租户导航 | 不直接复用，适合参考大规模列表和权限视图 |
| Devtron Dashboard | `https://github.com/devtron-labs/dashboard` | CI 构建、CD 发布、环境和应用详情 | 先核对目标版本许可证，再选择性参考 |
| Jenkins Blue Ocean | `https://github.com/jenkinsci/blueocean-plugin` | 流水线阶段状态和运行记录 | 仅参考流水线状态视觉表达 |

这些项目解决的问题不同，没有一个项目可以直接作为 AxiOps 的完整前端。AxiOps 需要将 Zadig 的交付模型、Argo CD 的资源/Diff 体验、Devtron 的构建发布信息和 Headlamp 的 Kubernetes 资源浏览组合起来。

## 本地参考代码

- `reference/zadig`：后端和示例，没有正式控制台前端。
- `reference/KubeDoor`：Vue 3、Element Plus、Monaco/YAML 编辑器、表格和资源详情，可作为 AxiOps 现有前端的组件参考。
- `reference/sxdevops/frontend`：Vue 3、Element Plus、RBAC 菜单、运维工作台和观测页面，可复用其页面壳、权限过滤和运维信息密度；它不是 CI/CD 产品前端。

## 许可证边界

调研项目多使用 Apache-2.0 或 MIT，但具体版本和目录仍需在落地前核对 LICENSE。需要特别注意：本地 `reference/zadig/LICENSE` 在 Apache-2.0 之外附加了商业产品、企业功能和多租户 SaaS 等限制，不能将其当成普通 Apache-2.0 代码直接复制。AxiOps 不复制 Zadig 品牌、图片、CSS 主题或大段源码；本次只借鉴公开的交互模式和领域概念，代码继续由 AxiOps 自己实现，并在确需引用其他第三方代码时保留版权和许可证声明。

## 最终选择

AxiOps 前端继续使用 Vue 3 + TypeScript + Element Plus + Monaco/YAML 编辑器，不引入 React 或整套外部控制台。第一版以 Zadig 的“项目 -> 服务 -> 环境 -> 工作流”为骨架，使用 Argo CD 的 Diff/资源树表达方式，使用 Devtron 的构建产物和发布批次信息，使用 Headlamp 的集群/Namespace 详情方式。
