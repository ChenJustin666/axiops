# 仓库与模块路径迁移

目标：`https://github.com/ChenJustin666/axiops.git`。

将 Go 模块和内部引用从 `github.com/yanshicheng/kube-nova` 改为
`github.com/ChenJustin666/axiops`，同步 Makefile、代码生成模板、`.proto` 的
`go_package` 和已生成的 Protobuf 描述信息。RPC 协议包名、消息结构和字段编号保持不变。
没有调整业务资源名称、Namespace、存储桶、数据库结构或业务数据。

前端完整源码以普通目录 `axiops-web/` 收录，无需额外初始化子模块。
旧后端及前端 Git 历史仅在本地保留，不推送旧历史中的凭据。首版提交保留上游许可证及版权声明。
本地配置保留；Git 只保存示例配置，真实私钥、数据库导出、缓存、二进制、参考工程均不入库。

前端已有项目页增加存储不可用时的降级，避免受限浏览器无法打开项目列表。
两份测试的工具模块依赖加载已补齐。

验证：

- `go mod tidy -diff`、`go test -p 3 ./...`、`go build -p 3 ./...` 通过。
- 配置隔离测试数据库后，appsvc、projectenv、helmconfig、templatelib、project、workflow 六包集成回归通过。
- 固定快照前端测试 346 项通过。
- 前端 Vite 正式构建通过（4 分 21 秒），保留既有 resource-center CSS 警告。
- Linux/amd64、CGO 关闭的 `cmd/...` 和 `application/...` 服务构建通过。
- 六份 Protobuf 描述与迁移前比较，除 Go 包路径外完全一致。
- Gitleaks 扫描无泄漏；实际数据库密码、私钥和业务数据库导出未提交。
- 配置示例已执行 YAML 语法检查；修复已有 portal 示例中未闭合的产品名称字符串。
- 其他编辑器同时新增的监控数据源源码、配置及迁移 053/054 尚未交付，保留在本地，不包含在本次提交。
  本次迁移继续使用已部署的迁移 052，无需为模块改名执行新数据库 DDL。
- 文档中的 Go 片段使用 `.go.txt` 后缀，避免被当作可编译包。

- 使用原有配置和凭据加密密钥，在隔离端口 18091 启动迁移后的 CI/CD API，健康检查 code=0；
  日志显示 current=052、applied=[]，没有执行新数据库迁移。前端开发首页 HTTP 200。

全仓 Vue 类型检查存在此前记录的既有诊断，
本次不宣称全仓类型检查通过。源码构建与测试不能替代实际应用镜像在目标集群的发布验收。
