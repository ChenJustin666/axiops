# 服务资源配置与校验

配置入口：项目 → 服务与配置 → 选择服务和目标环境。保存只修改配置，运行结果要通过工作流部署及就绪检查确认。

| 配置 | 填法 | 运行条件 |
| --- | --- | --- |
| Secret | 当前命名空间的 Secret 名称和键；键可以是 `password`、`SK`、`token`、`tls.key`，可添加多项。Env 名例如 `API_SK`，文件挂载例如 `/app/secrets/token`。 | 对应 Secret 和键必须存在。平台凭据通过配套配置加密保存并绑定，不写入普通 YAML。 |
| ConfigMap | 现有名称留空则创建；文件名如 `application.yaml`，挂载路径如 `/app/config/application.yaml`。内容原样保存，每项最多 256KiB。 | 引用已有 ConfigMap 时必须有指定文件键。应用需主动读取挂载文件；`subPath` 不热更新，需重新发布滚动 Pod。 |
| PVC | 引用已有 Claim，或填写新容量如 `10Gi`、StorageClass 和访问模式。至少 1Mi；`2` 是 2 字节，不能代表 2Gi。 | RWO 是单节点读写；多副本跨节点共享通常需 RWX 及兼容的存储类。RWOP 需 CSI 支持。StorageClass 留空使用集群默认类。 |
| Ingress | 域名如 `api.example.com`（无协议/端口），支持 `*.example.com`；路径如 `/api`。Class 填集群已有名称，TLS 引用当前命名空间证书 Secret。 | 控制器正常运行、域名解析指向入口。TLS Secret 必须有 `tls.crt` 和 `tls.key`，证书需匹配域名。 |

PVC Pending 不直接视为失败：WaitForFirstConsumer 存储类需要 Pod 调度后才绑定。存储配额、驱动状态和挂载结果仍以 Kubernetes 事件为准。

## bac 示例核对

- CPU `250m/1`、内存 `512Mi/2Gi`、TCP 探针端口 `8080`、HPA `1–3` 和 CPU `80%` 结构合法。资源是否够用需实际压测；HPA 需要 metrics API。Prometheus 注解不会自动创建指标接口。
- `nacos.example.invalid:8848`、`replace-with-namespace-id` 必须换成目标环境的真实值。`test` Profile 是否适用 QA 由应用配置决定。
- `axiops-config-validation.invalid/pending-image:validation` 仅供预览，真实部署必须替换为实际镜像。
- `rsc01-qa` 中 Secret `abc` 必须有 `password` 键，ConfigMap `app` 必须有 `application.yaml` 键，PVC `agd` 必须存在。引用已有资源时预览显示引用，不生成其正文。
- `sdfsdf` 可以是合法名称，但必须确实是目标集群的 IngressClass。
- 模板库保留基础 Deployment/Service；创建服务时追加的 HPA 属于该服务。Ingress/ConfigMap/PVC/Secret 配套配置按目标环境合并，不修改公共模板。

## 校验范围与生效

前端完成可选项时检查名称、键、路径、容量、重复项和冲突，API 再次校验。真实 YAML 部署前核对目标集群的 Secret/ConfigMap 引用及键、已有 PVC、显式 StorageClass、IngressClass 和 TLS 键；识别本次创建的资源。缺少依赖时在写入工作负载前失败；错误不包含密码内容。

这不是所有 Kubernetes 字段与应用配置的完整语义校验。证书有效期、DNS、Nacos 连通性、应用配置文件语法、存储驱动能力及任意手写 YAML 的最终准入结果，仍需结合目标集群与应用验证。预览通过不等于运行成功。

前端更新后刷新；后端格式校验需要更新 CI/CD API，部署预检需要更新 Workflow Worker。本轮未重启已有进程、未修改业务资源或触发实际部署。
