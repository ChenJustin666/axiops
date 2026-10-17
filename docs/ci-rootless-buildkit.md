# Rootless BuildKit 节点要求与 Build #8 排查

2026-09-25 已在 CI 集群 2 的节点 `us-east-1.172.50.0.18` 上只读检查确认：

- `user.max_user_namespaces=0`
- `kernel.unprivileged_userns_clone=1`
- `DiskPressure=False`、`MemoryPressure=False`、`PIDPressure=False`，节点 Ready。
- 通过该节点已有容器查看到根文件系统剩余约 9.6 GiB，inode 使用约 22%。

Build #8 已完成 git-clone，失败于 RootlessKit 创建用户命名空间。`fork/exec /proc/self/exe: no space left on device` 对应的 ENOSPC 在这里指命名空间配额为零，并非磁盘空间耗尽。`BackoffLimitExceeded` 是 Job 的最终状态，不是根因。增加构建内存、改镜像仓库密码、重启 AxiOps 都不能修复节点限制。

## 节点修复（需节点管理员确认范围后执行）

在**允许运行构建 Job 的 CI 节点**上检查并保存原值：

```sh
sysctl user.max_user_namespaces kernel.unprivileged_userns_clone
df -h / /var/lib/containerd
df -i / /var/lib/containerd
```

该节点原值为 0。可先临时设置一个明确的非零上限，例如 RootlessKit 常用的 28633，然后验证非特权用户可以创建命名空间：

```sh
sudo sysctl -w user.max_user_namespaces=28633
# 使用已有的非 root CI 账号执行，下面的 <ci-user> 需替换成真实账号。
sudo -u <ci-user> unshare -Ur true
```

验证通过后，把 `user.max_user_namespaces = 28633` 加入节点管理员维护的 `/etc/sysctl.d/90-axiops-buildkit.conf`，执行 `sudo sysctl --system` 并确认最终值。先检查已有 sysctl 配置，避免覆盖其他配置或被后加载的配置重新置零。该改动影响整个节点的非特权用户命名空间能力，应纳入节点池基线；使用 ACK 时还需同步节点池初始化配置，避免新节点/替换节点恢复为 0。

此操作通常不需要重启节点。原失败 Job 已终止，不会自动恢复；修复后手动创建新的构建，保留旧记录用于对比。若有多个可调度 CI 节点，需要逐个确认，否则重试可能落到仍禁用命名空间的节点。回滚到原值 0 会再次阻止新的 RootlessKit 构建。

如果节点安全基线不允许用户命名空间，应使用允许此能力的专用 CI 节点池或单独评估其他构建执行器；不要在业务 Job 中自动修改宿主机 sysctl。

## 本仓库已补充的诊断

- builder 在启动前读取命名空间开关，值为 0 时立即报 `ROOTLESS_USER_NAMESPACE_DISABLED`。
- 检测 BuildKit 进程提前退出，立即展示 daemon 日志，不再固定等 30 秒后笼统报超时。
- 启动错误写入 termination-log；worker 将明确的启动错误合并到构建失败原因。
- `tools/ci-node-check` 复用平台加密保存的集群连接，只读检查节点和指定已有容器，不落盘 kubeconfig，不创建 Pod 或改内核参数。

```powershell
go run ./tools/ci-node-check -node us-east-1.172.50.0.18
# 根据上一步显示的仍在运行的 Pod 选择检查目标：
go run ./tools/ci-node-check -node us-east-1.172.50.0.18 -namespace kube-system -pod <existing-pod> -container <container>
```

builder 的代码诊断需构建并发布新的平台 builder 镜像后才会在远端生效；仅更新网页不会更新已有 Pod。当前线上 builder 为 `platform/builder:v1.0.1`。节点修复本身也适用于该旧版本。
