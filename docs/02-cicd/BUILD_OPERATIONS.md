# 构建执行约定

> 本文是早期目标设计，其中 Registry 自动缓存、ECS 执行及 JAR 自动打包尚未接通。以 [CI_CLUSTER_SETUP.md](CI_CLUSTER_SETUP.md) 的当前实现与验证边界为准，不应把下文的目标能力视为已实现。

## 集群、资源和排队

默认 `kube_job` 使用 workflow-worker 配置中的 `Build.ClusterID` 和 `Build.Namespace`。这是平台级构建集群，避免用户把构建 Pod 误部署到业务命名空间。后续可将同一 Provider 扩展为按项目/构建池路由，但一次运行仍只选择一个构建集群。

每个构建创建一个短生命周期 Job/Pod。CPU 和内存是构建 Pod 的 requests/limits，不是业务服务资源。资源不足时 Pod 保持 Pending，队列不会丢失；状态接口会返回 `waiting for cluster resources` 及 Kubernetes 调度原因。平台和项目并发策略控制同时运行数，超过上限自动排队。要加快构建，应增加构建节点容量、合理设置项目并发，并启用 Registry/PVC 缓存，不能盲目把并发调到 10 倍。

默认超时 1800 秒（30 分钟），允许配置 60 秒到 86400 秒。超时后 Job 终止并记录日志；超时应通过拆分构建、缓存依赖和增加资源解决，不能无限提高超时掩盖死锁。

## 缓存

`emptydir` 是默认策略，任务结束即清理，适合安全隔离和小项目。`pvc` 使用管理员预先在构建命名空间创建的 PVC，多个 Pod 可复用但需要读写权限和项目隔离。`registry` 适合多节点和多 Agent，使用 BuildKit/Kaniko 的远端 layer cache；切换 Agent 不会丢缓存。PVC 名称必须显式配置，平台不会自动挂载任意 PVC。

## 构建模式

- `dockerfile`：默认模式。代码仓库中的 Dockerfile 和构建上下文由服务配置带出；多阶段 Dockerfile 在同一个 builder 中完成测试、编译和镜像推送，通常不需要额外 Shell。
- `jar_image`：适合标准 Maven/Gradle JAR。构建器生成 JAR，再放入受支持的 JRE 运行时镜像并推送镜像；部署阶段仍使用镜像 digest。
- `artifact`：用于暂时无法容器化的老 JAR 或压缩包。构建器上传制品并返回 HTTPS URL + SHA-256，ECS Agent 下载校验后用 systemd/脚本运行。制品不能直接当作 Kubernetes 镜像。

构建器镜像是“执行构建工具的镜像”（例如 JDK、Go、Node、Kaniko），产物镜像是“最终部署的应用镜像”，两者职责不同。模板版本应进入 BuildConfig 快照和输入指纹，升级模板会产生新的构建输入。

## 仓库、分支和镜像地址

界面先选择代码源、组织/仓库，再动态读取分支和 commit；提交构建时固定完整 commit，不能只依赖可变分支名。服务的 `registry_id + image_name` 是镜像目标的权威来源：`registry.url/image_name:tag`。镜像仓库必须先在平台配置并通过凭据校验；不存在的项目/命名空间由保存或首次推送前校验并给出明确错误，平台不会静默创建错误路径。

## ECS Agent

构建配置可把 Provider 临时切换为 `ecs_agent`，前提是项目存在在线 Agent；切换只影响新的构建，运行中的 Job 不迁移。Agent 可部署在全球任意位置，只要能通过 HTTPS 访问 AxiOps API、能访问代码仓库和镜像仓库，并满足组织的网络策略。需要安装 Agent 二进制、受信任 CA/证书和对应语言/Docker/BuildKit 工具；生产环境推荐 rootless BuildKit 或受控 Docker，不开放 AxiOps API 的集群管理员权限。Agent 断线时任务保持租约并可重试，日志和制品仍需回传到平台。
