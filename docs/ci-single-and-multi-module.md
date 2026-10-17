# 单模块和多模块的构建入口选择

服务的 Dockerfile 位置与构建上下文是两件事。Dockerfile 决定如何构建；上下文决定它的 `COPY` / `ADD` 可以读取哪些源码。子模块有自己的 Dockerfile，不代表必须把上下文缩小到子模块。

## 页面操作

项目工作区 → 服务 → 构建配置 → 选择代码源、仓库和分支 → **从仓库选择** Dockerfile。浏览器按目录加载当前分支的文件，支持进入目录、返回上级及读取失败后重试。选择文件后默认使用整个仓库，并展示实际文件位置和源码范围。

| 服务 | 选择的 Dockerfile（仓库路径） | 默认构建上下文 | 等价的 Docker CLI 参数 |
| --- | --- | --- | --- |
| risk_server | `Dockerfile` | 仓库根 `.` | `-f Dockerfile .` |
| CardScout gateway | `card-scout-gateway/Dockerfile` | 仓库根 `.` | `-f card-scout-gateway/Dockerfile .` |
| CardScout server | `card-scout-server/Dockerfile` | 仓库根 `.` | `-f card-scout-server/Dockerfile .` |

gateway 和 server 各保存自己的构建配置、镜像和部署定义；common / business 通常作为构建依赖参与，不需要仅因存在目录就创建可部署服务。平台不会猜测多个 Dockerfile 中哪一个应该替用户发布。

已有配置打开后保持原样。高级路径设置保留自定义目录及文件名；当前后端存储的 Dockerfile 路径仍相对于上下文。页面负责将仓库路径换算为这两个字段，切换上下文不会偷偷换成另一个文件。例如独立模块保存 `build_context=card-scout-gateway`、`dockerfile_path=Dockerfile`；页面仍展示 `card-scout-gateway/Dockerfile`。

## 多阶段 Dockerfile：在 Dockerfile 中编译

CardScout 需要父 POM 和 reactor 依赖，使用仓库根上下文，Dockerfile 放在各服务目录即可。例如 gateway 的编译阶段按以下结构组织：

```dockerfile
# 以下是编译阶段片段，基础镜像、Maven settings 和最终 JAR 选择沿用项目配置。
WORKDIR /build
COPY . .
RUN mvn -B -ntp -pl card-scout-gateway -am clean package
```

`-pl` 选择待构建模块，`-am` 连同其 reactor 依赖构建。server 使用 `-pl card-scout-server -am`；最终运行镜像取对应模块的可执行 JAR。父 POM 必须声明模块，相关 profile、私有 Maven 仓库及基础镜像凭据仍需符合项目实际情况。

`COPY . .` 是表达完整 reactor 输入的简单方式；项目已有精确 COPY / 缓存分层可以继续保留，但必须包含父 POM、目标模块及其依赖的 POM 和源码。注意 `.dockerignore` 不要排除这些输入。

平台只执行所选 Dockerfile，不会仅根据文件目录自动改写其中的 Maven 命令、COPY 或 JAR 名称。这些内容属于项目的版本化构建规则。

## JAR 镜像：先编译，再用运行时 Dockerfile 打包

选择 JAR 镜像 / Maven 后，在仓库浏览器选择模块目录中的 Dockerfile；该目录存在 `pom.xml` 时，提供 **使用此模块的 Maven reactor 编译命令**。点击才替换上方命令，普通文件选择不会覆盖已编辑的命令。例如：

```sh
mvn -B -ntp -Dmaven.repo.local="${CACHE_DIR:-/workspace/.cache/maven}" -pl 'card-scout-gateway' -am clean package
```

编译命令始终在完整仓库根目录执行，默认运行测试。此时构建上下文只影响后续镜像打包，可以根据运行时 Dockerfile 的写法选择：

| Dockerfile 的 JAR 源路径写法 | 打包上下文 |
| --- | --- |
| `COPY card-scout-gateway/target/具体文件.jar app.jar` | 整个仓库 `.` |
| `COPY target/具体文件.jar app.jar`（Dockerfile 位于 gateway 目录） | 仅 Dockerfile 所在目录 `card-scout-gateway` |

不能只看“多模块”就强制这一步必须使用仓库根上下文：JAR 已在前一步完成 reactor 编译，运行时镜像可以只读取目标模块。反过来，多阶段 Dockerfile 自己还要编译 reactor 时，就必须拿到父目录和依赖源码。

## Jenkinsfile 与部署文件

内置 BuildKit 模式使用 Dockerfile 和 AxiOps 工作流；仓库中存在 Jenkinsfile 不会自动执行其中的步骤。需要 Jenkins Pipeline 语义时应使用对应执行器，不能把两套执行规则混在一起。

`deploy/gateway/values-test.yaml`、`deploy/server/values-test.yaml` 属于各服务的部署定义/环境配置，与镜像构建上下文分别管理。选择 Dockerfile 不会自动改变部署文件、环境或生产命名空间。

## 验证边界

- 当前选择器基于仓库目录与文件元数据，不解析完整 Maven 依赖图；同目录有 `pom.xml` 只用于提供可选 reactor 命令，不能证明父 POM 已声明该模块。
- 已读取真实 `server/CardScout` 的 master 根目录及 gateway/server 子目录，确认对应 Dockerfile、Jenkinsfile 和 pom.xml 均可列出。未修改远端仓库或业务构建配置。
- 组件行为回归覆盖单模块、多模块、上下文换算、已有配置保留、切换分支时丢弃旧响应、失败重试及显式 Maven 命令生成。真实业务 Maven 编译/镜像推送需在用户选定配置后另行验证。
