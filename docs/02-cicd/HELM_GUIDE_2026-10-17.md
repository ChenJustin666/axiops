# Helm 使用步骤

本教程对应 2026-10-17 源码。先配套更新 CI/CD API、Workflow Worker 和前端；目前运行中的旧后端尚未替换，不能以旧页面作为新版功能验收。

本版新增共享模板版本和通用工作负载 Chart，使用前须按序执行迁移 052。多控制器、ALB、存储、Secret 多字段及模板更新操作见 [共享模板使用教程](HELM_SHARED_TEMPLATES_2026-10-17.md)。

## 1. 创建项目、服务与环境

1. 创建项目，交付类型选 **Helm Chart**。初始化创建需要的环境，绑定对应集群和 Namespace。只需测试环境时不必先建生产。
2. 创建服务只提供 **模板**、**代码仓库导入** 两种入口。模板从模板库选择 Helm Chart；仓库导入分别填写仓库路径（如 `team/repo`）、Chart 目录（如 `deploy/chart`，根目录填 `.`），选择完整 Chart 文件后固定到 Commit。创建完成后仍可编辑 Chart 文件；已有自定义 Chart 可继续维护。
3. 每个独立发布的应用建一个服务/Release。一个包含多个紧密耦合应用的 Chart 会作为一个 Release 升级、回滚，不等于每个 Deployment 都能单独回滚。
4. 服务的“构建配置”继续配置 Git、编译命令、Dockerfile 与镜像仓库。Helm 只负责把镜像部署成资源，不替代编译与打镜像。

默认文件结构：

```text
Chart.yaml
values.yaml
templates/
  deployment.yaml
  service.yaml
```

可在“Chart 文件”添加 templates/ingress.yaml、configmap.yaml 等。保存 Chart 创建新版本；其他环境仍使用之前选择的版本。模板库中已存在的用户条目不会被新的内置默认文件覆盖，可自行编辑或新建模板。

## 2. 配置测试环境

进入 **服务与配置 → 选择服务 → 选择测试环境 → 环境 values**，选择 Chart 版本，直接修改 values。例如起始 Chart：

```yaml
replicaCount: 1
image:
  repository: example.invalid/team/service
  tag: latest
service:
  port: 28080
resources:
  requests:
    cpu: 250m
    memory: 512Mi
  limits:
    cpu: "1"
    memory: 1Gi
env:
  - name: SPRING_PROFILES_ACTIVE
    value: test
  - name: JAVA_OPTS
    value: "-XX:MaxRAMPercentage=70.0 -Dfile.encoding=UTF-8"
envFrom:
  - secretRef:
      name: app-runtime
command: []
args: []
```

- 起始 Chart 支持以上字段。app-runtime 需在目标 Namespace 已存在，或由所用 Chart 创建；不需要 Secret 时将 envFrom 设为 `[]`。密码、SK、Token 不要作为普通 values 明文填写，优先使用 Kubernetes Secret 引用。
- image 的 example.invalid 是占位值；构建/选择镜像发布时平台注入实际固定摘要。默认路径是 image.repository / image.tag，通常不需要修改。自定义 Chart 的镜像键不同时，在折叠的“镜像映射与继承参数”中修改路径。
- command/args 为空表示使用镜像入口，填入后覆盖镜像启动方式。JAVA_OPTS 是否生效还取决于镜像入口是否使用它。
- 自定义 Chart 可能使用不同字段名，例如 replicas、container.port；必须按该 Chart 的 values 约定填写，不是所有 Chart 都适用上述键。

点 **预览与保存**，查看真实渲染的 Deployment、Service 等，再保存。预览中的 Secret 正文隐藏；没有可用业务镜像时可能出现校验占位镜像。**保存只保存配置，需执行工作流后才更新运行资源。**

## 3. 测试构建与部署

1. 打开绑定测试环境的工作流，执行，选择服务与分支，选择构建并部署。
2. 构建成功后，平台将该服务产出的 `repository:tag@sha256:...` 注入 Chart，验证渲染结果确实使用它。
3. Helm 安装或升级对应 Namespace 的 Release，等待资源就绪。运行详情查看部署节点的准备、目标、等待、结果及 Release revision。
4. 失败时查看节点原因，并到环境资源查看 Pod 事件和容器日志。阶段等待提示不代表应用已健康。

私有仓库通过已有仓库凭据链路生成/引用目标 Namespace 拉取 Secret，不需把仓库密码粘贴到 values。Chart hooks 若使用私有镜像，需自行在 hook 模板中声明拉取凭据引用。

## 4. 新增生产环境

1. 新建生产环境，绑定生产集群和 Namespace；它可以与测试处于不同集群。
2. 选择相同服务，在生产环境选择 Chart 版本，填生产 values。例如副本改 3、内存上限改 4Gi、SPRING_PROFILES_ACTIVE 改 prod；Nacos 地址、域名、Secret 名称也按生产配置。
3. 预览后确认生产目标，保存。环境复制不会自动绑定 Helm 服务；复制后也必须完成这一步。
4. 在生产工作流执行页选择服务，默认使用已有镜像，选择经过测试的构建镜像。来源默认测试，可在高级中修改；“构建成功”不等于“测试通过”，需结合来源部署结果、时间、Commit 核对。
5. 发布使用所选镜像摘要和**生产保存的配置**，不把测试 values 搬到生产。通常无需重新构建；确需重新构建时显式选择构建并部署。

镜像保留原构建 tag（可能含 test），环境不靠 tag 名区分。生产准确使用相同 digest，不通过改标签或重新构建伪装成同一份制品。

## 5. 日常修改与环境同步

- 改生产内存：选择生产环境 → 改 values 的 resources → 预览保存 → 执行生产工作流，复用当前或选定镜像。
- 改测试域名：只改测试环境 values；前提是 Chart 已有使用该字段的 Ingress 模板。
- 加通用能力：编辑 Chart 文件，保存新版本；选择要升级的环境，选此版本并预览保存。其他环境不自动升级。
- 同步：从来源环境点“同步到其他环境”，勾选需要复制的顶层字段；例如只选 resources。目标未选中的域名、地址等保留，数组整项替换。Chart 版本需另行勾选。核对目标 values 前后与渲染资源，再保存；不自动部署。

顶层字段内含多项配置时会整项覆盖，例如 env 是数组，选择 env 将替换目标整个 env。只需修改其中一个生产变量时，直接在生产 values 修改更精确。不要全选测试配置覆盖生产。

## 6. 回滚

在目标环境/服务的发布历史中选择已有成功 Helm 版本，核对 Release、镜像和目标后预览回滚。平台按保留的 Helm Release revision 恢复资源，并产生一个新的 revision；不修改 Git，也不重新编译。

Release 历史默认最多保留 20 个版本，历史已被清理或目标不一致则不能回滚。目标历史中的所有工作负载镜像须固定 digest；可变 tag 的 sidecar 需先改为不可变引用，才能保证恢复的是同一份镜像。回滚结果需等待执行记录成功，不能把点击或接口受理当作回滚完成。

## 当前验收范围

源码、隔离数据库、组件与 provider 测试有验证记录，见 [Helm 交付状态](HELM_DELIVERY_2026-10-17.md)。当前尚未替换运行中后端，未完成浏览器视觉验收和真实集群发布/回滚演练，不能据此称生产端到端验收完成。
