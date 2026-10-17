# Sinozo Java / Nacos / SkyWalking 模板

模板库名称：**Sinozo · Java / Nacos / SkyWalking**，内置标识 `helm-sinozo-java-v1`，Chart 名 `axiops-sinozo-java`。这是 Java 应用标准模板，多个服务共用模板，每个服务/环境拥有独立配置与 Release；不是一个 Release 同时发布多个独立服务。

参考了 `H:\sinozo-shared-library\charts\generic-service`，保留 `service.port/replicas`、资源、`java.opts`、`ingress.items` 的习惯，但没有改动 H 盘原库。未复制自动时间戳、空标签或不存在的 TLS Secret。所有路由自动指向当前服务的 Service；无需手填后端服务名和端口。

## 如何创建和使用

1. 更新配套 CI/CD API/Worker 后，模板库 → Helm Chart 选择本模板。API 启动的模板初始化会新增此项，不覆盖已有模板。已有版本迁移要求仍按 Helm 主交付说明执行；本模板没有新增数据库迁移。
2. Helm 项目 → 服务与配置 → 模板，选择本模板，填写服务名，创建服务。项目初始化中也可使用同一模板。新建服务只有模板或代码仓库导入两种方式。
3. 选择测试环境，在该服务的“环境 values”使用 [values-test.yaml](examples/java-skywalking/values-test.yaml)。此文件已经按提供的 `28080`、`rsc01-test`、测试域名和 MSE 地址填写；MSE 地址补全了 `:8848`，请确认实例实际开放端口、网络访问和 namespace ID。服务名默认跟随服务编码，不固定 `card-scout-gateway`。
4. 点“预览与保存”，检查真实 Deployment/Service/Ingress；核对目标集群和 Namespace。镜像沿用 AxiOps 默认 `image.repository`＋`image.tag` 映射，发布时注入所选/新构建制品的固定镜像。Chart 中的示例镜像不能直接用于业务运行。
5. 工作流执行发布。保存环境配置不会立即发布；其他服务和环境不会自动变化。

若暂不更新 AxiOps，可将导出的 Chart 解压提交到自己的代码仓库，再用“代码仓库导入”选择 Chart 完整文件。Chart 的 JSON schema 和模板校验随包携带；JVM 预算、资源数量等平台附加校验需要配套新版后端。

## 日常只改哪些差异

| 参数 | 使用方式 |
| --- | --- |
| `service.replicas` | 副本数，默认 1；0 表示停服 |
| `service.port` | 应用监听端口；默认联动 SERVER_PORT、Service、Ingress、探针和监控 |
| `resources.requests/limits` | 当前环境 CPU/内存，默认 100m/256Mi 请求、500m/512Mi 上限 |
| `application.name` | 空值跟随 Release；Spring 应用名、Nacos Data ID 和 SkyWalking 名默认跟随它 |
| `application.profile` | 明确填 test/prod 等，不按名称猜测；空值不注入 Profile |
| `application.logLevel` | 注入 LOG_LEVEL，需应用本身支持；不是强制修改所有 Spring 日志级别 |
| `nacos.serverAddr/namespace/group` | 对应环境的地址、Nacos namespace ID、分组；public namespace 可空 |
| `nacos.configImport.dataId` | 默认为应用名；可自定义，`optional` 控制配置缺失是否允许 |
| `java.opts` | 只放 JVM 参数；Spring/Nacos 参数在独立字段设置，避免系统属性覆盖环境变量 |
| `ingress.items` | 当前环境域名、路径、IngressClass、注解和 TLS；数组同步为整项替换 |

Service 和资源默认开启；无对外 Service 的后台消费者可设置 `service.enabled: false`，此时禁止同时开启 Ingress。资源一般不需要关闭；`resources.enabled: false` 是高级自定义能力，关闭后不生成 requests/limits。

应用如果在启动参数、配置文件或 Nacos 里另行固定端口，必须与 `service.port` 一致。默认通过 Spring 的 SERVER_PORT 设置监听端口；自定义框架可清空 `application.portEnv`，在 `env` 配置实际支持的端口变量。镜像 `EXPOSE` 不能替代应用监听配置。

## JVM 与 JDK

512Mi 上限下，原来的 72% 堆＋256Mi 元空间合计已超过容器上限，尚未计入线程、直接内存和 Agent。模板默认使用 **50% 堆＋128m 元空间**，给其他开销留出余量；这不代表任何 Java 应用都能在 512Mi 下稳定运行，需按实际负载调整。

默认变量为 `JAVA_TOOL_OPTIONS`，JVM 自动读取，不依赖镜像入口脚本展开 `JAVA_OPTS`。现有镜像若要求 `JAVA_OPTS`，将 `java.optionsEnv` 改成它，并确认入口确实执行 `java $JAVA_OPTS ...`。不要再把相同选项写入其他环境变量或命令参数。

JDK 版本由业务镜像和构建配置决定，Chart 不会因为填了一个版本号就更换运行镜像。JDK 8 使用支持容器和百分比参数的 8u191+（建议维护中的较新版本）；JDK 11/17/21 可按实际 JVM 调整 `java.opts`。不添加 Java 17 已移除的 `--illegal-access`。模板只为 Java 做默认参数；Node/Python/Go 应另建各自标准模板。

平台拦截已知的堆＋元空间达到内存上限、requests 大于 limits、无效资源单位等情况；应用真实直接内存、线程、Agent/JDK 兼容和配置中心连通性仍需运行验收。

## 开启链路追踪和 HeapDump

将 [values-tracing.yaml](examples/java-skywalking/values-tracing.yaml) 的字段合并到当前环境 values。示例上限提高到 1Gi，增加 SkyWalking Agent `9.5.0-alpine`，OAP 使用你提供的集群域名。该地址只在目标集群可解析且可访问对应服务时有效。Agent 默认使用已从官方 Registry 核对的多架构 digest（linux/amd64、linux/arm64），避免辅助镜像浮动导致精确回滚被拒绝；自定义 Agent/附加容器时也应固定摘要。

开启后生成：Agent initContainer → 共享卷复制 Agent → 主容器只读挂载 `/skywalking` → JVM 自动添加 `-javaagent`。服务名和 OAP 使用 Agent 官方支持的环境变量 `SW_AGENT_NAME`、`SW_AGENT_COLLECTOR_BACKEND_SERVICES`，日志写 stdout，排除插件可配置。已对照 [SkyWalking Java 9.5.0 官方 agent.config](https://github.com/apache/skywalking-java/blob/v9.5.0/apm-sniffer/config/agent.config) 核对变量名。

`java.heapDump.enabled` 独立于 SkyWalking。默认 `/dump` 是有大小上限的磁盘 emptyDir，Pod 删除后不保留；需要持久化时填 `existingClaim` 并确认容量、权限、多副本和访问模式。只引用现有 PVC，不自动创建存储。开启 `ExitOnOutOfMemoryError` 后 JVM 会退出并由 Kubernetes 重启。

## Nacos 认证、密码、SK 和 webhook

默认认证关闭，不生成任何 Nacos 用户名/密码 env。开启时仅配置目标 Namespace 的 Secret 引用：

```yaml
nacos:
  enabled: true
  auth:
    enabled: true
    secretName: nacos-auth-secret
    usernameKey: NACOS_USERNAME
    passwordKey: NACOS_PASSWORD
```

模板为 Spring Nacos 公共、发现和配置客户端生成认证引用，Secret 不存在或 key 不匹配时 Pod 会明确失败，不设置 `optional: true` 掩盖问题。不要把密码塞进 `java.opts`。

其他密钥使用原生 env 引用，同一个 Secret 可以含多个键：

```yaml
env:
  - name: FEISHU_WEBHOOK
    valueFrom:
      secretKeyRef:
        name: alert-auth
        key: webhook
  - name: APP_SK
    valueFrom:
      secretKeyRef:
        name: application-auth
        key: sk
```

本 Chart 只引用已有 Secret/ConfigMap/PVC，不把这些对象的敏感正文写进环境 values。需要 Chart 自身创建配套对象时，可从通用工作负载模板复用其配置/安全凭据绑定文件，或编辑此模板增加相应资源。不要将模板库当密码库。

旧项目若使用自定义的 `-DNACOS_SERVERS`、`-DNACOS_NAMESPACE` 而非 Spring Cloud 标准属性，需要按应用实际实现调整模板 env/JVM 参数；不能保证任意应用都识别相同变量。Spring Boot 2.4 前不使用 `spring.config.import` 时关闭 `nacos.configImport.enabled`。

## ALB / Nginx / TLS

- [测试 HTTP 示例](examples/java-skywalking/values-test.yaml)：TLS 关闭，完全不生成 `tls`；普通路径 `/` 使用 Prefix。
- [ALB TLS 示例](examples/java-skywalking/values-alb-tls.yaml)：开启 HTTPS 监听和重定向，TLS hosts 自动从规则域名取值，不凭空生成 Secret。ALB 仍需正确绑定证书、AlbConfig、网络和 IngressClass。
- [Nginx TLS 示例](examples/java-skywalking/values-nginx-tls.yaml)：class 改 nginx，使用 Nginx 注解，明确提供已有 TLS Secret。自定义 IngressClass 名称同样支持，证书要求由对应控制器决定。

ALB 的 CORS、排空、order、group、监听端口等注解可原样填写在每项 `annotations`，字符串中的 `/${2}` 原样保留。注解的组合与控制器版本需遵循该控制器规范。不要把 ALB 注解直接用于 Nginx。

正则/重写应采用 `ImplementationSpecific`，让具体控制器处理；普通路径使用 Prefix/Exact。例如：

```yaml
ingress:
  enabled: true
  items:
    - name: gateway
      className: alb
      labels: {ingress-controller: alb}
      annotations:
        alb.ingress.kubernetes.io/use-regex: "true"
        alb.ingress.kubernetes.io/rewrite-target: /${2}
      hosts:
        - host: test-api.zovrikiq583.xyz
          paths:
            - path: /prod(/|$)(.*)
              pathType: ImplementationSpecific
      tls: {enabled: false}
```

空 host 支持无 TLS 的 catch-all；开启 TLS 必须填写合法域名。模板不写空 project/environment 标签。单个路由始终指向本服务，前端和网关是两个独立服务时，各自配置路由即可共享同一域名。

## 探针与监控

探针总开关默认关闭。开启后默认 startup＋readiness，liveness 独立默认关闭；TCP 不要求路径，HTTP 才读取 `probes.path`。`port: 0` 跟随服务端口，可为特殊管理端口单独设置。

`monitoring.enabled: true` 自动生成标准 Prometheus scrape/path/port 注解，port 默认跟随服务；仍需采集器规则和应用实际暴露端点。你现有的 `prometheus.io/jvm/jvmpath` 可继续在 `extraAnnotations` 配置，它们是现有采集约定，不会凭注解自动安装监控系统。

## 生产环境

[values-prod.yaml](examples/java-skywalking/values-prod.yaml) 是明确待填写的生产覆盖文件：Profile=prod，资源和副本单独设置，但不默认复用测试 Nacos、域名和 Secret。生产 Nacos 地址未填时会校验失败。核对这些值后预览保存，再使用测试验证过的固定镜像发布。

同步环境仍需选择字段、查看差异并保存；同步不会自动发布。模板库修改会产生新版本，不自动修改已建服务/环境。

## 验证与导出

```powershell
go test ./internal/cicd/templatelib ./internal/cicd/helmconfig
go run ./tools/export-helm-template -key helm-sinozo-java-v1 -output local/dev/helm-templates
```

输出标准 `.tgz` Chart 和独立模板 JSON。已有 Helm CLI 可使用：

```sh
helm template card-scout-gateway ./axiops-sinozo-java-1.0.0.tgz \
  --namespace rsc01-qa -f values-test.yaml -f values-tracing.yaml
```

源码测试覆盖真实 Helm engine 渲染、Kubernetes 严格类型解析、端口联动、开关、认证引用、环境隔离、TLS 与示例文件；不等同于真实 ACK/Nginx、镜像/JDK/Agent、Nacos/OAP 网络连接和应用就绪验收。没有向业务集群发布或修改 H 盘共享库。
