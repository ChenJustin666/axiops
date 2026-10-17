# 日志模块

## 产品设计

提供类似 SLS/Grafana Explore 的统一日志搜索、字段分析、上下文、统计和告警体验，覆盖主机、容器和应用。

## 需求

- 支持 Loki 首期接入，并保留 VictoriaLogs/Elasticsearch provider。
- 支持关键词、字段、时间范围、上下文和实时追踪。
- 支持 JSON、Nginx、Java、Kubernetes 等解析模板。
- 支持日志告警、保存查询和查询审计。
- 所有查询按项目、环境、服务、集群和 Namespace 过滤。

## 技术设计

    Vector / Alloy / Fluent Bit
      -> regional gateway
      -> Loki
      -> observability-api
      -> log explorer

高基数字段不进入 Loki label；原始正文不复制到业务 MySQL。查询 API 只接受受控语法并强制追加权限标签。

## 实施计划

1. 定义日志空间、标签和保留策略。
2. 实现 Loki provider、范围查询、上下文和 tail。
3. 实现采集配置、解析模板和安全下发。
4. 实现搜索页面、字段面板、统计和告警。
5. 增加查询限流、超时、扫描量保护和审计。

## 完成进度

- 已完成：Loki 首期方向和统一采集方案确认。
- 可复用：现有 Loki 相关配置和页面代码。
- 未开始：目标日志 API、权限过滤和新搜索页面。

## 验收标准

- 无法使用原生 LogQL 绕过权限标签。
- 支持前后 100 行上下文和实时日志。
- 大范围查询有超时、限流和明确错误提示。
