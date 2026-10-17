# CI/CD 模块详细设计

> 状态：方案确认  
> 日期：2026-08-08  
> 目标：大量用户并发构建、自动化高、稳定、构建快、操作少、日志清晰

## 1. 设计原则

- 用户创建构建任务不需要复杂配置，选项目、选服务、选环境，一键触发。
- 构建节点不能成为瓶颈，利用 K8s 原生调度能力支持大量并发。
- 构建日志实时可见，提交信息、变更文件、构建步骤一目了然。
- 构建缓存加速，不重复下载依赖。
- 构建节点出问题不影响其他构建，每个构建独立 Pod。
- Jenkins 只作为兼容选项，面向已有 Jenkins 的公司；新用户默认 K8s Job 方式。

## 2. 构建架构

```text
用户触发构建（手动 / Git Webhook / 审批通过 / 定时）
        |
        v
┌──────────────────────────────┐
│    AxiOps Pipeline Engine    │
│  （流水线编排、权限、审批）    │
└──────────────┬───────────────┘
               │ BuildSpec
               v
┌──────────────────────────────┐
│    Build Controller          │
│  （选择执行器、创建任务）      │
└──────┬───────────┬───────────┘
       │           │
       v           v
┌────────────┐  ┌──────────────┐
│K8s Job     │  │Host Agent    │
│（默认）     │  │（ECS/隔离网络）│
└─────┬──────┘  └──────┬───────┘
      │                │
      v                v
  K8s Pod           主机进程
  独立隔离           模板化执行
```

## 3. K8s Job 构建（默认方式）

### 3.1 为什么选 K8s Job

- 不需要维护额外系统（不需要 Jenkins master）
- 每个构建独立 Pod，天然隔离，节点出问题只影响一个构建
- 并发构建量 = K8s 集群可调度 Pod 数，轻松支持 100+ 并发
- 构建完自动清理，不留垃圾
- 日志实时流，不需要轮询

### 3.2 构建 Pod 结构

一个构建 = 一个 K8s Job，Pod 内按步骤执行：

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: build-{service}-{short-id}
  namespace: axiops-builds
  labels:
    axiops.io/project: "my-project"
    axiops.io/service: "user-api"
    axiops.io/pipeline-run: "run-12345"
spec:
  backoffLimit: 0
  activeDeadlineSeconds: 1800
  ttlSecondsAfterFinished: 300
  template:
    spec:
      nodeSelector:
        axiops.io/node-role: build
      restartPolicy: Never
      initContainers:
        - name: git-clone
          image: axiops/git-init:latest
          env:
            - name: REPO_URL
              value: "https://git.company.com/team/user-api.git"
            - name: GIT_REF
              value: "abc123def"
            - name: GIT_BRANCH
              value: "feature/login"
          volumeMounts:
            - name: workspace
              mountPath: /workspace
      containers:
        - name: build
          image: maven:3.9-eclipse-temurin-17
          workingDir: /workspace
          command: ["sh", "-c"]
          args:
            - |
              echo "========== Build Info =========="
              echo "Project: my-project"
              echo "Service: user-api"
              echo "Branch: feature/login"
              echo "Commit: abc123def"
              echo "Author: zhangsan"
              echo "Message: fix: 修复登录超时问题"
              echo "================================"
              echo ""
              echo "========== Changed Files =========="
              echo "M src/main/java/com/app/LoginService.java"
              echo "M src/main/java/com/app/AuthFilter.java"
              echo "A src/test/java/com/app/LoginServiceTest.java"
              echo "=================================="
              echo ""
              mvn clean package -DskipTests -s /cache/.m2/settings.xml
          volumeMounts:
            - name: workspace
              mountPath: /workspace
            - name: cache
              mountPath: /cache/.m2
          resources:
            requests:
              cpu: "1"
              memory: "2Gi"
            limits:
              cpu: "4"
              memory: "8Gi"
        - name: kaniko
          image: gcr.io/kaniko-project/executor:v1.23.0
          args:
            - --dockerfile=/workspace/Dockerfile
            - --context=/workspace
            - --destination=harbor.company.com/my-project/user-api:abc123def
            - --cache=true
            - --cache-repo=harbor.company.com/build-cache/user-api
          volumeMounts:
            - name: workspace
              mountPath: /workspace
            - name: docker-config
              mountPath: /kaniko/.docker
              readOnly: true
      volumes:
        - name: workspace
          emptyDir: {}
        - name: cache
          persistentVolumeClaim:
            claimName: build-cache-my-project
        - name: docker-config
          secret:
            secretName: axiops-registry-auth
```

### 3.3 构建节点池

```yaml
# 构建专用节点，和业务隔离
apiVersion: v1
kind: Node
metadata:
  labels:
    axiops.io/node-role: build
spec:
  taints:
    - key: axiops.io/build-only
      value: "true"
      effect: NoSchedule
```

构建节点建议：
- 高 CPU、大内存（8C16G 或以上）
- SSD 本地磁盘（构建 IO 快）
- 专用节点池，不和业务 Pod 混跑
- 可以用 Cluster Autoscaler 动态扩缩

### 3.4 构建缓存

每个项目一个 PVC，构建时挂载：

| 语言 | 缓存目录 | 效果 |
|------|----------|------|
| Java/Maven | /cache/.m2 | 不重复下载 jar |
| Java/Gradle | /cache/.gradle | 不重复下载依赖 |
| Node/npm | /cache/.npm | 不重复安装 node_modules |
| Node/pnpm | /cache/pnpm-store | pnpm 全局缓存 |
| Go | /cache/gomod | go mod 缓存 |
| Docker | Kaniko --cache-repo | 镜像层缓存 |

效果：首次构建 3-5 分钟，后续构建 30s-1min（只编译变更部分）。

## 4. 构建日志体验

参考 Jenkins Blue Ocean 的日志展示，但更清晰：

### 4.1 构建详情页展示内容

```text
┌─────────────────────────────────────────────────────┐
│ 构建 #47 - user-api                          成功 ✓ │
├─────────────────────────────────────────────────────┤
│                                                     │
│ 提交信息                                            │
│ ─────────                                           │
│ 分支: feature/login                                 │
│ 提交: abc123def                                     │
│ 作者: zhangsan                                      │
│ 时间: 2026-08-08 16:30:22                           │
│ 消息: fix: 修复登录超时问题                          │
│                                                     │
│ 变更文件 (3)                                        │
│ ─────────                                           │
│ M src/main/java/com/app/LoginService.java           │
│ M src/main/java/com/app/AuthFilter.java             │
│ A src/test/java/com/app/LoginServiceTest.java       │
│                                                     │
│ 构建阶段                     耗时                    │
│ ─────────                   ─────                   │
│ ✓ 拉取代码                   3s                     │
│ ✓ 编译打包                   28s                    │
│ ✓ 构建镜像                   15s                    │
│ ✓ 推送镜像                   8s                     │
│                                                     │
│ 制品                                                │
│ ─────                                               │
│ 镜像: harbor.company.com/my-project/user-api:abc123 │
│ Digest: sha256:a1b2c3d4...                          │
│ 大小: 145MB                                         │
│                                                     │
│ [查看完整日志]  [重新构建]  [发布到测试环境]          │
└─────────────────────────────────────────────────────┘
```

### 4.2 实时日志流

```text
构建日志（实时）
──────────────────────────────────────────
[16:30:25] ========== Build Info ==========
[16:30:25] Project: my-project
[16:30:25] Service: user-api
[16:30:25] Branch: feature/login
[16:30:25] Commit: abc123def
[16:30:25] Author: zhangsan
[16:30:25] Message: fix: 修复登录超时问题
[16:30:25] ================================
[16:30:25]
[16:30:25] ========== Changed Files ==========
[16:30:25] M src/main/java/com/app/LoginService.java
[16:30:25] M src/main/java/com/app/AuthFilter.java
[16:30:25] A src/test/java/com/app/LoginServiceTest.java
[16:30:25] ==================================
[16:30:25]
[16:30:26] [INFO] Scanning for projects...
[16:30:27] [INFO] Building user-api 1.0.0
[16:30:28] [INFO] --- maven-compiler-plugin:3.11.0:compile ---
[16:30:35] [INFO] Compiling 42 source files
...
[16:30:52] [INFO] BUILD SUCCESS
[16:30:52] [INFO] Total time: 26s
[16:30:53] Building image with Kaniko...
[16:31:05] Image pushed: harbor.company.com/my-project/user-api:abc123def
[16:31:08] Build completed successfully ✓
```

### 4.3 日志实现

```go
// 通过 K8s Pod Logs API 实时推送
func (p *KubeJobProvider) StreamBuildLog(ctx context.Context, runID string) (<-chan LogLine, error) {
    podName := p.getPodForRun(runID)
    
    // Follow pod logs
    req := p.client.CoreV1().Pods("axiops-builds").GetLogs(podName, &corev1.PodLogOptions{
        Follow:     true,
        Timestamps: true,
        Container:  "build",
    })
    
    stream, err := req.Stream(ctx)
    if err != nil {
        return nil, err
    }
    
    ch := make(chan LogLine, 100)
    go func() {
        defer close(ch)
        defer stream.Close()
        scanner := bufio.NewScanner(stream)
        for scanner.Scan() {
            ch <- parseLine(scanner.Text())
        }
    }()
    
    return ch, nil
}

// 前端通过 WebSocket 接收
// ws://axiops.company.com/api/v1/builds/{runID}/logs/stream
```

## 5. 用户操作流程（极简）

### 5.1 首次配置（一次性）

```text
1. 创建项目
2. 添加服务 -> 填写 Git 地址
3. 平台自动检测语言 -> 推荐构建模板
4. 确认或微调构建命令
5. 配置镜像仓库（或用平台默认仓库）
6. 完成
```

之后每次构建只需要：
- 手动：点「构建」按钮
- 自动：提交代码触发 Webhook
- 审批：审批通过自动触发

### 5.2 创建构建任务（用户视角）

```text
选择服务: user-api
选择分支: feature/login（下拉列表，实时从 Git 拉取）
选择环境: test（决定构建后发布到哪里）
高级选项（可选）:
  - 自定义构建命令
  - 跳过测试
  - 自定义镜像 Tag
  
[开始构建]
```

用户不需要：
- 不需要写 Jenkinsfile
- 不需要配置构建节点
- 不需要管理 Docker
- 不需要配置缓存
- 不需要登录另一个系统

## 6. Git 信息展示

每次构建自动记录并展示：

```go
type BuildGitInfo struct {
    Branch       string   `json:"branch"`
    Commit       string   `json:"commit"`
    CommitShort  string   `json:"commit_short"`
    Author       string   `json:"author"`
    AuthorEmail  string   `json:"author_email"`
    Message      string   `json:"message"`
    Timestamp    time.Time `json:"timestamp"`
    ChangedFiles []GitFile `json:"changed_files"`
    CompareURL   string   `json:"compare_url"` // Git 对比链接
}

type GitFile struct {
    Action string `json:"action"` // A(新增) M(修改) D(删除) R(重命名)
    Path   string `json:"path"`
}
```

获取方式：
- Webhook 触发：从 Webhook payload 中直接拿到
- 手动触发：git-init 容器执行 `git log` 和 `git diff --name-status` 获取

## 7. 构建模板（平台内置，用户不需要写）

### 7.1 语言检测

```go
func DetectLanguage(repoFiles []string) BuildTemplate {
    if contains(repoFiles, "pom.xml") {
        return MavenTemplate
    }
    if contains(repoFiles, "build.gradle") || contains(repoFiles, "build.gradle.kts") {
        return GradleTemplate
    }
    if contains(repoFiles, "package.json") {
        if contains(repoFiles, "pnpm-lock.yaml") {
            return PnpmTemplate
        }
        return NpmTemplate
    }
    if contains(repoFiles, "go.mod") {
        return GoTemplate
    }
    if contains(repoFiles, "Dockerfile") {
        return DockerOnlyTemplate
    }
    return GenericTemplate
}
```

### 7.2 内置模板

| 模板 | 构建镜像 | 默认命令 | 缓存 |
|------|----------|----------|------|
| Maven | maven:3.9-eclipse-temurin-17 | mvn clean package -DskipTests | /root/.m2 |
| Gradle | gradle:8-jdk17 | gradle build -x test | /root/.gradle |
| npm | node:20-alpine | npm ci && npm run build | /root/.npm |
| pnpm | node:20-alpine | pnpm install && pnpm build | /root/.local/share/pnpm |
| Go | golang:1.22-alpine | go build -o /output/app . | /go/pkg/mod |
| Python | python:3.12-slim | pip install -r requirements.txt | /root/.cache/pip |
| Docker Only | kaniko | (只打镜像) | --cache-repo |

用户也可以自定义模板或在服务级别覆盖。

## 8. Webhook 自动触发

```go
// POST /api/v1/webhooks/git
func HandleGitWebhook(ctx *gin.Context) {
    payload := parseWebhook(ctx) // 支持 GitLab / GitHub / Gitee / Gitea
    
    // 匹配服务（通过 Git URL）
    service := findServiceByRepo(payload.Repository.URL)
    
    // 检查分支规则（是否触发构建）
    if !matchBranchRule(service.BuildConfig.BranchRules, payload.Branch) {
        return
    }
    
    // 自动创建构建任务
    buildRun := &BuildRun{
        ServiceID:   service.ID,
        TriggerType: "webhook",
        GitInfo: BuildGitInfo{
            Branch:       payload.Branch,
            Commit:       payload.After,
            Author:       payload.UserName,
            Message:      payload.Commits[0].Message,
            ChangedFiles: extractFiles(payload.Commits),
        },
    }
    
    // 触发构建
    pipeline.TriggerBuild(ctx, buildRun)
}
```

## 9. 稳定性设计

### 9.1 构建节点故障不影响其他构建

- 每个构建是独立 Pod
- 节点故障 -> K8s 标记 Node NotReady -> 新构建调度到其他节点
- 正在运行的构建失败 -> 平台记录失败 -> 通知用户 -> 用户可一键重试

### 9.2 防止资源耗尽

```yaml
# namespace 级别 ResourceQuota
apiVersion: v1
kind: ResourceQuota
metadata:
  name: build-quota
  namespace: axiops-builds
spec:
  hard:
    requests.cpu: "32"      # 最多同时使用 32 核
    requests.memory: "64Gi" # 最多 64G 内存
    pods: "50"              # 最多 50 个并发构建 Pod
```

### 9.3 构建超时

- 默认 30 分钟超时
- 超时自动终止 Pod
- 记录失败原因
- 通知用户

### 9.4 构建队列

达到并发上限时排队：

```go
type BuildQueue struct {
    maxConcurrent int           // 最大并发（基于 ResourceQuota）
    running       atomic.Int32  // 当前运行中
    queue         chan *BuildRun // 等待队列
}

func (q *BuildQueue) Submit(run *BuildRun) {
    if q.running.Load() < int32(q.maxConcurrent) {
        q.running.Add(1)
        go q.execute(run)
    } else {
        run.Status = BuildStatusQueued
        q.queue <- run
    }
}
```

## 10. Host Agent（ECS/隔离网络）

对于不在 K8s 中运行的场景，使用轻量 Agent：

```text
适用场景：
- 隔离网络中的主机
- Windows/macOS/ARM 构建
- 需要 GPU 或特殊硬件
- ECS 主机部署（文件发布）

设计参考：reference/devops/agent
- WebSocket 长连接 + 心跳 + 断线重连
- 任务类型注册（build/deploy/health）
- 日志缓冲批量上报
- 工作空间隔离
- 脚本完整性校验
```

## 11. Jenkins 兼容（可选）

面向已有 Jenkins 的公司，提供兼容接入：

```text
接入方式：
- 在 AxiOps 配置 Jenkins 地址和凭据
- AxiOps 通过 Jenkins API 触发已有 Job
- 拉取构建状态和日志展示在 AxiOps 中
- 制品信息同步回平台

限制：
- 日志非实时（轮询，2-5s 延迟）
- Jenkins 故障时构建不可用
- 需要额外维护 Jenkins
```

新用户推荐直接用 K8s Job，不装 Jenkins。

## 12. 数据模型

```sql
-- 构建配置（每个服务一份）
CREATE TABLE build_config (
    id BIGINT PRIMARY KEY,
    service_id BIGINT NOT NULL,
    template_type VARCHAR(32),      -- maven/gradle/npm/pnpm/go/docker
    build_image VARCHAR(256),       -- 构建镜像
    build_command TEXT,             -- 构建命令
    dockerfile_path VARCHAR(256),   -- Dockerfile 路径
    image_repo VARCHAR(256),        -- 镜像仓库
    cache_enabled BOOL DEFAULT TRUE,
    timeout_seconds INT DEFAULT 1800,
    branch_rules JSON,              -- webhook 分支触发规则
    env_vars JSON,                  -- 构建环境变量
    created_at TIMESTAMP,
    updated_at TIMESTAMP
);

-- 构建运行记录
CREATE TABLE build_run (
    id BIGINT PRIMARY KEY,
    service_id BIGINT NOT NULL,
    project_id BIGINT NOT NULL,
    pipeline_run_id BIGINT,         -- 关联的流水线
    config_snapshot JSON,           -- 构建配置快照
    trigger_type VARCHAR(16),       -- manual/webhook/approval/schedule
    trigger_user_id BIGINT,
    status VARCHAR(16),             -- queued/running/success/failed/cancelled/timeout
    provider VARCHAR(16),           -- kube_job/host_agent/jenkins
    
    -- Git 信息
    git_branch VARCHAR(128),
    git_commit VARCHAR(64),
    git_author VARCHAR(64),
    git_message TEXT,
    git_changed_files JSON,
    
    -- 构建产物
    image_url VARCHAR(512),
    image_digest VARCHAR(128),
    artifact_url VARCHAR(512),
    artifact_size BIGINT,
    
    -- 执行信息
    job_name VARCHAR(128),          -- K8s Job 名
    node_name VARCHAR(128),         -- 执行节点
    started_at TIMESTAMP,
    finished_at TIMESTAMP,
    duration_seconds INT,
    error_message TEXT,
    
    created_at TIMESTAMP,
    INDEX idx_service_status (service_id, status),
    INDEX idx_project (project_id),
    INDEX idx_created (created_at)
);

-- 构建日志（大段日志存对象存储，这里只存元信息）
CREATE TABLE build_log (
    id BIGINT PRIMARY KEY,
    build_run_id BIGINT NOT NULL,
    stage VARCHAR(32),              -- git_clone/build/image/push
    log_url VARCHAR(512),           -- 对象存储地址
    line_count INT,
    created_at TIMESTAMP
);
```

## 13. API 设计

```text
# 构建
POST   /api/v1/builds                    触发构建
GET    /api/v1/builds                    构建列表（按项目/服务/状态过滤）
GET    /api/v1/builds/{id}               构建详情
DELETE /api/v1/builds/{id}               取消构建
POST   /api/v1/builds/{id}/retry         重新构建
GET    /api/v1/builds/{id}/logs           获取构建日志
WS     /api/v1/builds/{id}/logs/stream   实时日志流

# 构建配置
GET    /api/v1/services/{id}/build-config  获取构建配置
PUT    /api/v1/services/{id}/build-config  更新构建配置
POST   /api/v1/services/{id}/detect-lang   检测语言和推荐模板

# Webhook
POST   /api/v1/webhooks/git               Git Webhook 入口

# 构建模板
GET    /api/v1/build-templates             构建模板列表
GET    /api/v1/build-templates/{type}      模板详情
```

## 14. 实施计划

```text
第 1 步（3-4 天）：Build Controller + K8s Job 基础
  - Job 创建、状态 Watch、日志获取
  - 一个 Go 项目能跑通：拉代码 -> 编译 -> 打镜像 -> 推镜像

第 2 步（3-4 天）：构建模板和自动检测
  - 语言检测
  - Maven/Node/Go 模板
  - PVC 缓存
  - 构建信息注入（commit/branch/changed files）

第 3 步（3-4 天）：API + 前端
  - 触发构建 API
  - 构建详情和列表
  - 日志 WebSocket 实时流
  - Webhook 接入

第 4 步（2-3 天）：串联流水线
  - 构建完成 -> 触发部署
  - 审批节点
  - 失败/成功通知

总计约 2-3 周完成构建系统核心。
```

## 15. 持久化执行队列与状态边界

API 不直接在 HTTP 请求中执行构建。触发构建时，`build_run` 和一条
`build_command(command_type=submit)` 在同一个事务中写入；worker 通过
`build_command` 领取任务，再调用具体 provider。这样 API 进程重启不会丢失构建请求。

取消构建同样写入幂等的 `cancel` 命令。运行记录先从 `queued/running` 进入
`cancelling`，只有 provider 确认终止后才允许进入 `cancelled`。终态不能再次变更。

允许的核心状态流转：

```text
queued -> dispatching -> running -> success/failed/timeout
queued/running -> cancelling -> cancelled
dispatching -> failed
```

`build_command.idempotency_key` 保证 submit/cancel 重试不会创建重复外部任务；
worker 领取时使用 MySQL 8 `FOR UPDATE SKIP LOCKED`，通过 `locked_by/locked_at`
实现租约。超过最大尝试次数转为 `dead`，必须告警并人工处理，不能静默丢弃。

worker 进程通过 provider registry 分发命令。provider 调用失败时按租约重试；达到最大
次数后，命令转为 `dead` 与 `build_run` 转为 `failed` 在同一数据库事务中完成。
worker 异常退出后由 stale lease 回收任务，避免命令永久停留在 `processing`。

工作流侧已新增 `axiops-workflow-worker`，注册 build adapter、YAML apply 和 YAML rollback executor。
启用 `Build.Enabled` 后，同一进程的独立子循环消费 `build_command`，在专用构建集群提交/取消 Kubernetes Job，
观察 Job condition 和 Pod 定位并同步成功、失败、超时。构建触发时冻结仓库 URL 与非敏感配置，防止排队期间服务编辑
改变执行输入；仓库凭据、实时日志、镜像 digest 和制品元数据仍待接入。

YAML provider 使用按 `cluster_id` 创建的动态 Kubernetes client，kubeconfig 必须为 `v1:` AES-GCM 密文，
仅在 worker 内存中短时解密。客户端设置 QPS、Burst、请求超时和缓存 TTL；集群非 connected、凭据明文、
密文认证失败或资源 discovery 失败时任务明确失败，凭据内容不写入工作流、快照或日志。

## 16. 三类型项目目标架构

本文件前面的构建 Job 设计仍可作为构建子系统参考；原第 14 节的单一构建实施周期不再是
CI/CD 总体计划。总体产品、数据和阶段验收以 [PRODUCT.md](PRODUCT.md)、
[DATA_MODEL.md](DATA_MODEL.md) 和 [IMPLEMENTATION_PLAN.md](IMPLEMENTATION_PLAN.md) 为准。

```text
Project(k8s_yaml | helm | host)
  -> Service definition/revision
  -> Project environment + variables + resource binding
  -> Workflow DAG / workflow run
  -> build_command / task worker
  -> yaml provider | helm provider | host agent provider
  -> deployment snapshot / logs / artifacts / audit
```

控制面只保存不可变定义、授权、快照、状态和审计。执行面必须按 provider 分离：

- `k8s_yaml`：解析并校验 Git YAML，渲染变量后执行受控 apply，保存 rendered manifest。
- `helm`：解析 Chart 和 values，预览 `helm template`，以 release revision 作为回滚基线。
- `host`：由出站 Agent 领取短期签名任务，下载校验过的制品，执行白名单脚本并回传探活结果。

三个 provider 共享工作流、命令租约、日志、制品、权限和审计契约，但不得共享不兼容的
变量渲染、资源绑定或回滚实现。生产部署必须在执行前重新检查环境范围授权和审批结果。
