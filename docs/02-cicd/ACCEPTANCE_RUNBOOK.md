# AxiOps CI/CD 验收手册

本文用于本地开发环境的权限隔离与三类交付流程验收。验收账号和示例项目只用于测试，生产上线前必须删除或修改密码。

## 1. 启动与版本检查

在仓库根目录执行：

```powershell
cd E:\axiops
.\start.cmd
go run ./cmd/axiops-migrate -f configs/platform-api.yaml -check-only
```

预期迁移结果：

```text
current=028_access_department_workflow_scope.sql applied=[]
```

访问 `http://127.0.0.1:3006`。若服务已经运行，不需要重复执行启动脚本。

查看本地进程和监听端口：

```powershell
Get-Process -Name axiops-platform-api,axiops-cicd-api,axiops-workflow-worker -ErrorAction SilentlyContinue |
  Select-Object Id,ProcessName,Path
Get-NetTCPConnection -State Listen -LocalPort 3006,8090,8091 -ErrorAction SilentlyContinue |
  Select-Object LocalPort,OwningProcess
```

停止服务执行 `.\stop.cmd`。停止脚本会额外清理本仓库遗留的 worker 进程，避免重复消费任务。

## 2. 权限隔离验收

| 身份 | 用户名 | 密码 | 预期项目范围 | 权限中心 |
|---|---|---|---|---|
| 管理员 | `admin` | `REPLACE_ME` | YAML、Helm、主机全部项目 | 可见 |
| 开发 | `axiops_dev` | `REPLACE_ME` | 仅 `axiops-demo-yaml` | 不可见 |
| 测试 | `axiops_test` | `REPLACE_ME` | 仅 `axiops-demo-helm` | 不可见 |
| 运维 | `axiops_ops` | `REPLACE_ME` | YAML、Helm、主机全部项目 | 不可见 |

自动回归命令：

```powershell
powershell -ExecutionPolicy Bypass -File scripts/test-access-acceptance.ps1
```

脚本只读取数据，验证项目列表、菜单和开发账号跨项目访问返回 HTTP 403。通过时最后输出：

```text
AxiOps authorization acceptance passed.
```

页面验收时还需确认：开发账号只能查看构建、日志、环境和工作流，不能新建服务、修改构建配置或执行工作流；测试账号只进入 Helm 项目；运维账号具备交付操作权限。

### 测试/生产两环境项目

创建仅包含 `test`、`prod` 的项目并完成以下检查：

- 项目环境列表只显示 `test` 和 `prod`，不会出现开发或预发布占位项。
- 工作流运行目标只允许选择已验证的 `test`、`prod` 环境。
- 首次构建生成一个不可变 digest；测试部署成功后，生产只能选择该 digest，不能输入任意 tag。
- 生产运行进入审批状态；审批拒绝不会创建生产部署 Job，审批通过后才创建目标 Job。
- 测试和生产可绑定不同集群/Namespace，权限按项目、环境和集群范围分别校验。

## 3. 当前验收数据

| 类型 | 项目 | 项目 ID | 服务 ID | 当前状态 |
|---|---|---:|---:|---|
| K8s YAML | `axiops-demo-yaml` | 4 | 5 | 项目和服务已创建，等待真实代码源与集群 |
| Helm Chart | `axiops-demo-helm` | 2 | 3 | 项目和服务已创建，等待真实 Chart 与集群 |
| 主机应用 | `axiops-demo-host` | 3 | 4 | 项目和服务已创建，等待主机组与 Agent |

ID 仅描述当前本地验收库，不应写入业务代码。自动化和外部集成应使用项目、服务编码查询资源。

## 4. YAML 流程验收

### 4.1 先接入 Kubernetes 集群

1. 使用 `admin / REPLACE_ME` 登录 AxiOps，打开“集群管理”。
2. 点击“接入集群”，粘贴专用测试 kubeconfig。该 kubeconfig 应使用测试集群 ServiceAccount，并确保 API Server 能从运行 `workflow-worker` 的机器访问；不要使用生产 `cluster-admin`。
3. 页面校验 YAML 格式并自动填充 API Server。保存后等待状态变为“已连接”，确认版本、节点数、CPU 和内存指标。
4. 在“权限中心 -> 资源授权”中选择范围角色、授权对象和“集群”，绑定到指定集群；使用开发/测试账号确认只能看到被授权集群。
5. 删除集群前，先确认它未被任何 CI/CD 环境或发布记录引用。被引用时删除必须失败；解除引用后才允许删除。

项目隔离补充：进入某个 YAML/Helm 项目工作区创建环境时，集群下拉框按当前项目查询；项目成员只能看到该项目已引用或被明确绑定的集群，不能通过切换项目读取其他项目集群。

前置资源：可访问的 Git 仓库、镜像仓库、状态为 `connected` 的 Kubernetes 集群，以及目标 Namespace。当前本地数据库尚未接入集群，不能把部署 E2E 标记为通过。

验收顺序：

1. 管理员进入 `axiops-demo-yaml`，在“代码源”创建连接并执行校验。
2. 在“服务”绑定代码源，导入包含 `Deployment`、`Service` 等白名单资源的 YAML，确认预览后生成不可变 revision。
3. 保存构建脚本、Dockerfile、镜像名称和构建资源限制。
4. 创建非生产环境，选择集群和 Namespace，保存变量组及服务 revision 覆盖。
5. 执行资源预览，确认 Namespace、镜像、变量渲染和服务端 Diff 正确。
6. 创建 `build -> yaml_deploy` 工作流，校验 DAG 后执行。
7. 在运行抽屉检查节点状态、实时日志、commit、镜像 digest 和失败原因。
8. 使用已应用快照执行回滚，确认幂等键、审计记录和最终资源版本。

## 5. Helm 流程验收

前置资源与 YAML 相同，Git 仓库中还必须包含 Helm v3 application Chart。

验收顺序：代码源校验、Chart/values 预览与导入、环境 values 覆盖、服务端渲染 Diff、`build -> helm_deploy` 执行、release 状态检查、`helm_rollback` 回滚。生产环境必须经过人工审批节点。

## 6. 主机流程验收

前置资源：已同步的主机资产、主机组、可下载的二进制制品地址，以及在线 AxiOps Agent。

验收顺序：导入主机服务定义、配置制品 SHA-256 和部署脚本、关联主机组、配置滚动批次与探活、执行工作流、检查 Agent 日志与批次状态、使用历史制品回滚。校验失败或探活失败必须停止后续批次。

## 7. 通知与最终验收

为项目创建飞书、钉钉、企业微信、邮件或通用 Webhook 订阅，至少验证一次成功和一次失败事件。通知应包含项目、服务、环境、触发人、commit、运行结果、失败原因和详情链接，投递结果可在项目“通知”页追踪。

最终完成标准：三种类型均通过构建、部署、日志、审批（生产）、通知、失败处理和回滚；四类账号重新执行权限矩阵；前后端测试及生产构建通过；`PROGRESS.md` 记录结果和外部环境信息。

## 8. 已知数据风险

当前库存在历史服务 `app_service.id=1`、`project_id=1`、`code=test-svc`，其原项目已不存在。该记录暂不删除，避免误伤历史数据。后续应增加外键/软删除约束并完成孤儿数据核对；在核对完成前，不得复用项目 ID 1 作为验收数据。
