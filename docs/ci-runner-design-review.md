# AxiOps 构建生命周期与成熟 Runner 的设计对照

核对日期：2026-09-25。依据官方文档，不把不同产品的默认值直接照搬为 AxiOps 的产品规则。

## 核对的依据

- [GitLab Runner Kubernetes executor](https://docs.gitlab.com/runner/executors/kubernetes/)：构建分为 Prepare、Pre-build、Build、Post-build；Post-build 包含缓存创建及制品上传。提供 Pod 总超时、终止宽限期、Kubernetes API 重试和 `cleanup_resources_timeout`。其中默认 5 分钟的 cleanup timeout 是**清理操作时限**，不是额外保留 Pod 五分钟。
- [GitLab Runner advanced configuration](https://docs.gitlab.com/runner/configuration/advanced-configuration/)：`concurrent` 是跨 Runner 的总并发上限，各 Runner 另有 `limit`；容器、存储、关闭过程分别配置。不能单凭 CPU 核数决定并发数。
- [GitLab job artifacts](https://docs.gitlab.com/ci/jobs/job_artifacts/)：制品通过 `expire_in` 等规则单独保留，最近成功流水线的制品还有独立保护行为。执行容器的清理不等于删除制品。
- [Jenkins Kubernetes plugin](https://plugins.jenkins.io/kubernetes/)：Pod 生命周期提供 `podRetention`、`activeDeadlineSeconds`、`idleMinutes` 等设置，支持 `onFailure()` 等排查保留策略。基础设施故障的 `retry(..., conditions: [kubernetesAgent(), nonresumable()])` 区别于编译错误、OOM 等普通失败。
- [Jenkins Workspace Cleanup](https://plugins.jenkins.io/ws-cleanup/)：工作目录清理是显式的构建前/后步骤，可按成功、失败、取消等状态配置；deferred wipeout 可交给 Resource Disposer 后台处理。不能把工作目录缓存、构建记录和临时执行器当作同一个存储对象。

## AxiOps 的取舍及实现

| 事项 | 本项目规则 | 当前实现 |
| --- | --- | --- |
| 成功/失败后的 Job | 日志及结果持久化后回收，正常路径不额外保留现场 | 默认额外保留 0 秒；主动前台删除 Job，确认执行资源删除后才确认清理完成，不依赖 TTL 控制器做立即删除 |
| 排查保留 | 仅需要检查集群现场时主动设置有界时长 | 保留秒数仍可配置；不默认保留所有失败 Pod 24 小时 |
| 失败原因 | 删除前保存能解释失败的诊断 | 保存 Pod/node 身份、等待/退出原因及日志；补齐 Pod `Reason/Message`，包括磁盘不足导致的 Evicted |
| 日志完整性 | 页面可见不等于日志已完整持久化 | 结果/日志落库后才回收；日志截断或降级标记会在页面输出中提示 |
| 清理重试 | 构建结果与资源回收分别记账 | cleanup_pending 在数据库持久化，失败重试且不改写成功/失败结论；成功后替换旧“将重试”提示；不支持清理确认的执行器不能假报完成 |
| worker 重启 | 优先恢复原执行身份 | 固定 run ID 与幂等键查回 Job；提交结果不确定时保留额度，不盲目再跑一遍业务命令 |
| 自动重试 | 区分临时基础设施错误与确定的业务失败 | Git 瞬时网络失败最多 3 次；编译错误、认证失败、OOM 不通过无限重试或自动扩大权限掩盖 |
| 并发 | 总并发、执行器并发与资源调度共同约束 | 平台/项目队列上限、可选节点槽位上限、CPU/内存/临时磁盘设置；4c8g 的并发建议需要实际峰值验证 |
| 工作区/缓存/产物 | 分别设置隔离和生命周期 | 临时源码独立 emptyDir；PVC 按项目/服务分目录，同服务持久缓存串行；日志期限独立；镜像和制品沿用仓库策略 |
| 资源所有权 | 只能取消/清理本次执行创建的资源 | Job 核对构建 ID/幂等键，Secret 核对归属，删除使用 UID 前置条件，避免重名误删 |
| 宿主机 | 复用公共规则，由 agent 实现本机执行细节 | 已约束 RecoverBuild / FinalizeBuild；实际构建 agent 尚未接入，必须具备持久台账、资源预算、整个进程组终止、隔离目录和确认后清理 |

默认立即回收是 AxiOps 针对短生命周期构建资源的选择，依据是数据已经从执行资源转移到平台，并且额外占用资源没有必要；不是声称 GitLab、Jenkins 所有执行模式都使用完全相同的默认值。

## 尚不能宣称已经达到成熟产品同等能力的部分

- 当前构建日志仍是有上限快照，未实现完整的增量上传与断点续传。不能承诺超出采集上限或 kubelet 已轮转的日志都可以找回；即时回收不会改变这项事实，因此页面必须显示截断/降级提示。
- 没有在真实集群执行本轮断网、强杀 worker、节点故障的完整演练；目前相关流程通过 fake Kubernetes client、MySQL 隔离库及脚本回归验证。
- 节点就绪/构建器前置条件、长期离线的遗留资源、持久缓存容量，仍需要运维监控和回收策略。现有 RootlessKit 的内核条件不会靠任务重试自动修复。
- 依赖缓存的容量淘汰、Runner 自动伸缩、完整宿主机构建 agent 不是本轮已交付能力；应按独立需求和验收场景实现，不能只暴露一个界面选项就算接入完成。

详细操作规则见 [构建并发、恢复与清理](ci-execution-reliability.md)。
