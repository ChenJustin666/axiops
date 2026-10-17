# YAML 部署的镜像拉取凭据

构建成功推送镜像，只能证明构建 Job 有推送权限。部署 Pod 由目标节点上的 kubelet 拉取镜像，需要 Pod 所在命名空间中的拉取 Secret。构建命名空间的 Secret 不会自动传到 QA 或生产。

## 配置与执行

服务沿用系统管理 → 构建资源 → 镜像仓库中的现有绑定，无需再次手工填写账号密码。YAML 工作流部署时：

1. 按项目、服务读取绑定的镜像仓库和加密凭据。
2. 校验渲染后的资源目标 namespace、镜像 host 和配置允许的仓库空间。
3. 在本次部署目标 namespace 中创建或更新 `kubernetes.io/dockerconfigjson` Secret。
4. 保留已有 `imagePullSecrets`，向匹配仓库的 Pod 模板追加平台管理的引用。
5. 保存只包含引用的部署快照，再 Apply 工作负载。

部署集群账号需具备目标命名空间 Secret 的 get / create / update 权限；目标 namespace 尚不存在时沿用现有部署的 namespace 创建能力。Secret 准备失败会直接阻断部署，并返回目标 Secret 与检查方向。

当前支持 Deployment、StatefulSet、DaemonSet、Job、CronJob，检查普通容器和 initContainers。其他 host 的容器继续使用自身配置。未绑定仓库或匿名仓库不自动创建凭据。

Kubernetes 拉取凭据使用用户名和密码；仓库要求 access token 时，将 token 作为密码配合仓库要求的用户名。只有 identity token 的配置会明确报错，不生成 kubelet 无法使用的 Secret。

## 生命周期与隔离

- Secret 名含 project ID、service ID 和 registry host 摘要，相同服务在不同 namespace 各有自己的 Secret。
- 平台只更新自身 ownership 标签匹配的 Secret，不覆盖用户同名 Secret。
- 重试时内容不变则不写入；仓库凭据轮换后，在下一次对应环境部署时更新 Secret。
- Secret 是部署工作负载的持续依赖，不能跟构建 Job 一起删除，否则节点重调度或镜像缓存清理后会再次拉取失败。
- 密码不进入用户 YAML、工作流配置快照或日志。没有给默认 ServiceAccount 广泛添加凭据，也不要求逐台节点执行 docker login。
- 此机制只处理鉴权。自建 HTTP registry 或私有 CA 的运行时信任配置仍需由目标集群支持。

此次仅接入 YAML 部署执行器。Helm 图表、预览和旧回滚快照不自动新增引用；新 YAML 部署快照包含引用，回滚依赖该 namespace 中相应 Secret 仍然可用。服务换绑仓库后旧 Secret 暂不自动回收，以免影响仍使用旧镜像的工作负载。

## 2026-09-28 risk-qa 实查

`risk-server` 原 Pod 缺少拉取凭据，仓库返回 `insufficient_scope: authorization failed`。平台中现有凭据验证镜像 manifest 成功，因此根因是部署凭据遗漏。

已在 `risk-qa` 创建 `axiops-pull-p19-s17-08d3cbb82d`，仅为现有 Deployment 补上引用。新 Pod 成功下载原镜像，digest 为 `sha256:65fff76674866000848f3065c6b86f04b87e9ec86e696d87e23e851d1c37c6c3`。worker 已重新编译并启动，后续 YAML 部署执行新逻辑。

镜像拉取成功后，应用另报 `No spring.config.import property has been defined`，exit code 1。当前手写 Kubernetes 模板没有消费仓库 Helm values 的 Java / Nacos 配置，应用尚未就绪；需要确认 QA 的 Nacos 配置后补齐，不能通过关闭 import 检查假装部署成功。

历史 workflow #34 保留失败状态。本次现场修复不篡改历史工作流、构建验证状态或镜像产物。
