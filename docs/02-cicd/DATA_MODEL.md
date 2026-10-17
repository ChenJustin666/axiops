# CI/CD 目标数据模型

## 1. 迁移原则

现有 `project`、`app_service`、`environment`、`build_run` 和 `release_record` 是早期骨架，不能直接删除。新增迁移先引入目标字段和表，旧项目默认标记为 `unclassified`，在管理员完成类型确认前禁止新部署；不猜测旧项目属于 YAML、Helm 或主机类型。

## 2. 核心实体

| 实体 | 关键字段 | 说明 |
| --- | --- | --- |
| `project` 扩展 | `delivery_type`, `delivery_status` | 类型为 `k8s_yaml/helm/host/unclassified`；类型确认后不可直接修改 |
| `app_service` 扩展 | `delivery_type`, `source_type`, `source_ref`, `current_revision_id`, `component_count` | 服务类型与项目强一致；保留旧 `deploy_type` 仅作兼容读取 |
| `service_component` | `service_id`, `name`, `component_type`, `image_ref_path`, `health_profile_id` | 一个服务可解析多个工作负载/组件 |
| `service_revision` | `service_id`, `revision`, `source_commit`, `definition`, `content_hash` | 不可变的 YAML、Chart 或主机定义快照 |
| `service_helm_spec` | `service_id`, `chart_source`, `chart_path`, `values_path`, `release_name_pattern` | 仅 Helm 项目使用 |
| `service_host_spec` | `service_id`, `artifact_rule`, `deploy_script_ref`, `start_script_ref`, `stop_script_ref`, `health_profile_id` | 仅主机项目使用 |
| `project_environment` | `project_id`, `name`, `environment_type`, `cluster_id`, `namespace`, `host_group_ref`, `binding_status`, `is_production` | 项目环境上下文；不复用全局环境名称表达资源绑定 |
| `variable_set` / `variable_item` | `scope`, `project_id`, `environment_id`, `service_id`, `variable_key`, `credential_id`, `is_secret` | 分层变量；Secret 只引用凭据 |
| `environment_service_override` | `project_environment_id`, `service_id`, `revision_id`, `variables`, `replicas` | 环境内服务覆盖与目标版本 |
| `workflow` / `workflow_node` | `project_id`, `definition_version`, `node_type`, `depends_on`, `timeout`, `retry` | 声明式 DAG 与节点定义 |
| `workflow_run` / `workflow_task_run` | `workflow_id`, `definition_version`, `trigger`, `operation_type`, `source_snapshot_id`, `request_key`, `status`, `context_snapshot`, `node_snapshot`, `node_status` | 普通工作流与系统回滚的运行审计、幂等键、节点状态和恢复依据（迁移 `013`、`017`） |
| `artifact` | `build_run_id`, `kind`, `uri`, `digest`, `checksum`, `metadata` | 镜像和主机制品统一登记 |
| `deployment_snapshot` | `workflow_run_id`, `workflow_task_run_id`, `node_key`, `project_id`, `environment_id`, `service_id`, `service_revision_id`, `cluster_id`, `namespace`, `provider`, `manifest_digest`, `rendered_manifest`, `values_snapshot`, `helm_release_name`, `helm_revision`, `status` | 精确到运行节点和目标的部署输出；YAML 保存受控清单，Helm 保存脱敏资源身份与 release revision |
| `helm_execution` | `idempotency_key`, `request_hash`, `operation`, `cluster_id`, `namespace`, `release_name`, `target_revision`, `status`, `helm_revision` | 官方 Helm SDK 发布/回滚的持久化幂等和崩溃恢复记录 |
| `source_snapshot` | `build_run_id`, `code_source_id`, `repo_path`, `branch`, `commit_sha`, `commit_url`, `commit_message`, `author`, `changed_files`, `snapshot_hash` | 构建使用的不可变源码元数据和代码变更摘要 |
| `build_config_snapshot` | `build_run_id`, `template_version`, `runner_pool`, `build_image`, `command`, `variables`, `resources`, `cache_summary`, `script_hashes` | 构建实际使用配置，Secret 仅保存凭据引用 |
| `artifact_promotion` | `artifact_id`, `source_environment`, `target_environment`, `gate_summary`, `approved_by`, `request_key`, `created_at` | 同一 digest/checksum 在环境间晋级的不可变记录 |
| `cache_entry` | `scope`, `project_id`, `service_id`, `cache_key`, `backend`, `size`, `hit_count`, `expires_at`, `content_hash` | 构建缓存索引和治理信息；缓存不可承载 Secret |
| `deployment_target` | `environment_id`, `cluster_id`, `namespace`, `target_order`, `strategy`, `status` | 一个逻辑环境下的具体集群/Namespace 发布目标 |

## 3. 类型约束

```text
k8s_yaml: service_revision(manifest) + project_environment(cluster, namespace)
helm:     service_helm_spec + project_environment(cluster, namespace, release name)
host:     service_host_spec + project_environment(host_group)
```

数据库和 API 均需校验：YAML/Helm 环境不能绑定主机组，主机环境不能绑定 Namespace；Helm 节点不能运行在 YAML 项目；主机部署任务必须由已认证 Agent 主动拉取。

## 4. 不可变快照

构建配置、服务定义、变量解析结果、工作流定义、制品 digest 和部署输出都以 revision/snapshot 保存。运行只引用这些不可变快照，配置更新不会改变已发起或已完成的运行记录。

YAML 部署在调用 provider 前写入 `prepared` 快照，apply 成功后更新为 `applied`。回滚只能读取
`applied` 状态，并在执行前重新计算 `rendered_manifest` 的 SHA-256 与 `manifest_digest` 比对；客户端不能
上传替代清单。Secret 资源在渲染校验阶段即被拒绝，因此不会进入部署快照。

## 5. 兼容迁移

1. 已由 `migrations/007_cicd_project_delivery.sql` 为旧项目增加
   `delivery_type=unclassified`、`delivery_status=pending_confirmation` 和查询索引。
2. 管理员在项目设置中使用 `PUT /api/v1/projects/{id}/delivery-profile` 确认类型，并选择或导入服务定义。
3. 旧 `app_service.deploy_type` 只能在人工盘点时作为候选信息，系统不得自动写入项目类型。
4. 新目标 API 完成读写切换、历史运行可查询、回滚演练后，再单独评估旧字段/表下线。

`migrations/008_cicd_service_revision.sql` 已实现服务扩展和不可变 revision 表。revision 编号通过
`service_revision_sequence` 在事务内分配，避免并发写入使用 `MAX(revision_number)+1` 产生重复编号；
同一事务内写入 revision 并更新服务的 `current_revision_id`。

`migrations/009_cicd_project_environment.sql` 已实现 `project_environment`、`variable_set` 和
`variable_item`。创建环境时在同一事务创建默认环境变量组；变量更新通过 revision 乐观锁防止覆盖，
Secret 变量只保存 `credential_id`，不保存明文。

当前主机资产仍来自旧 Mongo `devops_host`，没有统一的 MySQL 主机组授权实体。因此主机环境只保存
外部 `host_group_ref` 并标记 `pending_validation`；后续主机组注册中心完成校验后才能转为
`validated` 并参与部署。

`migrations/010_cicd_environment_service_override.sql` 已实现环境内服务目标状态。记录只引用不可变
`service_revision_id`，可选覆盖镜像和副本数，并通过 revision 乐观锁更新；环境、服务和 revision
必须在 API 层验证同一项目和类型兼容性。
