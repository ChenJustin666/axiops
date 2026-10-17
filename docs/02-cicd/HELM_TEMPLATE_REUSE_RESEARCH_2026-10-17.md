# Helm 模板复用调研与 AxiOps 结构建议

调研日期：2026-10-17。依据本仓库 Zadig 参考源码、Helm 3 官方文档、Argo CD 与 Flux 官方文档。本文保留调研时的方案与差距分析；后续已实现固定版本、使用方查询、更新提示及单环境升级，准确交付/验收状态以 [共享模板教程](HELM_SHARED_TEMPLATES_2026-10-17.md) 和 [Helm 交付状态](HELM_DELIVERY_2026-10-17.md) 为准。

## 结论

Helm 没有“一份 Chart 只能对应一个服务”的限制。建议 AxiOps 默认采用：**一份可复用、固定版本的应用 Chart，多服务复用；每个服务在每个环境有独立 values 和独立 Release。**

Chart 的复用单位与发布/回滚单位不同。前者按相似部署结构共享，后者按实际独立交付边界划分。不是所有服务必须共用一个万能模板，也不是所有服务必须手写一份不同 Chart。

| 概念 | 作用 | 示例 |
| --- | --- | --- |
| Chart | 部署资源的版本化定义，可以复用 | web-application@1.2.0 |
| AxiOps 服务 | 独立构建、选镜像、发布和回滚的业务单位 | card-scout-server、card-scout-gateway |
| 环境服务配置 | 该服务在目标环境选择的 Chart 版本和 values | gateway 的 qa 配置、gateway 的 prod 配置 |
| Helm Release | Chart 在指定集群/Namespace 的安装实例 | qa 中的 gateway、prod 中的 gateway |
| Kubernetes Service | Chart 可以生成的一种网络资源 | ClusterIP Service；不等于上述业务服务 |

示例：server 和 gateway 都使用 web-application@1.2.0。它们可以有不同镜像、端口、启动方式、路由和资源配置。QA/生产分别保存配置并独立发布；gateway 回滚不应连带 server 回滚。Release 名称可以相同，但必须由集群、Namespace 与 Release 名的组合明确识别，平台还需校验业务归属。

## 调研证据

### 本地 Zadig 参考源码

仓库确有 reference/zadig 和 reference/zadig-portal。本次核对的是本地参考快照，不宣称等同于 Zadig 当前所有商业版本能力。

- `reference/zadig/pkg/microservice/aslan/core/service/service/types.go` 的 CreateFromChartTemplate 有 TemplateName、ValuesYAML、Variables，说明模板来源和服务参数分离。
- 同目录 `helm.go` 的 CreateOrUpdateHelmServiceFromChartTemplate / createOrUpdateHelmServiceFromChartTemplate（约 671 行）从模板加载 Chart，合并默认与服务 values，复制为服务目录并生成服务 Revision，保存模板名称等来源信息。
- `helm.go` 的 CreateOrUpdateBulkHelmServiceFromTemplate（约 1162 行）支持从同一模板批量创建服务。
- `template.go` 的 syncServicesFromChartTemplate（约 208 行）筛选模板来源和 AutoSync，匹配模板名再更新服务；所以不是“模板只给一个服务用”，也不是所有服务无条件同步。
- 前端 `reference/zadig-portal/src/api/index.js` 有独立 Chart 模板管理 API 与 Helm 服务创建 API。

因此，Zadig 参考实现也是“模板可复用 + 服务独立版本”，内部复制快照并不意味着业务必须维护多份不相关模板。其自动同步/自动部署策略不直接照搬到 AxiOps；AxiOps 用户明确要求保存配置不自动发布。

### Helm 官方

[Using Helm（Helm 3）](https://helm.sh/docs/v3/intro/using_helm/) 将安装参数区分为 Release 名与 Chart 名；upgrade/rollback 操作针对 Release。相同 Chart 可以按不同 Release 名和 values 部署多次。

[Charts](https://helm.sh/docs/v3/topics/charts/) 支持依赖与子 Chart，可把一组组件组成一个应用包。因此一个 Chart 也可以包含多个 Deployment、Service 等资源；这是一种组合交付选择，不是微服务必须采用的结构。

[Library Charts（Helm 3）](https://helm.sh/docs/v3/topics/library_charts/) 用于复用 helper/templates，library 类型不能直接安装。AxiOps 普通用户应选择可安装的 application Chart；共享模板并不要求用户先创建 library Chart。复杂 Chart 的公共代码才适合在内部用 library Chart 提取。

### Argo CD

[Argo CD Helm 文档](https://argo-cd.readthedocs.io/en/stable/user-guide/helm/) 的 Application 分别配置 Chart 来源/版本、values 与目标环境，多个 Application 可引用同一 Chart。

注意它使用 Helm 渲染资源，生命周期由 Argo CD 管理，不使用 Helm install/upgrade 作为发布生命周期。参考其来源/配置分离结构即可，不能直接宣称它具有 AxiOps 的 Helm Release 回滚语义。

### Flux

[Flux HelmRelease 文档](https://fluxcd.io/flux/components/helm/helmreleases/) 区分 Chart 来源引用、Release 名、目标 Namespace、values；支持从 Secret/ConfigMap 获取 values。多个 HelmRelease 可复用 Chart 来源并分别管理配置。

它通过控制器持续协调发布，和 AxiOps 用户主动执行工作流不同；不照搬“values 改完自动协调发布”的策略。

## 默认方案与特殊情况

| 方案 | 优点 | 成本与适用范围 |
| --- | --- | --- |
| 通用应用 Chart + 独立服务/环境 Release（推荐默认） | 模板维护集中，镜像/环境参数独立，按服务发布回滚 | Chart 需参数化，资源命名与选择器需避免冲突 |
| 每个服务专用 Chart | 完全定制，适合特殊资源结构 | 大量重复维护；少数特殊服务采用，不作为所有服务的默认要求 |
| 一个组合 Chart/Release 部署多个业务服务 | 整套应用整体交付方便 | 升级/回滚范围是整个 Release；需要多镜像映射与组合版本管理，仅适合紧密耦合交付 |

Chart 应按部署结构建立少量类型，例如普通 Web 应用、带配置中心的 Java 应用、定时任务、StatefulSet 应用。不要把数据库、普通 Java 服务、批处理强塞到一份参数极多的万能 Chart。

一个组合 Release 即使能通过 values 开关/修改部分组件，Helm 历史和 rollback 仍以 Release 为单位。不能向用户承诺其中一个业务服务可用标准 Helm rollback 完全独立回滚。

## AxiOps 应有的操作方式

1. 模板库维护通用 Helm Chart 和不可变发布版本。普通用户创建服务时选模板/版本，填写服务与目标环境参数；无需再次维护 Chart 文件。
2. 每个环境的服务配置明确显示模板名称、当前固定版本、镜像与 values。环境参数保留当前独立编辑/预览/保存方式，不再额外增加用户必须理解的多层继承。
3. 模板更新只产生新版本，提示受影响服务“有可用更新”；不静默覆盖现有服务，不自动发布生产。
4. 升级时选择服务及目标环境，合并新版本默认值与目标已有配置，校验 schema/必填值、镜像映射及渲染差异。有字段删除、类型变更或冲突时要求处理；不存在通用的无损自动合并保证。预览保存后再执行发布。
5. 特殊服务可明确“派生为自定义 Chart”。之后不假装仍能无冲突跟随通用模板更新；来源记录保留，更新要单独比较。
6. 工作流中可以多选独立服务并调整顺序，不需要为了批量发布把它们合并为一个 Release。多服务工作流也不是跨服务的原子事务，必须逐服务显示结果与回滚范围。

values.schema.json 可作为参数约束依据。图形参数表单需额外的显示分组/组件约定，不能假设任意第三方 Chart 都能自动变成完善表单；原生 values 编辑与真实渲染预览始终保留。

## 当前实现与缺口

已实现：多个服务可分别从同一 Helm 模板创建；Chart 内容进入服务不可变 Revision；每个环境独立 values；服务按 Release 发布和回滚；模板保存不自动影响已创建服务。来源版本在创建 Revision 的 source_commit 中记录为 `template:<id>@<revision>`。

目前属于“从共享模板生成独立快照”，不是持续绑定最新版。尚未形成完整的模板使用方列表、已发布不可变模板版本选择、更新提示、逐服务/逐环境升级预览和自定义派生管理；本次调研不把这些列为已完成。

当前镜像注入主要面向每个 AxiOps 服务的一份构建产物，Chart 可以包含多个资源，但“一个组合 Release 对应多个独立业务服务并各自选镜像”还没有完整产品闭环。继续完成共享模板版本管理更符合现有使用需求，无需先扩展组合发布类型。
