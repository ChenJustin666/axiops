# 部署结果判定与日志展示

部署节点不能复用“容器构建日志”作为结果。构建 Job 的日志来自 CI builder；YAML 部署由 Kubernetes API 执行，结果应由 Apply、滚动发布和 Pod 状态共同判定。

## 成功条件

`Apply` 成功只表示资源已提交。生产 worker 继续检查 Deployment、StatefulSet、DaemonSet 的 generation 和副本状态，Job 检查 Complete/Failed condition。未就绪时重试，达到部署期限后失败。业务健康依赖应用配置的 readinessProbe；本轮未增加外部业务接口探测。

副本为 0 的缩容目标可以完成，日志明确标记停服，不能理解为应用正在提供服务。Service、CronJob 等只记录提交，没有支持的 rollout 对象时明确说明应用健康未验证。

## 失败信息

进行中、成功和失败都保存执行证据：资源提交、各 workload 副本数量和 conditions、Pod phase、容器 ready/restarts/waiting/termination/previous exitCode，以及 workload/Pod UID 关联 Events。采集失败单独标记 unavailable，不改变原 rollout 结论。检查所有支持的 workload，不在第一个未就绪对象处提前返回。

完整 selector 支持 matchLabels/matchExpressions，空 selector 不扫描 namespace；最多查询 20 个 Pod 并显示其中较新的 5 个，每个对象显示最多 5 条采样 Events。查询有 8 秒期限和文本上限，采样和截断明确提示。不保存 manifest/env/Secret/kubeconfig，事件和错误中的常见 URL 密码、Bearer、password/token 等作脱敏。本轮不采集容器 stdout/previous stdout，状态摘要不能替代应用堆栈日志。

## 当前系统结论

迁移 `038_workflow_task_log.sql` 增加独立 `workflow_task_run.log_text`。worker 在完成任务前持久化日志，重复快照去重；上限约 64 KiB，保留开头与最近部分，截断有提示。重启从 DB 继续，终态拒绝迟到日志覆盖。API 使用原工作流详情和权限，不另开无鉴权接口。

页面区分构建日志、YAML 部署/回滚检查和其他执行日志。历史任务缺少日志时明确提示“未保存”，不回填当前资源状态。先应用迁移，再重启 CI/CD API/worker。

验证覆盖 rollout、超时、诊断权限失败、缺失 container state、CrashLoop previous exitCode、selector/UID 隔离、脱敏、日志限额、MySQL 持久化和前端轮询。当前 QA 只读核实为副本 0；本轮未触发实际业务部署。验证详情、设备策略限制与进程状态见 HANDOFF.md。
