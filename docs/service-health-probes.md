# 服务健康检查与端口修改

## 如何选择

探针是 Kubernetes 的可选能力。平台发布时还会观察 Deployment/Pod 的就绪状态；这与给容器添加探针是两件事。

| 方式 | 检查内容 | 适用情况 |
| --- | --- | --- |
| 不生成探针 | 不添加平台生成的应用探针 | 尚无可靠健康检查、批处理，或已在原生 YAML 中配置检查 |
| TCP | kubelet 能否连接 Pod 的指定端口，无需 HTTP 路径 | 没有 HTTP 健康接口，先确认应用开始监听 |
| HTTP | 请求 Pod 的实际健康接口；HTTP 200–399 为成功 | 应用提供可靠、无需登录的健康接口 |

TCP 连通不代表数据库、Nacos 或业务正常。探针访问 Pod，不经过 Ingress 域名。探针端口填 `0` 表示跟随应用端口，不表示让 Kubernetes 自动发现端口。

三种目的可以独立开启：

| 检查 | 失败的结果 | 使用建议 |
| --- | --- | --- |
| 就绪 readiness | Pod 不再作为就绪后端接收 Service 流量，检查本身不触发重启 | 常驻服务优先配置 |
| 启动 startup | 在启动阶段保护慢启动；连续失败超过阈值后重启容器 | Java 等慢启动服务按实际启动时间配置 |
| 存活 liveness | 连续失败超过阈值后重启容器 | 确认能识别死锁等需要重启的问题后再开 |

配置 startup 后，在它成功之前，readiness/liveness 不会执行。默认周期 10 秒、单次超时 3 秒、普通连续失败 3 次、启动连续失败 30 次；启动允许时间大约 5 分钟，实际还受探测执行与终止宽限时间影响。启动慢可以增加启动失败次数，不能靠关闭探针修复错误端口。

新启用检查时，界面默认选就绪和启动检查，存活检查默认不选。已有服务没有独立开关参数时，保留原先三类探针的行为；用户可以在对应环境中主动调整。

没有应用探针时，Kubernetes 仍会检查调度、镜像、容器运行和资源 Ready 等状态，但 Ready 不能证明业务接口可用。长时间对外服务建议配置可靠的 readiness。

## 在界面使用

1. 项目 → 服务与配置 → 选择服务 → 选择准确的目标环境 → 解锁编辑。
2. 右侧“健康检查”选择不生成、TCP 或 HTTP；按需勾选就绪、启动、存活检查。
3. 默认跟随应用端口。独立管理端口可取消跟随，填写实际端口。TCP 不显示或要求 HTTP 路径；HTTP 填实际存在的路径，不猜测 `/health` 一定可用。
4. “预览与保存”检查完整资源及前后差异，确认目标环境后保存。保存配置不会自动改变集群，仍需通过配置发布或部署工作流使配置生效。

模板创建时，在“模板健康检查”中进行同样的选择；没有健康参数的模板，可在可选资源中配置探针。不会重复添加两个探针配置入口。

原生 YAML 中已有的探针优先于快捷参数。选择“不生成探针”只停止参数生成，不删除手写探针；界面会提示。要关闭手写探针，在目标环境 YAML 中删除对应 `readinessProbe`、`startupProbe`、`livenessProbe`，预览后保存。不同容器、exec/gRPC、独立 HTTP 路径、HTTP 请求头、独立周期等高级设置直接使用原生 YAML。

Spring Boot 有 Actuator 时，可使用真实暴露的 `/actuator/health/readiness` 和 `/actuator/health/liveness`。这些接口需要应用确实启用；使用独立管理端口时，还要考虑主业务端口是否正常。不要把通用聚合 `/actuator/health` 中的 Nacos/数据库故障直接作为 liveness 重启依据。需要不同路径时，分别在原生 YAML 中配置。

## gateway 从 8080 改为 28080

本次读取的环境 #22、服务 #69：应用端口变量为 8080，TCP 探针端口为 0（跟随应用），配套路由后端固定为 8080，YAML 未手写探针。实际应用若监听 28080，旧探针访问 8080 必然失败，启动探针最终可能重启容器。这发生在部署运行阶段，与镜像是否构建成功无关。

在该环境将应用端口改为 `28080`，保留探针跟随应用端口，进入预览。对于原先绑定同一命名 Service 端口的配套路由，平台会更新到新端口，并提示 `8080 → 28080`。检查 Service、探针和 Ingress 后再保存并发布。修改 `containerPort` 或 Service 声明不会修改应用自己的监听端口；Spring `server.port` 等应用配置也必须与实际一致。

联动只在配置保存事务中进行：已有效的后端端口不动；按同名 Service 的同名端口和协议映射，或两个版本都是唯一无名端口时映射。无法确定、多端口歧义、删除 Service、用户主动选择非法端口，仍拒绝保存。原生 Ingress 由用户显式维护。Ingress 指向的是 **Service port**，例如 Service 80 → container 28080 时，Ingress 仍应指向 80。

预览回滚，保存与路由调整原子提交；只修改目标环境，不影响其他环境、共享模板或已经提交的部署快照。

## 验证与参考

本次验证包括三类开关、TCP 无路径、HTTP readiness、关闭后不生成探针、原生探针保留、端口联动、Service/容器端口区分、歧义拒绝、真实 MySQL 临时库中的预览回滚和环境隔离。真实服务只执行预览，没有修改集群或发起新的发布。

- [Kubernetes：探针用途与风险](https://kubernetes.io/docs/concepts/configuration/liveness-readiness-startup-probes/)
- [Kubernetes：探针配置与时间参数](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/)
- [Spring Boot：Kubernetes Probes](https://docs.spring.io/spring-boot/reference/actuator/endpoints.html#actuator.endpoints.kubernetes-probes)
