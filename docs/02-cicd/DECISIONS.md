# CI/CD 执行方案决策记录

> 基于运维便捷性、标准化、快速交付的需求分析

## 一、核心诉求

1. **运维便捷**: 无需每次手动创建 Job，配置一次，多环境自动适配
2. **环境灵活**: 开发/测试/预发/生产环境可按需启用，不强制四环境
3. **批量操作**: 支持批量发布、选择性发布
4. **快速交付**: 从代码提交到发布上线，流程自动化、速度快

## 二、方案对比

### 方案 A：Jenkins Provider（已有代码）

**工作流程**：
```
运维在 AxiOps 创建服务配置
  → AxiOps 自动调用 Jenkins API 创建/更新 Job
  → 用户触发构建
  → AxiOps 调用 Jenkins 构建
  → Jenkins 执行（拉代码、编译、构建镜像、推送）
  → AxiOps 获取结果和制品信息
  → 发布到 K8s（AxiOps 控制）
```

**优势**：
- ✅ Jenkins 生态成熟，构建环境丰富
- ✅ 已有代码基础
- ✅ 可通过 Jenkins Job Template + Parameterized Build 实现自动化

**痛点**：
- ❌ **Jenkins Job 创建慢**: 虽然可以 API 自动创建，但 Jenkins 本身较重
- ❌ **多环境复杂**: 需要为每个环境创建不同 Job，或用参数化构建
- ❌ **依赖外部系统**: Jenkins 故障影响构建
- ❌ **运维负担**: Jenkins 本身需要维护、升级、监控

**改进方案**：
1. AxiOps 自动管理 Jenkins Job（用户无感知）
2. 使用 Jenkins Job DSL 或 JobBuilder 模板化创建
3. 一个服务对应一个 Parameterized Job，环境通过参数传递

---

### 方案 B：K8s Job + 自研 Agent（推荐）

**工作流程**：
```
运维在 AxiOps 创建服务配置（指定代码源、分支、构建脚本、环境）
  → 用户触发构建
  → AxiOps 创建 K8s Job（带构建镜像）
  → Job 内执行：
      1. git clone 代码
      2. 执行构建脚本（Maven/Gradle/npm/go build）
      3. 构建 Docker 镜像
      4. 推送到 Harbor
      5. 回传制品信息（image digest、tag）
  → AxiOps 发布到目标环境 K8s
```

**优势**：
- ✅ **零配置**: 运维只需在 AxiOps 配置一次，无需操作 Jenkins
- ✅ **环境灵活**: 多环境只是数据库配置，无需重复创建 Job
- ✅ **云原生**: K8s Job 自动调度、弹性扩展、资源隔离
- ✅ **批量发布**: 直接操作数据库和 K8s API，支持批量、并发、选择性发布
- ✅ **快速交付**: 无 Jenkins 中间层，构建触发更快
- ✅ **统一管理**: 构建和发布都在 AxiOps 内，审计、监控、日志统一

**挑战**：
- ⚠️ 需要维护构建镜像（Java/Go/Node/Python 基础镜像 + 工具链）
- ⚠️ 需要实现日志流式回传
- ⚠️ 需要处理构建超时、失败重试

**实现细节**：
1. **构建镜像**:
   - `axiops/builder-java:17` - Maven 3.9 + JDK 17
   - `axiops/builder-go:1.25` - Go 1.25
   - `axiops/builder-node:22` - Node 22 + pnpm
   - `axiops/builder-python:3.12` - Python 3.12 + pip
   - 镜像构建使用 Kaniko 或 Rootless BuildKit，禁止挂载宿主机 Docker Socket

2. **K8s Job 模板**:
```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: build-<service-id>-<build-id>
  namespace: axiops-cicd
spec:
  template:
    spec:
      automountServiceAccountToken: false
      initContainers:
      - name: git-clone
        image: alpine/git:v2.47.2
        volumeMounts:
        - name: workspace
          mountPath: /workspace
      containers:
      - name: builder
        image: axiops/builder-<language>:<version>
        volumeMounts:
        - name: workspace
          mountPath: /workspace
      volumes:
      - name: workspace
        emptyDir: {}
      restartPolicy: Never
  backoffLimit: 0
```

代码仓库和镜像仓库凭据不得写入环境变量或 Job manifest 明文。worker 只创建短生命周期
Secret 引用，Job 完成后清理；构建容器不获得 Kubernetes API Token。

3. **日志回传**:
   - K8s Job 实时 tail logs
   - 通过 WebSocket 推送到前端
   - 持久化到 AxiOps 数据库或对象存储

4. **构建脚本示例**（Java Maven）:
```bash
#!/bin/bash
set -e

# 1. 克隆代码
git clone -b $GIT_BRANCH $GIT_URL /workspace
cd /workspace

# 2. 编译构建
mvn clean package -DskipTests

# 3. 由独立的 Kaniko/Rootless BuildKit 步骤构建并推送镜像
# 4. worker 从构建器结构化结果中读取不可变 image digest
```

---

## 三、多环境管理

### 3.1 环境模型

当前数据库决策：
```sql
CREATE TABLE project_environment (
  id BIGINT PRIMARY KEY,
  project_id BIGINT,
  name VARCHAR(64),
  code VARCHAR(32),
  environment_type VARCHAR(24), -- k8s_namespace/host_group
  cluster_id BIGINT,
  namespace VARCHAR(63),
  host_group_ref VARCHAR(128),
  binding_status VARCHAR(24),   -- validated/pending_validation/invalid
  is_production BOOLEAN,
  deploy_approval BOOLEAN
);

CREATE TABLE variable_set (
  id BIGINT PRIMARY KEY,
  project_id BIGINT,
  environment_id BIGINT,
  service_id BIGINT,
  scope VARCHAR(16),
  revision BIGINT
);

CREATE TABLE variable_item (
  id BIGINT PRIMARY KEY,
  variable_set_id BIGINT,
  variable_key VARCHAR(128),
  value_text MEDIUMTEXT,
  credential_id BIGINT,
  is_secret BOOLEAN
);
```

不再维护一张全局环境字典：`dev/test/prod` 是项目自己的运行上下文。服务副本、资源限制和版本覆盖
后续进入 `environment_service_override`，不回退到 JSON `env_vars`。真实字段、索引和权限初始化以
`migrations/009_cicd_project_environment.sql` 为准。

### 3.2 配置示例

**场景 1: 项目只有开发和生产（无预发布）**
```json
{
  "project_id": 1,
  "environments": [
    {
      "env_id": 1,  // dev
      "enabled": true,
      "cluster_id": 10,
      "namespace": "project-dev",
      "auto_deploy": true,
      "require_approval": false
    },
    {
      "env_id": 4,  // prod
      "enabled": true,
      "cluster_id": 20,
      "namespace": "project-prod",
      "auto_deploy": false,
      "require_approval": true
    }
  ]
}
```

**场景 1a: 项目只有测试和生产（无开发、无预发布）**

这是受支持的最小生产链。项目只创建 `test` 和 `prod` 两条环境记录，测试成功后才能将同一个不可变 artifact 晋级到生产：

```text
build once -> test deploy/verify -> production approval -> prod deploy
```

平台不会隐式创建 `dev`、`stage` 或其他占位环境；环境列表、运行目标和权限范围只展示该项目实际配置且用户有权访问的环境。

**场景 2: 项目有完整四环境**
```json
{
  "project_id": 2,
  "environments": [
    { "env_id": 1, "enabled": true },  // dev
    { "env_id": 2, "enabled": true },  // test
    { "env_id": 3, "enabled": true },  // staging
    { "env_id": 4, "enabled": true }   // prod
  ]
}
```

### 3.3 发布流程

**单环境发布**:
```
构建完成 → 选择环境 → 检查权限 → 检查审批 → 执行发布
```

**批量发布**（开发环境到测试环境批量推进）:
```
前端勾选多个服务 → 后端并发执行发布 → 实时返回进度
```

**选择性发布**:
```
运维选择 10 个服务中的 3 个 → 只发布选中的 3 个
```

---

## 四、最终决策

### 推荐方案：**K8s Job + 自研 Agent**

**理由**：
1. **运维体验最佳**: 在 AxiOps 一个界面完成所有配置，无需操作 Jenkins
2. **环境灵活**: 数据库配置即可，增删改环境无需创建 Job
3. **批量发布**: 直接操作 K8s API，支持并发、批量、选择性
4. **快速交付**: 无 Jenkins 中间层，构建触发延迟低
5. **云原生**: 利用 K8s 调度、资源隔离、弹性扩展
6. **长期维护成本低**: 不需要维护 Jenkins 集群

### 实施路径

**Phase 1（MVP - 2 周）**:
- ✅ 实现 K8s Job Builder（model/repository/logic）
- ✅ 实现 Java/Go/Node.js 三种语言构建模板
- ✅ 实现日志流式传输
- ✅ 前端显示构建日志（WebSocket）
- ✅ 发布到 K8s（使用 client-go）

**Phase 2（优化 - 1 周）**:
- 构建镜像优化（缓存依赖、加速构建）
- 支持自定义 Dockerfile
- 支持 Helm Chart 发布
- 支持蓝绿发布、金丝雀发布

**Phase 3（高级 - 后续）**:
- 流水线编排（多阶段、并行、依赖）参考 zadig
- 构建缓存优化（Maven 依赖、npm node_modules）
- 分布式构建（多集群调度）
- Host Agent（非 K8s 环境构建）

### Jenkins Provider 保留策略

**保留但降级为可选项**：
- 如果用户已有 Jenkins 环境且不想迁移，可配置使用
- AxiOps 通过 Jenkins API 自动创建/更新 Job（用户无感知）
- 优先推荐自研方案，Jenkins 作为兼容方案

---

## 五、对比总结

| 维度 | Jenkins Provider | K8s Job + 自研 Agent |
|------|-----------------|---------------------|
| 运维便捷性 | ⭐⭐ 需配置 Jenkins | ⭐⭐⭐⭐⭐ 只配置 AxiOps |
| 多环境灵活性 | ⭐⭐ 需创建多个 Job | ⭐⭐⭐⭐⭐ 数据库配置 |
| 批量发布 | ⭐⭐ 需循环调用 API | ⭐⭐⭐⭐⭐ K8s 原生并发 |
| 快速交付 | ⭐⭐⭐ 有 Jenkins 延迟 | ⭐⭐⭐⭐⭐ 直接执行 |
| 维护成本 | ⭐⭐ 需维护 Jenkins | ⭐⭐⭐⭐ 只维护构建镜像 |
| 生态成熟度 | ⭐⭐⭐⭐⭐ 非常成熟 | ⭐⭐⭐ 需自研部分功能 |
| 云原生 | ⭐⭐ 传统架构 | ⭐⭐⭐⭐⭐ K8s 原生 |

**结论**: 自研方案在运维体验、灵活性、云原生方面全面领先，推荐作为主力方案。

---

## 六、项目类型和领域模型决策

### 决策

采用 `项目 -> 服务 -> 环境 -> 工作流` 四层模型；项目在创建时选择且固定为 `k8s_yaml`、`helm` 或 `host` 三种交付类型。MVP 不提供混合项目。

### 原因

- YAML、Helm 和主机的服务解析、变量渲染、部署幂等和回滚依据不同，混用会让权限、前端和工作流节点产生大量隐式分支。
- 项目类型固定后，服务导入、环境向导和工作流模板可以做到低配置自服务。
- 需要跨类型编排时，后续使用“编排项目”引用多个标准项目，不破坏三种基础模型。

### 约束

- 旧项目统一标记为 `unclassified`，必须人工确认，不根据旧 `deploy_type` 自动推断。
- `app_service.deploy_type` 和 `release_record` 是兼容骨架，不作为新功能的主模型。
- 前端必须从项目内进入服务、环境和工作流；不能提供跨项目的无范围发布入口。

## 七、工作流和执行器决策

工作流定义采用声明式 DAG，不将前端画布作为唯一事实来源。服务端保存节点、依赖、输入、超时、重试、权限和版本快照，前端只负责可视化编辑与运行观察。

执行面分离：K8s 构建任务由 `kube_job` provider 默认执行，受限网络或固定工具链场景由独立的 `ecs_agent` 构建 provider 执行；主机部署任务仍由部署 Agent 主动领取。Tekton 不进入当前主干，仅作为未来多步骤编排的可选适配器。平台 API 不直接持有集群管理员 kubeconfig、SSH 私钥或执行任意 Shell。所有 provider 必须按幂等键提交，并回传结构化状态、日志索引和制品/部署快照。

构建输出统一遵循 `axiops.build/v1` 结果协议：容器构建必须回传不可变 `image_digest`，非容器化遗留 JAR 可回传 HTTPS 制品 URL 和 `artifact_sha256`。构建输入由服务、固定 commit、分支和构建配置生成 SHA-256 指纹；同一指纹在 dev/qa/prod 复用，不重复构建。
