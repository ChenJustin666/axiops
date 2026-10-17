# CI/CD API 契约

## 构建

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/builds` | `build:run` | 创建构建记录并持久化 submit 命令 |
| `GET` | `/api/v1/builds` | `build:view` | 按项目、服务、状态分页查询 |
| `GET` | `/api/v1/builds/{id}` | `build:view` | 查询详情、阶段和日志 |
| `POST` | `/api/v1/builds/{id}/cancel` | `build:cancel` | 幂等请求取消，状态先变为 `cancelling` |
| `POST` | `/api/v1/builds/{id}/retry` | `build:run` | 基于原提交创建新的构建运行 |
| `GET` | `/api/v1/builds/{id}/logs` | `build_log:view` | 查询构建日志 |

触发接口返回的 `id` 只代表构建记录已经可靠入队。实际执行、阶段、制品 digest
由 worker/provider 回调更新；客户端应轮询详情或订阅后续日志流。

## Worker 命令协议

`build_command.payload` 当前只包含 `build_run_id`，worker 必须从平台数据库读取该运行
的配置快照和非敏感构建参数。凭据只能通过凭据引用获取，禁止把 Token、密码、私钥写入
`config_snapshot` 或日志。

命令状态：`pending -> processing -> succeeded`，失败可回到 `pending`，超过最大尝试
次数转为 `dead`。`idempotency_key` 是全局唯一键，provider 必须以它实现外部幂等。

## Provider 契约

provider 必须实现 `Name`、`Submit`、`Cancel`。`Submit` 返回外部执行 ID 及可选的
Job/Pod/Node 信息；重复提交同一幂等键必须返回同一执行，不得创建第二个构建。

首选 provider 为 `kube_job`；受限网络、内网仓库或固定工具链使用独立的 `ecs_agent` 构建 provider。`host_agent` 仅用于主机部署，不得作为构建 provider。Tekton 暂不进入主干。构建配置支持 `dockerfile`（默认，多阶段）、`jar_image` 和 `artifact` 三种输出模式。
`jenkins`。provider 注册和凭据校验由 worker 完成，API 不直接持有集群管理员凭据。

## 项目交付模型 API

项目类型接口已在阶段 0 落地；服务 API 仍是下一项工作。当前旧 `app_service` 和
`release_record` API 不应被前端扩展为最终交付模型。

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/projects` | `project:create` | 新项目必须在请求中提供 `delivery_type` |
| `PUT` | `/api/v1/projects/{id}/delivery-profile` | `project:update` | 仅确认旧项目的类型一次 |
| `GET` | `/api/v1/projects/{id}/delivery-profile` | `project:view` | 查询项目类型、迁移状态和限制 |
| `POST` | `/api/v1/projects/{id}/services/import` | `service:create` | 从 Git 导入 YAML、Chart 或主机服务定义并返回解析预览 |
| `POST` | `/api/v1/projects/{id}/services` | `service:create` | 基于已确认项目类型创建服务 |
| `POST` | `/api/v1/services/{serviceID}/revisions` | `service:update` | 创建不可变服务定义版本 |
| `GET` | `/api/v1/services/{serviceID}/revisions` | `service:view` | 查看不可变服务定义版本 |

创建请求示例：

```json
{
  "name": "payment",
  "code": "payment",
  "delivery_type": "helm"
}
```

`delivery_type` 只允许 `k8s_yaml`、`helm`、`host`。`unclassified` 是旧项目迁移状态，只允许查看和类型确认，不允许创建部署运行。`PUT delivery-profile` 只接受
`unclassified/pending_confirmation -> <明确类型>/active` 的一次转换；已经明确类型的项目返回参数无效，不能改型。

部署顺序：先运行 `axiops-migrate`，再部署包含该 API 的平台服务；未完成迁移时服务会拒绝启动。
生产执行清单见 [数据库迁移与发布步骤](../10-engineering/DATABASE_MIGRATIONS.md)。

服务创建时不接受客户端决定最终交付类型。CI/CD 服务读取项目的 `delivery_type` 并派生服务类型：

```json
{
  "name": "payment-api",
  "code": "payment-api",
  "source_type": "git",
  "repo_url": "https://git.example.com/payment/api.git",
  "repo_branch": "main",
  "source_ref": {
    "path": "deploy/helm"
  }
}
```

`source_type` 支持 `git`、`manual`、`template`；`source_ref` 必须是非敏感 JSON，Token、密码和私钥只能保存为凭据引用。Git 来源必须提供 `repo_url`。

revision 创建请求最大 4 MiB，由服务类型决定 `definition_format`，服务端计算 SHA-256：

```json
{
  "source_commit": "d34db33f",
  "definition": "apiVersion: apps/v1\nkind: Deployment\n..."
}
```

`008_cicd_service_revision.sql` 必须在新版 CI/CD API 前执行，并且依赖 `007` 已完成。旧服务默认保持 `unclassified`，不会根据旧 `deploy_type` 自动变成 YAML、Helm 或主机服务。

### Helm Chart Git 导入

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/services/{id}/helm-import/preview` | `service:view`、`code_source:view`、`credential:view` | 读取固定 Git commit，校验并预览 Chart |
| `POST` | `/api/v1/services/{id}/helm-import` | 上述读取权限 + `service:update` | 再次读取并校验 commit 后创建不可变 service revision |

请求中的 `paths` 是仓库根目录相对路径，必须全部位于 `chart_path` 内并包含 `Chart.yaml`：

```json
{
  "code_source_id": 9,
  "repo_path": "payment/backend",
  "ref": "main",
  "chart_path": "deploy/payment",
  "paths": [
    "deploy/payment/Chart.yaml",
    "deploy/payment/values.yaml",
    "deploy/payment/templates/deployment.yaml",
    "deploy/payment/templates/service.yaml"
  ],
  "expected_commit": "d34db33f"
}
```

预览返回 Chart 名称、版本、依赖、文件数、模板数、默认 values 顶层键、镜像提示、commit 和内容摘要，
但不返回完整 definition。导入接口必须携带预览返回的 `commit_id` 作为 `expected_commit`；分支在预览后发生变化时拒绝创建 revision。
当前只接受 `apiVersion: v2` 的 application Chart、UTF-8 文本文件和 4 MiB 以内的确定性快照。
`values.yaml` 中非空的 password、token、apiKey 等敏感默认值会被拒绝，环境密钥仍必须通过凭据引用注入。
工作流 Worker 已注册官方 `helm.sh/helm/v3 v3.18.5` provider。执行 `helm_deploy` 时先用官方引擎渲染并生成不含 Secret 明文的资源清单快照，再执行等价于 `upgrade --install --atomic --wait --wait-for-jobs` 的发布；成功后记录 Helm release revision。
同一个工作流节点使用稳定幂等键，Worker 在 Helm 已成功但任务状态尚未回写时崩溃，可通过 release description 恢复执行结果，不会重复创建 revision。

### 主机服务 Git 导入

| 方法 | 路径 | 权限 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/services/{id}/host-import/preview` | `service:view`、`code_source:view`、`credential:view` | 校验主机定义和脚本引用，返回冻结前预览 |
| `POST` | `/api/v1/services/{id}/host-import` | 上述读取权限 + `service:update` | 校验预览 commit 后创建不可变 `host_spec` revision |

```json
{
  "code_source_id": 10,
  "repo_path": "payment/backend",
  "ref": "main",
  "spec_path": "deploy/host/service.yaml",
  "paths": [
    "deploy/host/service.yaml",
    "deploy/host/scripts/deploy.sh",
    "deploy/host/scripts/health.sh",
    "deploy/host/scripts/rollback.sh"
  ],
  "expected_commit": "abc123"
}
```

`service.yaml` 必须使用 `axiops.io/v1alpha1` 的 `HostService`，目标 OS 只允许 `linux` 或 `windows`。
Linux 脚本必须为 `.sh`，Windows 脚本必须为 `.ps1`；deploy、health、rollback 三个脚本均从同一固定 Git commit 读取并冻结到 revision，导入请求不能提交内联脚本。
制品来源可选择同一工作流的成功构建，或提供固定的 HTTPS URL 和 64 位 SHA-256。预览只返回脚本路径、大小和摘要，不返回脚本正文。

主机定义可配置滚动策略；`batchSize` 与 `batchPercent` 二选一，未设置时默认每批 1 台，批次间隔范围为 0-3600 秒：

```yaml
apiVersion: axiops.io/v1alpha1
kind: HostService
metadata:
  name: payment
spec:
  os: linux
  artifact:
    source: build
  scripts:
    deploy: scripts/deploy.sh
    health: scripts/health.sh
    rollback: scripts/rollback.sh
  timeoutSeconds: 900
  rollout:
    batchSize: 5
    intervalSeconds: 30
```

首批以外的命令在数据库中保持 `blocked`，当前批全部成功后才原子释放下一批。任一 Agent 返回 `failed` 或 `rolled_back` 时，尚未领取的命令转为 `cancelled`；已领取命令仍允许完成并保存日志，工作流整体立即进入失败状态。

## 环境与变量 API

项目环境、默认环境变量组、环境克隆、服务覆盖和发布前资源预览均已实现。

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/projects/{id}/environments` | `environment:view` | 按项目或环境 ID 授权范围列出环境 |
| `POST` | `/api/v1/projects/{id}/environments` | `environment:create` | 创建项目环境；按项目类型绑定 Namespace 或主机组 |
| `GET` | `/api/v1/environments/{id}` | `environment:view` | 查询单个环境，不越过环境范围授权 |
| `GET` | `/api/v1/environments/{id}/variable-set` | `environment:view` | 查询默认环境变量组，Secret 只返回凭据 ID |
| `PUT` | `/api/v1/variable-sets/{id}` | `environment:update` | 使用 revision 乐观锁替换变量项 |
| `GET` | `/api/v1/environments/{id}/services/{serviceID}/override` | `environment:view` + `service:view` | 查询环境内服务目标版本 |
| `PUT` | `/api/v1/environments/{id}/services/{serviceID}/override` | `environment:update` + `service:view` | 使用 expected_revision 保存目标 revision、镜像和副本数 |
| `POST` | `/api/v1/environments/{id}/clone` | `environment:create` | 克隆非生产环境配置，不复制 Secret 实体 |
| `GET` | `/api/v1/environments/{id}/render-preview?service_id={serviceID}&service_revision_id={revisionID}&release_name={name}` | `environment:view` + `service:view` + `cluster:view` | 对 YAML/Helm 执行集群 Server-Side dry-run，只返回资源身份、动作和脱敏 values |

`service_id` 必填；`service_revision_id` 可选，未提供时依次使用环境服务覆盖 Revision 和服务当前 Revision。
Helm 的 `release_name` 可选，默认使用服务编码。接口最长执行 45 秒，且会再次验证环境、服务、Revision、
项目和集群归属。YAML 返回 `create/update/unchanged`；当前 YAML SSA 发布不做 prune，因此不会虚构
`delete`。Helm 在集群 dry-run 结果之外，会比较同一 Release 的上一条成功快照并返回 `delete`。

响应示例：

```json
{
  "delivery_type": "helm",
  "project_id": 1,
  "environment_id": 2,
  "service_id": 3,
  "service_revision_id": 4,
  "cluster_id": 5,
  "namespace": "payment-test",
  "release_name": "payment-api",
  "changes": [
    {
      "api_version": "apps/v1",
      "kind": "Deployment",
      "namespace": "payment-test",
      "name": "payment-api",
      "action": "update"
    }
  ],
  "summary": { "create": 0, "update": 1, "delete": 0, "unchanged": 0 },
  "values": { "database": { "password": { "credential_id": 7 } } }
}
```

Helm Secret 只在服务端内存中参与渲染和 Kubernetes dry-run；响应不返回 Secret 清单、运行时明文、
凭据密文或完整 rendered manifest。上一成功快照必须通过 SHA-256 校验，损坏时拒绝返回不完整 Diff。

环境创建时必须验证资源授权：YAML/Helm 只能绑定有权限的集群和 Namespace；主机项目只能绑定有权限的主机组。生产环境标记只能由具备环境配置权限的运维角色设置。

YAML/Helm 环境请求示例：

```json
{
  "name": "测试环境",
  "code": "test",
  "cluster_id": 2,
  "namespace": "payment-test",
  "is_production": false,
  "deploy_approval": false
}
```

项目类型决定环境类型，客户端不能直接提交 `environment_type`。集群必须处于 `connected`，并且当前用户同时拥有项目环境创建权限和目标集群查看权限。生产环境强制开启发布审批，且必须具备 `environment:manage_production`；项目管理员不能自行把环境标记为生产。

主机项目环境当前接收 `host_group_ref`，但状态固定为 `pending_validation`。在统一主机组注册中心和授权校验完成前，工作流不得向该环境部署。

变量更新请求必须携带当前 revision：

```json
{
  "revision": 1,
  "items": [
    {"key": "IMAGE_TAG", "value": "latest", "is_secret": false},
    {"key": "DB_PASSWORD", "credential_id": 7, "is_secret": true}
  ]
}
```

Secret 项禁止传 `value`，只能引用当前项目可用且用户有权查看的凭据。服务端不返回 Secret 明文；revision 不匹配表示配置已被其他用户更新，客户端必须重新读取后再提交。

首次创建服务覆盖时 `expected_revision` 为 0，后续更新必须使用查询返回的 revision：

```json
{
  "expected_revision": 0,
  "service_revision_id": 12,
  "image_ref": "registry.example.com/payment@sha256:abc123",
  "replicas": 2
}
```

环境与服务必须属于同一项目，服务 revision 必须属于该服务，服务类型必须与环境类型一致。未验证的主机环境不能保存覆盖；已存在覆盖在环境失效后仍可只读查询用于审计。

## 工作流 API（目标）

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `POST` | `/api/v1/projects/{id}/workflows` | `workflow:create` | 创建声明式 DAG |
| `PUT` | `/api/v1/workflows/{id}` | `workflow:update` | 更新定义，保存前校验无环和类型兼容性 |
| `POST` | `/api/v1/workflows/{id}/validate` | `workflow:view` | 返回节点依赖、权限、变量和 provider 校验结果 |
| `POST` | `/api/v1/workflows/{id}/runs` | `workflow:run` | 创建运行快照和首个命令；可选 `service_id` + `service_revision_id` 进行受控的运行时版本覆盖，服务端校验项目/交付类型/权限 |
| `GET` | `/api/v1/workflow-runs/{id}` | `workflow:view` | 获取 DAG 节点、日志索引、制品和发布结果 |
| `POST` | `/api/v1/workflow-runs/{id}/approve` | `workflow:approve` | 完成生产审批节点 |
| `POST` | `/api/v1/workflow-runs/{id}/cancel` | `workflow:cancel` | 请求取消未完成节点 |

当前实现状态：工作流 DAG 的服务端模型、定义持久化 API、运行快照 API、调度器和 worker 租约已完成，覆盖节点唯一性、依赖存在性、环检测、超时/重试边界，以及 YAML/Helm/主机部署节点与项目类型的兼容性。构建、YAML、Helm 和主机 executor 均已接入；前端画布仍按实施计划推进，画布 JSON 不能绕过服务端校验直接执行。

运行 API 已落地运行快照：创建运行时会保存 `definition_version`、环境上下文和每个节点的不可变快照。生产环境必须包含 `approval` 节点，运行初始状态为 `pending_approval`；取消只接受 `pending`、`pending_approval` 或 `running` 状态。

调度器先根据快照计算 `runnable`、`waiting`、`skippable` 和 `complete`，再由 worker 调用 provider。依赖失败的节点不会被执行，运行重启后可以重复计算计划，不依赖前端状态。

worker 通过 `workflow_task_run` 的租约字段领取任务；同一节点在租约有效期内只能由一个 worker 执行，worker 崩溃后允许租约过期重领。executor 按节点类型注册，未注册的节点会被标记为失败，不会静默跳过。

`build` 节点通过适配器调用已有构建队列，不直接重复创建 Kubernetes Job。节点配置至少包含 `service_id`，可选 `branch` 和 `commit_id`；任务保存 `provider_run_id=build:{id}` 与日志地址，轮询 `build_run` 终态后才将节点标记成功、失败或取消。

构建日志继续使用 `GET /api/v1/builds/{id}/logs` 和 `build_log:view` 范围授权。worker 只采集 builder 容器，
单次保存上限 1 MiB，`truncated=true` 表示已截断；相同构建阶段使用稳定 `log_key` 幂等更新。该接口当前为
日志快照轮询，不承诺流式推送语义。构建结果由 builder 写入 `AXIOPS_RESULT_FILE=/dev/termination-log`：

```json
{"image_digest":"sha256:<64 hex>","image_size":123,"artifact_url":"https://artifacts.example.com/app.tgz","artifact_size":456}
```

worker 只接受合法 SHA-256 和 `https/s3/oss/cos` 制品 URI。仓库 Token 从 `code_source.credential_id` 引用的
统一 `git_token` 凭据短时解密，写入构建 Namespace 的临时 Secret；Job 仅保存 `secretKeyRef`，终态或取消后删除 Secret。

## 代码源 API

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/code-sources?project_id={id}` | `code_source:view` | 按项目范围列出代码源元数据 |
| `POST` | `/api/v1/projects/{id}/code-sources` | `code_source:create` + `credential:view` | 使用统一 `git_token` 凭据引用创建代码源 |
| `GET` | `/api/v1/code-sources/{id}` | `code_source:view` | 查询详情，不返回 Token |
| `PUT` | `/api/v1/code-sources/{id}` | `code_source:update` + `credential:view` | 更新 URL、凭据引用或状态 |
| `DELETE` | `/api/v1/code-sources/{id}` | `code_source:delete` | 未被服务引用时删除 |
| `POST` | `/api/v1/code-sources/{id}/check` | `code_source:test` | 真实请求仓库列表，校验地址、凭据和网络连通性 |
| `GET` | `/api/v1/code-sources/{id}/repositories` | `code_source:view` | 查询可用仓库元数据，不返回凭据 |
| `GET` | `/api/v1/code-sources/{id}/repository-files?repo_path={path}&ref={ref}&path={dir}` | `code_source:view` | 查询仓库单层目录，供服务 YAML 导入浏览 |
| `GET` | `/api/v1/code-sources/{id}/branches` | `code_source:view` + `credential:view` | 查询仓库分支 |
| `GET` | `/api/v1/code-sources/{id}/tags` | `code_source:view` + `credential:view` | 查询仓库标签 |
| `GET` | `/api/v1/code-sources/{id}/commits` | `code_source:view` + `credential:view` | 查询指定 ref 的提交 |

代码源只保存 `credential_id`，API 不接受 Token 字段。URL 禁止内嵌用户信息；内网 Git 服务可使用 HTTP，公网建议 HTTPS。GitLab provider
已有仓库查询实现；GitHub、Gitee、Gitea 当前明确返回未支持，不能在端到端验收前标记为可用。
`check` 当前返回 `configuration_valid`，表示配置、权限和凭据格式有效，不代表已完成远端网络探测。

`yaml_deploy` 节点必须引用同项目的 `service_id` 和不可变 `service_revision_id`。服务端从项目环境读取目标集群和 Namespace，再次校验环境为 active/validated、服务类型为 `k8s_yaml`。MVP 拒绝 Secret、Namespace、集群级资源和未列入白名单的 Kind，资源显式 Namespace 必须与目标环境一致。

Kubernetes provider 使用 client factory 获取目标集群客户端，按资源执行 Server-Side Apply，并使用 `fieldManager=axiops-cicd` 和稳定幂等键。预览通过 provider 的 dry-run/差异接口返回资源级 `create/update/unchanged` 结果；Helm 结合上一成功 Release 快照补充 `delete`。kubeconfig 仅存在于 client factory，不进入数据库或日志。

YAML executor 在 apply 前将已验证清单写入 `deployment_snapshot`，状态为 `prepared`；同步或异步 apply 成功后才更新为 `applied`。快照保存 SHA-256，后续回滚必须校验摘要且只能使用 `applied` 快照。当前已完成快照写入基础，回滚 HTTP API 尚未开放。

节点类型和项目类型必须匹配：`yaml_deploy` 仅用于 `k8s_yaml`，`helm_deploy` 仅用于 `helm`，`host_deploy` 仅用于 `host`。生产部署节点前必须存在满足策略的审批节点。

## 发布和回滚 API（目标）

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/environments/{id}/deployments` | `environment:view` | 查询环境服务版本和发布历史 |
| `POST` | `/api/v1/deployment-snapshots/{id}/rollback` | `deployment:rollback` | 依据已应用 provider 快照创建系统工作流回滚 |
| `GET` | `/api/v1/deployments/{id}/diff` | `deployment:view` | 返回 YAML/Helm/主机版本差异 |

回滚不得接受客户端直接上传的清单、values 或脚本；必须从已审计的 `deployment_snapshot` 或历史 revision 派生。YAML 回滚使用已校验清单快照；Helm 回滚使用快照中的 release name 和 Helm revision，并继续复用生产审批、资源范围授权和幂等执行记录。

YAML 回滚接口已实现。请求必须携带 8-128 字符的 `Idempotency-Key`，相同用户、来源快照和请求键只创建
一个运行并返回同一 `workflow_run_id`；请求键被其他用户或快照占用时拒绝。服务端复核快照为
`applied`、provider 为 `yaml_deploy`、SHA-256 一致，并执行项目/服务/环境/集群范围的
`deployment:rollback` 授权。生产环境自动生成审批节点，沿用 `workflow:approve` 权限和运行审批 API。

旧 `/api/v1/releases/rollback` 只服务于早期 `release_record` 兼容模型，不具备新工作流、快照和多交付类型
语义；新前端和新调用方不得使用该接口，待历史流量核对后按下线计划移除。

## 环境克隆实现约束

`POST /api/v1/environments/{id}/clone` 同时要求源环境 `environment:view` 和目标 `environment:create`。源环境不能是生产环境，目标请求也不能创建生产环境；K8s 目标会重新检查集群为 `connected` 且用户拥有目标集群查看权限。事务只复制非 Secret 变量和不可变服务 revision 覆盖，Secret 值及 credential 关联不会复制。

## 构建治理与产物校验

| 方法 | 路径 | 权限动作 | 说明 |
| --- | --- | --- | --- |
| `GET` | `/api/v1/build-metrics` | `build:view` | 按授权项目/服务查询成功率、耗时和每日趋势 |
| `GET` | `/api/v1/build-concurrency` | `build_governance:view` | 查询平台或项目并发与排队超时策略 |
| `PUT` | `/api/v1/build-concurrency` | `build_governance:update` | 保存平台或项目覆盖策略 |

构建日志可通过 MinIO 归档，数据库只保留 64 KiB 尾部。API 在完成 `build_log:view` 鉴权后读取完整对象，凭据仅从环境变量读取：

```text
AXIOPS_BUILD_LOG_ENDPOINT
AXIOPS_BUILD_LOG_BUCKET
AXIOPS_BUILD_LOG_ACCESS_KEY
AXIOPS_BUILD_LOG_SECRET_KEY
AXIOPS_BUILD_LOG_USE_SSL
AXIOPS_BUILD_LOG_PREFIX
```

主机制品在构建成功前通过 HTTPS 流式计算 SHA-256 和实际大小；校验结果写入 `artifact_verification_status`，主机发布只接受 `verified`。镜像 manifest digest 校验通过 `AXIOPS_BUILD_VERIFY_IMAGE=true` 开启，使用服务关联的 `image_registry` 和加密 `harbor_auth/registry_auth` 凭据。最大制品校验大小可通过 `AXIOPS_BUILD_VERIFY_MAX_ARTIFACT_BYTES` 调整，默认 2 GiB。
## Platform code sources and repositories

Code sources are reusable Git connections. A row with `project_id=0` is
platform-scoped; project rows remain isolated. Project users with an allowed
`code_source:view` scope may consume global sources, while global mutations
require platform scope.

| Method | Path | Permission | Description |
| --- | --- | --- | --- |
| `GET` | `/api/v1/code-sources?project_id=0` | `code_source:view` | List platform sources |
| `POST` | `/api/v1/code-sources` | `code_source:create` | Create a platform source |
| `GET` | `/api/v1/code-sources/{id}/repositories` | `code_source:view` | List repositories |
| `POST` | `/api/v1/code-sources/{id}/check` | `code_source:test` | Make a real repository-list request |

Supported types are `gitlab`, `gitea`, `github` and `gitee`. Source URLs may be
HTTP or HTTPS, which is required for private networks without TLS. Credentials
are encrypted server-side and are never returned. Repository metadata contains
no token or credential payload.

The project service form consumes the repository list and writes the selected
clone URL and default branch into the service request. The legacy manual URL
field remains available for existing service records.
