# 专用 CI 集群接入（当前实现）

构建集群和部署集群独立。先在「集群管理」接入专用集群，使用平台分配的集群 ID；在该集群准备 `axiops-cicd` Namespace。管理员可在 CI/CD 的「构建资源」页面保存绑定和默认值；Worker 启动时读取数据库设置。未保存平台设置时，才使用 `configs/workflow-worker.yaml` 中的 `Build.Enabled`、`Build.ClusterID`、`Build.Namespace` 作为引导回退。启动会检查 Namespace 并输出实际绑定。

`Build.Defaults` 集中配置 BuilderImage、BuildCommand、TimeoutSeconds、ResourceCPU、ResourceMemory、CachePVCName、CacheDir、JobRetentionSeconds。服务的构建器、命令、CPU、内存留空、超时填 0、缓存选「继承平台配置」即可复用。已有服务的显式值保持优先。构建镜像必须包含所需工具和可执行的结果上报命令；仅填写 `golang` 或 `maven` 镜像不会自动获得 Docker/BuildKit 推送能力。

CPU/内存可留空；Job deadline 默认 1800 秒（包括调度和 checkout）。成功、失败、超时的 Job 默认在结果和日志保存后立即安排清理，不额外保留 24 小时；尚未完成保存时不启用 TTL，便于 Worker 恢复采集。页面记录和已保存日志不随 Job 删除，额外保留时长仅用于主动排查。平台/项目并发沿用 `/api/v1/build-concurrency`，不需要每个服务重复配置。不要在有活跃构建时修改 CI ClusterID；当前数据库中的 Job 身份不包含旧集群路由，必须先停止接单并排空运行。

## 源码与缓存

每次实际构建创建新的 emptyDir 工作区，浅拉取指定 commit，并校验 checkout 的完整 SHA。没有 commit 时按分支浅克隆，此类请求禁止按构建指纹复用历史结果，避免分支更新后误部署旧代码。源码没有跨 Pod 持久化；同一产物向其他环境发布应选择既有构建结果，不需要重新拉代码和编译。

继承平台构建器的任务目前也不自动复用历史构建，因为平台默认工具链尚未冻结到构建指纹中。跨环境发布应明确引用已经验证的 `仓库/镜像@sha256:...`；再次点击「构建并部署」仍可能触发构建，不能将其等同于发布既有产物。前端编译时写死的环境地址也需要改为运行时配置，才能复用同一镜像。

持久化依赖缓存可一次性配置 `CachePVCName`。PVC 必须位于 CI Namespace，多节点共享推荐支持 RWX 的存储；平台按 `project-ID/service-ID` 使用 subPath。此分目录是防止误混用，不是恶意构建的安全隔离。不同信任域应使用独立 PVC/构建池。同一服务的并发缓存仍要求工具支持锁；不要共享可写 Git 工作目录。工具必须指向 CACHE_DIR，例如 Maven 使用 `-Dmaven.repo.local="$CACHE_DIR/m2"`。

Registry layer cache 是跨节点优选，但当前 `registry` 选项只向自定义构建器传递策略，不会自动启动 BuildKit 或设置 import/export-cache。需要先提供经过验证的构建器命令。PVC 与日志归档、产物仓库三者不同：镜像应存 OCI registry，JAR 应存制品库，日志应存对象存储。

## 日志

Worker 采集 git-clone 和 builder 两个容器的日志，附容器标题，中文以 UTF-8 保存。调度等待原因写入运行记录。观察使用轮转避免一个长任务阻塞其他构建日志，轮询有间隔。

MinIO/S3 归档通过已有 AXIOPS_BUILD_LOG_ENDPOINT、BUCKET、ACCESS_KEY、SECRET_KEY、USE_SSL、PREFIX 环境变量集中配置。API 和 Worker 使用一致配置。当前是有大小上限的日志快照，归档也不会补回被截断的数据；不是完整无损流式采集。生产长日志需接入持续日志采集方案后再承诺全量保留。

## 跨集群部署与检查

在项目的每个环境中绑定目标 cluster_id 和 namespace。workflow-worker 通过目标集群凭据使用 client-go Server-Side Apply；Helm 使用现有 Helm SDK。CI Pod 不持有目标集群 kubeconfig，不需要 kubectl。Worker 必须能访问目标 Kubernetes API，目标节点必须能拉取产物镜像；私有镜像的 imagePullSecrets 必须在目标 Namespace 配好（构建推送凭据不会自动成为拉取凭据）。

YAML Apply 后保持运行中，轮询 Deployment、StatefulSet、DaemonSet 的就绪计数/观察代次，以及 Job 的完成或失败条件；默认部署检查 600 秒，节点 timeout_seconds 可覆盖。部署目标从数据库快照读取，Worker 重启后可以恢复检查。配置资源提交成功不等于业务接口测试通过；业务 HTTP、数据库迁移等应配置独立测试步骤。StatefulSet 的 OnDelete/partition 策略按 Kubernetes 语义处理，不强制所有副本更新。

## 上线验证边界

本轮未接入新的真实 CI 集群、未推送镜像或执行真实部署。先运行有序迁移（包含前轮 031/032），再重启 cicd-api 和 workflow-worker；前端开发模式热更新，生产需重新构建。验收应使用一个测试仓库完成构建、查看拉代码/构建日志、拿同一个 digest 发布至两个测试环境，并演练 ImagePullBackOff、就绪超时、Worker 重启。

ECS 构建 Provider 尚无可运行的下发实现，不能临时切换并期待执行。JAR 自动包装、制品自动上传、完整构建池 UI 也未由本轮提供。
