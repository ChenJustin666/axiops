# Java / Spring Boot / Nacos 模板使用

## 选择模板

1. 打开左侧“模板库” → K8s YAML，找到 **Java 17 / Spring Boot / Nacos 服务**。
2. 先创建环境并绑定集群/Namespace。新服务：在“服务与配置”点击顶部“模板”，选择目标环境、该模板，填写服务名称及当前环境参数。已有服务：右键服务或点击“⋯ → 更换模板”，选择目标环境后“保存当前环境”，不需要删除重建；如果已删除，先到“已删除服务”恢复。
3. 模板包含 `deployment.yaml` 和 `service.yaml`，在同一编辑区以 `---` 分隔。点击文件名会定位对应区域。
4. 核对服务代码和应用端口。服务代码自动用于 Deployment、Service、Spring application name 和默认 Nacos Data ID。如果实际应用名称或 Data ID 不同，直接修改对应 env 的 value。

模板导入会保存一份服务定义。以后修改模板库不会自动覆盖已有服务或环境；已有服务按环境修改后可使用“对比并同步其他环境”，核对差异后再保存。

模板库更新用于之后的创建；服务配置默认显示具体环境，点击“保存环境配置”只保存当前服务的当前环境。A、B 即使来自同一模板也互不影响，测试/生产同样互不跟随修改。跨环境修改需明确选择同步字段或参考来源、核对目标差异后保存；保存不等于部署。

## 配置测试与生产

新建时直接填写模板表单中的副本、CPU/内存、端口、Nacos 地址/Namespace 等参数。Env / JVM、模板健康检查、其他参数分别折叠；模板健康检查选择关闭、TCP 或 HTTP，默认关闭，TCP 不显示 HTTP 路径。此模板已有健康检查参数，不再显示可选探针按钮。展开可选资源后点击“+”添加路由、存储、Secret 引用、配置文件、HPA 或监控注解，填写后“完成添加”；可以添加多项，每次只展开当前编辑项，完成项可编辑或删除。完成或删除后自动生成“最终 YAML 预览”，显示真实资源和 Deployment 中的挂载或引用，变量已替换；已有 Secret 只展示引用，不显示密码正文。也可点击“预览完整配置”，修改参数后重新预览。新 PVC 的 StorageClass 可留空使用集群默认类；已有 PVC 只填写名称和挂载目录。模板源文在“高级：编辑模板 YAML”中单独编辑，内部变量元数据注释已隐藏。创建后在“服务与配置”选择实际环境，解锁编辑 YAML 或运行参数。新环境点击“配置此环境”，选择模板或参考已有环境的完整 YAML，再核对差异。

按你给出的 gateway 示例，测试环境可填写：

| 参数 | 测试环境示例 |
| --- | --- |
| `springProfile` | `test` |
| `nacosServerAddr` | `mse-7a42cca7-nacos-ans.mse.aliyuncs.com`，端口按实际 MSE 配置填写 |
| `nacosNamespace` | `rsc01-test`，前提是它确实是 Nacos 的命名空间 ID |
| `nacosGroup` | `DEFAULT_GROUP` |
| `port` | 应用实际监听端口 |
| `memoryLimit` | 默认 `2Gi`，按环境调整 |

模板的 Nacos 地址和命名空间是占位示例，使用前必须替换。生产环境分别填写真实的 Profile、Nacos 地址、命名空间 ID 和运行资源。Nacos 命名空间 ID 与 Kubernetes Namespace 是两回事，不会自动推断。

修改后点击“保存环境配置”。保存只更新配置，下次执行目标环境的部署工作流才会应用；已排队的运行沿用提交时的快照。同步时只选择需要的字段，检查目标环境的地址、Profile 和凭据，避免把测试配置带入生产。

默认 JVM 堆比例为 72%，元空间上限 256m，容器内存上限 2Gi。若改成 1Gi，可先将 `javaMaxRamPercentage` 调成 `60.0`、`javaMaxMetaspaceSize` 调成 `128m`，再根据实际应用的线程、直接内存和负载测试调整，避免 OOM。

## Nacos 账号密码

新建服务使用的 Nacos 模板默认不生成账号密码 Env，也不引用 Secret。无需鉴权时直接保存；需要鉴权时，在目标环境“配套配置 → 密码与凭据”选择已有 Secret，或新建多键 Secret，添加 `username`、`password` 两行，分别注入 `SPRING_CLOUD_NACOS_USERNAME`、`SPRING_CLOUD_NACOS_PASSWORD`。系统按选择生成 Secret 引用，不会把密码放进 JAVA_OPTS 或普通变量。

已经用旧版模板创建的服务不会被模板库更新自动覆盖。如果当前环境 YAML 仍有两条 `optional: true` 的 Nacos Env 引用：无需鉴权就删掉它们；需要鉴权就先删掉旧引用，再通过“密码与凭据”添加同名 Env，预览确认后保存。不能同时保留旧 Env 和配套配置中的同名 Env，冲突校验会阻止保存。不同环境分别选择凭据，不同步密码正文。

以后点击该组“编辑键值与引用”，只填写要修改的密码，账号留空保留原值；预览、保存并发布当前环境。其他服务和环境不会跟随变化。不要将真实账号密码粘贴到公共模板或 Git 仓库。

## 镜像启动要求

Kubernetes 模板负责注入环境变量，不负责重新编译 JAR。镜像的启动入口需要读取 JAVA_OPTS，例如：

```dockerfile
ENTRYPOINT ["sh", "-c", "exec java $JAVA_MODULE_OPTS $JAVA_OPTS -jar app.jar"]
```

如果镜像使用 `ENTRYPOINT ["java", "-jar", "app.jar"]`，JAVA_OPTS 不会自动展开，需要先调整镜像入口。模板不覆盖已有启动命令。

Spring 配置通过标准环境变量注入，SERVERADDR、FILEEXTENSION、REFRESHENABLED 对应属性中去除连字符后的名称。应用必须包含匹配版本的 Spring Cloud Alibaba / Nacos 依赖；`optional:nacos:` 的 Data ID 和导入语法按实际依赖版本核对。平台模板与 YAML 渲染已验证，具体应用连接 Nacos 仍需部署后验证。

## 编辑 YAML

- `Tab`：插入空格缩进；选中多行时整段缩进，不删除内容。
- `Shift+Tab`：减少缩进。
- `Enter`：保持当前行缩进，在映射键或 `value: >-` 后补下一层缩进。
- 工具栏格式化图标、`Shift+Alt+F` 或 `Ctrl+Shift+F`：格式化 YAML，保留 AxiOps 变量、注释和文件标记。
- `Ctrl+Z`：撤销缩进、输入和格式化。
- `Ctrl+S`：调用当前配置的保存操作，仍执行原有校验。

“语法校验”在编辑、保存后和只读查看状态都可以使用，无需先解锁。错误会提示行号并定位编辑区，原文保留；修正后再次校验即可。点击保存也会自动检查，后端再次验证公共服务版本的 YAML 结构，错误时不保存新版本。

公共基础配置中留给环境填写的变量，可以通过结构校验并显示“待目标环境提供变量”；选定具体环境后会提示缺少的值。校验检查 YAML 结构与已有变量，不等于集群部署成功，实际发布仍检查环境、命名空间、渲染值、资源和就绪状态。

这是现有轻量编辑器的增强，不是完整 VS Code。复杂 Helm 模板不使用这个格式化器。语法错误时格式化会提示错误并保留原文，不猜测字段归属。

你原文的 JAVA_OPTS 多行内容必须缩进在 `value: >-` 内，例如：

```yaml
          env:
            - name: JAVA_OPTS
              value: >-
                -XX:+UseContainerSupport
                -XX:MaxRAMPercentage=72.0
                -XX:MaxMetaspaceSize=256m
                -XX:+UseG1GC
                -Dfile.encoding=UTF-8
```

不能把 `-XX:` 参数退到 env 或 Deployment 外层，格式化器也无法从这种错误缩进恢复你的原意。
