# Kubernetes 管理模块

## 产品设计

统一管理多集群、Namespace、工作负载、Pod、Service、Ingress、配置和受控终端，关联项目、环境、服务、监控、日志与发布。

## 需求

- 支持 20-50 集群扩展和多云集群接入。
- 支持 Deployment/StatefulSet/DaemonSet、Pod、Service、Ingress。
- 支持 YAML 查看、扩缩容、重启、事件、日志和受控 Terminal。
- Secret 默认脱敏，不允许普通用户导出明文。
- 按集群、项目、环境、Namespace 和动作授权。

## 技术设计

platform-api 保存集群元数据和加密凭据；访问集群时先解析资源归属，再鉴权并调用 client-go。受限网络优先通过 ClusterAgent 主动连接。

旧版页面操作说明见[使用指南](USER_GUIDE.md)，该指南仅用于现有界面，目标页面完成后更新。

## 实施计划

1. 建立集群、项目、环境、Namespace 归属关系。
2. 接入统一授权和凭据加密。
3. 收敛工作负载查询与受控写操作。
4. 重构多集群前端和资源详情联动。
5. 接入监控、日志和发布事件。

## 完成进度

- 已完成：旧代码具备多集群和基础工作负载管理能力。
- 进行中：目标平台的 cluster API 骨架。
- 未完成：细粒度授权、统一资源归属、目标前端和 ClusterAgent。

## 验收标准

- 开发只能看到授权的 dev/test 集群与 Namespace。
- 生产写操作只有授权运维可执行并全量审计。
- 任一资源详情可跳转到对应监控、日志和发布记录。
