# 可观测 / 监控模块设计方案（复用 KubeDoor 能力）

> 目标：把 `reference/KubeDoor` 的可观测与管控能力，以 **AxiOps 自己的技术栈与设计语言**重建，
> 落地为独立的 observability 模块；**不改动任何 CICD 代码**。
> 对齐 `docs/03-monitoring/README.md` 已确认的目标架构。

---

## 0. 结论先行：可以完全不碰 CICD

| 层 | CICD 现状 | observability 新增 | 是否相交 |
|---|---|---|---|
| 后端进程 | `cmd/cicd-api`（独立二进制） | `cmd/observability-api`（新独立二进制） | 否 |
| 后端领域 | `internal/cicd/**`（23 个包） | `internal/observability/**` | 否 |
| 后端路由 | `/api/v1/*` | `/observability/v1/*` | 否 |
| 前端页面 | `views/cicd/**` | `views/observability/**` | 否 |
| 前端 API | `api/axiops/cicd.ts` | `api/axiops/observability.ts` | 否 |

**唯一需要改的既有文件（各 1 行）：**

1. `axiops-web/src/router/modules/index.ts` —— 把 `observabilityRoutes` 加进 `routeModules`
2. `axiops-web/vite.config.ts` —— 加一条 `/api/observability` 代理

其余全部是**新增文件**。后端若走独立进程（推荐，仿 `cmd/cicd-api`），连 `routes.go` 都不用改。

---

## 1. 目标架构（沿用已确认方案）

```
exporter / kube-state-metrics
   -> 区域 vmagent
   -> remote_write
   -> VictoriaMetrics Cluster
   -> observability-api（新增）
   -> axiops-web（views/observability，新增）
```

- 标签规范：`cloud / region / cluster_id / project_id / environment / service_id / instance_id`
- 平台只保存：数据源、采集配置、模板、规则、审计、事件与告警索引；**不复制原始时序数据**
- 查询必须注入项目/集群/服务/环境权限，**禁止自定义 PromQL 绕过标签权限**（验收标准明文要求）

---

## 2. 为什么不整包搬 KubeDoor

| 维度 | KubeDoor | AxiOps | 处理 |
|---|---|---|---|
| 后端 | Python（aiohttp + asyncpg） | Go（go-zero 风格） | 只搬逻辑（PromQL、SQL、规则），代码用 Go 重写 |
| 鉴权 | nginx basic auth（读写/只读两级） | JWT + RBAC scope | 重写，接 `common/middleware/JWTAuthMiddleware` |
| 集群接入 | agent WebSocket 外连（业务集群免开端口） | 已有 `onec_cluster` + `common/k8smanager` 直连 | 优先复用 AxiOps 直连；agent 模式后置评估 |
| 前端底座 | vue-pure-admin + Element Plus | Art Design Pro + Element Plus | 只搬**信息架构与交互**，UI 全部用 AxiOps 设计系统重写 |
| 时序库 | Prometheus / VM + PG 快照 | 目标 VictoriaMetrics | 对齐 |
| 图表 | 内联 ECharts，无组件 | 已有 `components/core/charts/*`（ECharts 6） | 用 AxiOps 现成图表组件 |

---

## 3. 复用资产清单（分三层）

### A. 直接搬（语言无关，价值最高）

| KubeDoor 资产 | 去向 |
|---|---|
| `src/kubedoor-master/promql.py`（高峰期采集 PromQL 字典） | → `internal/observability/promql/queries.go` |
| `src/kubedoor-master/func_manager/prom_overview.py`（24 条总览 PromQL） | → 同上，作为总览页查询模板 |
| `deploy/dashboards/*.json`（3 个 Grafana 看板：Dash / K8S / Node） | → `internal/observability/dashboard/templates.go` 或 Grafana 面板导入 |
| `deploy/manifests/master/alert_rules.json`（事件忽略 + 告警规则配置） | → 规则种子数据 |
| vmalert 规则（Pod / Deployment / 节点 / JVM 共 44 条 + 2 条记录规则） | → AxiOps 告警规则模板，落到既有 `alert_rules` 表 |

### B. 改写后搬（前端信息架构 / 交互）

| KubeDoor 页面 | AxiOps 落地 | 类型 |
|---|---|---|
| `monit/overview`（总览大屏：集群/节点/工作负载/Pod/PVC 计数、风险节点徽章、Top10 告警与事件、10 日告警趋势） | `views/observability/overview/` | 只读 |
| `monit/index`（Deployment 实时 CPU/内存 vs request/limit） | `views/observability/workload/deployment` | 只读 |
| `monit/nodemanager`（节点用量与可分配量） | `views/observability/node/` | 只读 |
| `monit/pod`（Pod 列表、日志、事件跳转） | `views/observability/workload/pod` | 只读 |
| `monit/stateful`、`daemon`、`service`、`ingress`、`configmap` | `views/observability/workload/*` | 只读 |
| `alarm/index` + `alarm/detail`（告警聚合、处理标记） | `views/observability/alert/` | 只读 + 处理 |
| `alarm/events`（事件中心） | `views/observability/event/` | 只读 |
| `alarm/silence` + `SilenceEditDialog`（matchers、命中预览、续期/克隆） | `views/observability/silence/` | 写（仅屏蔽） |
| `resource/index`（高峰资源管控） | `views/observability/capacity/` | 依赖采集，P4 |

### C. 暂不搬（依赖重 / 与 AxiOps 现状冲突）

- **准入管控 MutatingWebhook**：依赖 KubeDoor agent 在业务集群动态建 webhook + `k8s_res_control` 热路径表
- **高峰期采集 CronJob**：依赖 agent 在业务集群建定时任务
- **节点均衡 / 弹性调度**：与 cordon、分批迁移状态机深度耦合
- **Istio 路由管理**：AxiOps 暂无对应诉求
- **AI 助手 / MCP**：AxiOps 已有独立 AI 方向（`docs/08-ai/`），后续单独对接，不在本模块

---

## 4. 前端设计

```
axiops-web/src/views/observability/
  overview/          集群总览大屏
  cluster/           集群维度（节点、资源、健康）
  node/              节点列表与详情
  workload/          deployment / statefulset / daemonset / pod / service / ingress / configmap
  alert/             告警列表、详情、规则
  event/             事件中心
  silence/           告警屏蔽
  capacity/          高峰容量（P4）
  datasource/        数据源与采集配置（P0）
```

- **API**：`axiops-web/src/api/axiops/observability.ts`，前缀 `/observability/v1/`，复用 `utils/http` 封装
- **路由**：`axiops-web/src/router/modules/observability.ts`
  - `meta.menuCode = 'observability'`，`meta.access = { resource: 'observability', action: 'view' }`
  - `component` 用字符串路径（由 `ComponentLoader` 惰性加载），照抄 `modules/axiops.ts` 写法
- **视觉规范**（沿用我们已经统一好的那套）：
  - 卡片：`components/core/cards/*`；图表：`components/core/charts/*`（ECharts 6）
  - 区块：白底 + 1px 边框 + `0 1px 2px rgb(15 23 42/4%)`，标题条左侧 teal 强调条
  - 所有编号/数值走等宽字体，次要说明 11px 等宽灰字
  - 不引入 KubeDoor 的任何视觉资产（它的卡片/图表是内联的，本来也没有组件）
- **全局上下文**：集群选择器 + 命名空间 + 时间范围，对齐 `docs/09-frontend/README.md` 的全局上下文约定

---

## 5. 后端设计

```
cmd/observability-api/main.go            新独立进程
configs/observability-api.yaml
internal/observability/
  datasource/    数据源注册、连通性测试、凭证加密存储（复用 internal/shared/secretbox）
  query/         VictoriaMetrics / Prometheus 查询适配器（强制注入标签权限、降采样、缓存）
  promql/        从 KubeDoor 移植的查询模板（Go 常量/模板）
  dashboard/     看板定义与面板查询编排
  cluster/ node/ workload/ pod/   资源与指标聚合
  alert/ event/ silence/          告警、事件、屏蔽
  collect/                        高峰采集（P4）
migrations/0xx_observability_*.sql
```

- **复用现有基建**：`common/k8smanager`、`common/prometheusmanager`、`common/observabilityaccess`、
  `common/middleware/JWTAuthMiddleware`、`common/handler/{okx,errorx}`、`internal/shared/*`
- **接口前缀**：`/observability/v1/...`（vite 代理 `/api/observability` → `/observability`）
- **复用现有表**：`onec_cluster`、`onec_cluster_auth`、`onec_cluster_node`、`onec_cluster_resource`、
  `alert_rules`、`alert_rule_groups`、`alert_instances`、`alert_channels`
- **新增表**：
  - `obs_datasource`（数据源连接与凭证引用）
  - `obs_query_template`（PromQL 模板，移植自 KubeDoor）
  - `obs_dashboard`（看板与面板定义）
  - `obs_silence`（屏蔽规则，matchers 语义对齐 KubeDoor `alert_silences`）

---

## 6. 分阶段实施计划

| 阶段 | 内容 | 交付物 / 验收 |
|---|---|---|
| **P0 骨架** | 前后端骨架、数据源管理、最小查询页 | 能连通数据源、能画出一条曲线；路由与菜单可见；CICD 零改动 |
| **P1 只读观测（核心）** | 总览大屏 + 集群/节点/工作负载/Pod 四张页 + 集群/NS/时间范围切换 | KubeDoor「看」的能力全部落地；P95 < 3s |
| **P2 告警与事件** | 告警列表/详情、事件中心、屏蔽；对接既有 `alert_rules` 与 Alertmanager | 能看到告警、能屏蔽、能跳事件 |
| **P3 规则与模板** | 移植 44 条 vmalert 规则为模板、看板模板、下发与预览 | 规则可管理、可下发 |
| **P4（可选）写操作与高级** | 扩缩容/重启/隔离、高峰采集、节点均衡、AI 对接 | 需先定 agent 方案与审批流程 |

建议顺序：**P0 → P1 → P2**，这三步就能交付一个完整可用的可观测模块；P3/P4 视你的优先级再排。

---

## 7. 需要你拍板的问题（阻塞项）

1. **时序数据源**：现在有没有可连的 VictoriaMetrics / Prometheus / Thanos？地址与鉴权方式？
   若没有，是否允许先起一套（docker-compose 单机 VM + vmagent）用于开发联调？
2. **Grafana**：有没有现成 Grafana 可以内嵌？还是全部用 ECharts 自绘？
   （我的建议：**自绘为主**，保证视觉与交互统一；Grafana 作为可选的高级分析入口）
3. **集群接入方式**：用现有 `onec_cluster` + k8smanager 直连（需要控制面能直连 apiserver），
   还是要移植 KubeDoor 的 agent 外连模式（业务集群免开端口）？
4. **写操作范围**：先只做**只读观测**，还是要包含扩缩容 / 重启 / Pod 隔离 / 镜像更新？
5. **告警体系**：复用 `manager-api` 现有的 `alert_*` 体系，还是 observability 自建一套？
6. **容量目标**：5000 主机 / 20-50 集群是硬指标吗？这决定是否要在 P1 就做区域 vmagent 缓冲与降采样。

---

## 8. 主要风险

- **数据源未就绪**：P0/P1 无法真实验证 —— 这是当前最大的阻塞，建议先解决第 1 个问题
- **标签权限绕过**：自定义 PromQL 必须经服务端改写注入标签，前端不得直连时序库（验收标准明确要求）
- **大集群性能**：5000 主机下的 TopN 与列表查询需要降采样 + 缓存，否则 P95 难以达标
- **现有监控代码是「半成品」**：`views/cluster/monitor/prometheusrule` 等页面存在但未挂载路由，
  `application/manager-api/internal/handler/monitoring` 只有配置读写 —— 不要直接在其上扩展，
  新模块独立起，避免被旧结构绑住
