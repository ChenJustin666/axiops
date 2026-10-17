# 监控模块

## 产品设计

为多云主机、Kubernetes、应用和中间件提供统一监控。交互参考 Grafana/KubeDoor，但页面以运维高频扫描、跨资源对比和精确数据为主。

## 需求

- 支持约 5000 台主机、首期 10 个并扩展到 20-50 个集群。
- 支持多云、多 Region、VPC 和受限网络主动上报。
- 支持总览、主机、节点、工作负载、服务和中间件模板。
- 支持告警规则、静默、聚合、值班和发布事件关联。
- 查询必须注入项目、集群、服务和环境权限。

## 技术设计

    exporter / kube-state-metrics
      -> regional vmagent
      -> remote_write
      -> VictoriaMetrics Cluster
      -> observability-api
      -> axiops-web

标签至少统一 cloud/region/cluster_id/project_id/environment/service_id/instance_id。平台保存数据源、采集配置、模板、规则和审计，不复制原始时序数据。

## 实施计划

1. 建立标签规范、数据源和权限映射。
2. 实现 VictoriaMetrics 查询适配器与基础指标 API。
3. 实现主机、集群、节点、工作负载和服务页面。
4. 实现规则、告警、值班和发布标记。
5. 做 5000 主机容量与查询压测。

## 完成进度

- 已完成：目标架构和容量方向确认。
- 可复用：现有 Prometheus/Kubernetes 监控相关代码。
- 未开始：目标 observability-api、统一查询和新前端。

## 验收标准

- 常用页面 P95 小于 3 秒，数值与源系统一致。
- 断网可缓冲，恢复后可续传，区域故障不拖垮中心。
- 用户无法通过自定义 PromQL 绕过标签权限。
