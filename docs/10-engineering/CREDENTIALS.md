# 凭据存储约束

凭据 `credential.data` 从本阶段起必须使用 `v1:` AES-GCM 密文。密钥通过
`PlatformAPIConfig.CredentialEncryptionKey` 注入，不能写入数据库、代码仓库或日志。

- 创建和更新接口只接受合法 JSON，并在写入前加密。
- 密文包含随机 nonce；同一明文每次加密结果不同。
- 未配置加密密钥时，创建和更新直接失败，不降级为明文。
- `platform-api` 未配置加密密钥时拒绝启动，避免错误配置下继续提供明文凭据能力。
- 修改旧的明文记录时，如果没有同时提供新凭据数据，接口拒绝保存，避免继续扩散明文。
- 生产环境应使用 Secret Manager/KMS 注入密钥，并安排密钥轮换和旧密文迁移窗口。
- `cluster.kubeconfig` 使用相同的 `v1:` AES-GCM 格式；集群创建/更新先验证 kubeconfig，再加密写入。
- `axiops-workflow-worker` 必须使用与 `platform-api` 相同的密钥，只在内存中短时解密目标集群配置。
- 历史明文 kubeconfig 不会自动降级读取；管理员必须通过集群更新接口重新提交后才能用于新 worker。
- 构建仓库认证只读取 `code_source.credential_id` 指向的统一 `git_token` 密文。worker 创建短期 Kubernetes Secret，Job 只保存 `secretKeyRef`，终态或取消后删除。
- `build_run.config_snapshot` 只冻结 `code_source_id` 和非敏感仓库 URL，不保存 Token；Pod 日志只采集 builder 容器，避免输出 git clone 环境。

当前仍需补充：build/host worker 通过受控的 credential reference 读取并短时注入 Job/Agent，不能把
解密后的内容放入 `build_run.config_snapshot`、命令队列或构建日志。
