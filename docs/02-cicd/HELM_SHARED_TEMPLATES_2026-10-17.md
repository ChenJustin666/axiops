# Helm 共享模板、通用应用 Chart 与验收

本版结构：一份共享应用 Chart 可以被多个服务使用；每个服务、环境固定模板版本，使用独立 values 和 Release。共享的是部署结构，发布和回滚仍按服务、环境进行。保存模板、Chart 或 values 都不会自动发布。

## 日常使用

1. 在 **模板库 → Helm Chart** 选择“通用应用 · 工作负载与配套资源”，或创建自己的 application Chart。保存自动生成不可变版本。可在“版本与使用情况”查看历史文件及有权查看的使用方；采用旧版本后保存，会产生新版本。
2. 在 Helm 项目创建服务只提供“模板”和“代码仓库导入”两种方式；选“模板”后选择该模板。多个服务可选择相同模板。平台从服务器加载固定版本，客户端不能给修改过的 Chart 冒用模板版本。服务创建后仍可独立编辑 Chart，不自动修改其他服务。
3. 在 **服务与配置 → 服务 → 目标环境 → 环境 values** 修改参数。预览实际 Kubernetes YAML，核对环境与 Namespace 后保存。生产需要额外确认；保存不会部署。
4. 测试环境执行构建并部署；生产优先选择测试过的固定摘要镜像发布，使用生产自己的 values。
5. 模板库发布新版本后，服务旁显示可用更新。点“选择模板版本 / 有新模板版本”→ 比较新增、修改、删除的 Chart 文件 → 用当前环境 values 校验预览 → 保存到此环境。其他服务和环境不跟随更新。
6. 已编辑的自定义 Chart 不被模板自动覆盖；换回共享模板需要明确确认替换。历史模板无法验证时也按自定义配置处理。

“Chart 版本 #N”是服务的不可变 Revision；“模板 vN”是模板库版本；运行记录中的 Helm Release revision 是一次实际发布的版本。这三个编号用途不同。

## 通用 Chart 支持范围

| 能力 | values 入口 | 行为 |
| --- | --- | --- |
| 无状态服务 | workload.kind: Deployment | 默认生成 Deployment 和 Service |
| 有状态服务 | workload.kind: StatefulSet | 生成 headless Service；持久化默认每副本独立 claim |
| 节点常驻组件 | workload.kind: DaemonSet | 生成 DaemonSet；可关闭网络 Service |
| 一次性任务 | workload.kind: Job | 每个 Release revision 使用新 Job 名，等待完成；默认失败不重试，不自动回滚重放旧任务 |
| 定时任务 | workload.kind: CronJob | 五段 cron，默认 Forbid；发布成功表示计划已创建，不代表未来任务执行成功 |
| ALB / NGINX 等路由 | ingress | 多域名、多路径、三种 pathType、class、标签、注解；TLS 默认关闭 |
| 配置文件 | configMaps | 多份 ConfigMap、多键，可选择目录挂载 |
| 安全凭据 | secrets + 安全凭据引用 | Opaque、TLS、basic-auth、ssh-auth；多字段注入，不在 values 保存正文 |
| 已有 Secret | env / envFrom / secretMounts | 原生环境变量引用或只读文件挂载 |
| 持久化 | persistence | 新 PVC 或 existingClaim；新独立 PVC 默认保留数据 |
| 伸缩与中断预算 | autoscaling / pdb | HPA 仅用于 Deployment、StatefulSet；自动伸缩时不覆盖 replicas |
| 健康检查 | probes | 默认关闭；原生 HTTP、TCP、exec、gRPC，各探针独立 |
| 监控注解 | monitoring | 可选 Prometheus 抓取注解；不代表已安装监控系统 |
| 调度及安全 | nodeSelector / affinity / tolerations 等 | 调度、拓扑、Pod/容器安全、初始化容器、sidecar |
| 访问控制 | serviceAccount / rbac / networkPolicy | Namespace 内 ServiceAccount、Role、RoleBinding、NetworkPolicy |

CRD、Namespace、ClusterRole 等集群级资源不在应用发布范围内。Ingress Controller、StorageClass/CSI、metrics-server、支持 NetworkPolicy 的 CNI 等需由集群预先提供；模板无法替代这些组件。第三方 Chart 使用自己的 values 约定，可导入已打包依赖；一个组合 Chart 中的多个应用会作为同一个 Release 发布、回滚。

包含普通 Job 资源的 Release 关闭 Helm 自动原子回滚和失败清理，以免旧任务被重新创建执行；失败 Job 按自身 TTL 保留供排查。手工服务回滚也会在预览和执行阶段拒绝此类历史，需要明确重新发布一个任务。Helm hook 遵循 Chart 的 hook 生命周期，不能把数据库副作用当成可自动回滚的数据。CronJob 的发布只验证计划创建，后续每次执行需查看 Job/Pod 状态。

## 常用配置示例

下面是通用 Chart 的环境 overrides，未填写的字段使用所选模板版本的默认值。Service 对外端口与容器实际监听端口可不同；Ingress 始终跟随此 Service 的名称和 port。

```yaml
workload:
  kind: Deployment
replicaCount: 1
service:
  port: 9099
  targetPort: 28080
resources:
  requests: {cpu: 250m, memory: 512Mi}
  limits: {cpu: "1", memory: 2Gi}
env:
  - name: SPRING_PROFILES_ACTIVE
    value: test
  - name: JAVA_OPTS
    value: "-XX:MaxRAMPercentage=72.0 -XX:MaxMetaspaceSize=256m"
ingress:
  enabled: true
  className: alb
  labels: {ingress-controller: alb}
  annotations:
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP":80},{"HTTPS":443}]'
  hosts:
    - host: app.example.com
      paths: [{path: /, pathType: Prefix}]
  tls: []
probes:
  readiness: {tcpSocket: {port: 28080}, periodSeconds: 10}
  startup: {tcpSocket: {port: 28080}, failureThreshold: 30, periodSeconds: 10}
```

示例域名需替换；annotations 全部可省略。ALB TLS 可填写 hosts 而不使用 Kubernetes secretName，由 ALB 配置证书；其他控制器通常需指定有效 TLS Secret。探针端口按应用实际监听端口填写，修改端口时同时核对探针。模板不覆盖镜像启动命令，只有明确填写 command/args 才覆盖。

有状态服务可设置：

```yaml
workload: {kind: StatefulSet}
replicaCount: 3
persistence:
  enabled: true
  size: 20Gi
  storageClass: your-storage-class
  accessModes: [ReadWriteOnce]
  mountPath: /data
```

StatefulSet 的 PVC 数据生命周期仍由 Kubernetes 管理。独立 PVC 的 retain 默认 true；明确关闭后卸载可能删除 claim。不应把回滚当成数据库数据恢复。多副本 Deployment 新建共享 PVC 时须选择 ReadWriteMany；不能靠模板保证 RWO 卷能跨节点挂载。

## 密码、SK、Token、证书的多字段引用

1. 在平台凭据管理创建 **secret 类型安全凭据**，可含 password、SK、username、token 等多个字符串字段；正文只在创建/修改凭据时输入。
2. 在环境 values 声明资源结构，例如 `secrets: {auth: {type: Opaque}}`。
3. 展开 **安全凭据 · 多字段引用**，添加 `secrets.auth.stringData.password`，选择凭据及 password 字段；再添加 `secrets.auth.stringData.SK` 选择 SK 字段。
4. 在 values 使用原生引用，例如当前 Release 为 gateway：

```yaml
secrets:
  auth: {type: Opaque}
envFrom:
  - secretRef:
      name: gateway-auth
```

也可用 `env[].valueFrom.secretKeyRef` 只引用其中一个字段。生成 Secret 的名称为 `<Release>-<配置名称>`。引用不会自动把所有字段变成容器 env；这是为了避免无意向应用注入不需要的密钥。

含点号的 Secret 键（例如 tls.crt）通过 fields 映射：

```yaml
secrets:
  tls:
    type: kubernetes.io/tls
    fields:
      cert: {key: tls.crt}
      private: {key: tls.key}
```

分别绑定 `secrets.tls.fields.cert.value` 和 `secrets.tls.fields.private.value` 到凭据字段。不要在 values 手写 value 正文。字段引用支持修改、添加、移除环境覆盖；移除覆盖后恢复继承。同步其他环境只同步选中的普通 values，不复制密钥身份或字段映射；各环境分别绑定自己的凭据。

保存时校验凭据项目归属、权限、字段存在及 Chart 类型；预览重新用占位值渲染，避免 b64enc 等转换泄露。Secret 正文隐藏。Helm 自身在集群中的 Release 存储仍须用 RBAC 保护。

## 校验与生效边界

服务端执行 values/schema 校验、渲染和 Kubernetes 字段/类型校验；通用 Chart 另检查资源数量及上下限、端口、域名、探针、cron、HPA 请求、PVC/卷引用与挂载冲突等。不兼容的新模板版本会连同新服务 Revision 和环境绑定一起回滚；旧预览不能覆盖新配置。

这些静态校验不证明镜像内应用能启动，也不能证明集群有可用存储、证书或网络。实际发布继续等待 Helm 就绪结果，通过运行日志、环境事件和容器日志排查。

配置预览渲染为有效 Kubernetes YAML；Job 的最终名称还含执行时的 Helm Release revision，不能在保存配置时承诺该编号与实际发布完全相同。

## 数据库与验收状态

新增 `052_delivery_template_versions.sql`，必须通过统一迁移工具按序执行，再配套更新 CI/CD API、Workflow Worker 和前端。启动只回填迁移前模板的确切当前版本；更早已被覆盖的内容无法恢复。默认模板不会覆盖用户已编辑的条目，归档不会删除已保存历史或环境快照。

本轮只在独立临时数据库执行集成测试，没有迁移业务库，也没有触发真实构建、发布、回滚或通知。相关 Go、数据库和前端测试记录位于 `local/dev/helm-shared-*.log`；全仓测试和浏览器/真实集群验收的具体结果见 [Helm 交付状态](HELM_DELIVERY_2026-10-17.md)。
